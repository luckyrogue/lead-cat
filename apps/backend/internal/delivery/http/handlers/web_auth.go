package handlers

import (
	"context"
	"crypto/subtle"
	"net/mail"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

// cookieSecure derives the cookie Secure flag from the configured base URL.
func (a *API) cookieSecure() bool {
	return strings.HasPrefix(a.App.AppBaseURL(), "https")
}

// setSessionCookies sets the long-lived lc_session (HTTPOnly) and lc_csrf
// (JS-readable, for double-submit) cookies.
func (a *API) setSessionCookies(c *fiber.Ctx, session, csrf string) {
	secure := a.cookieSecure()
	exp := time.Now().Add(30 * 24 * time.Hour)
	c.Cookie(&fiber.Cookie{
		Name:     "lc_session",
		Value:    session,
		Path:     "/",
		Domain:   a.WebCookieDomain,
		Expires:  exp,
		Secure:   secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "lc_csrf",
		Value:    csrf,
		Path:     "/",
		Domain:   a.WebCookieDomain,
		Expires:  exp,
		Secure:   secure,
		HTTPOnly: false,
		SameSite: "Lax",
	})
}

// clearSessionCookies expires the session and csrf cookies.
func (a *API) clearSessionCookies(c *fiber.Ctx) {
	a.clearCookie(c, "lc_session", true)
	a.clearCookie(c, "lc_csrf", false)
}

// setShortCookie sets a short-lived HTTPOnly cookie (oauth state / pkce).
func (a *API) setShortCookie(c *fiber.Ctx, name, value string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   600,
		Secure:   a.cookieSecure(),
		HTTPOnly: true,
		SameSite: "Lax",
	})
}

// clearCookie expires a cookie by name. httpOnly must match how it was set.
func (a *API) clearCookie(c *fiber.Ctx, name string, httpOnly bool) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   a.WebCookieDomain,
		Expires:  time.Now().Add(-time.Hour),
		MaxAge:   -1,
		Secure:   a.cookieSecure(),
		HTTPOnly: httpOnly,
		SameSite: "Lax",
	})
}

// WebAuthStart begins an SSO authorization-code + PKCE flow.
func (a *API) WebAuthStart(c *fiber.Ctx) error {
	provider := c.Params("provider")
	p, ok := a.App.SSOProviderByName(provider)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown_provider")
	}
	state, err := authweb.NewState(nil)
	if err != nil {
		return err
	}
	verifier, challenge, err := authweb.NewPKCE(nil)
	if err != nil {
		return err
	}
	a.setShortCookie(c, "lc_oauth_state", state)
	a.setShortCookie(c, "lc_pkce", verifier)
	redirectURL := a.App.AppBaseURL() + "/api/auth/web/" + provider + "/callback"
	return c.Redirect(p.AuthURL(state, challenge, redirectURL), fiber.StatusFound)
}

// WebAuthCallback completes the SSO flow: validates state, exchanges the code,
// upserts the identity, and issues a session.
func (a *API) WebAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	p, ok := a.App.SSOProviderByName(provider)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown_provider")
	}
	ctx := c.UserContext()

	state := c.Query("state")
	cookieState := c.Cookies("lc_oauth_state")
	if state == "" || cookieState == "" ||
		subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
		return fiber.NewError(fiber.StatusBadRequest, "bad_state")
	}
	verifier := c.Cookies("lc_pkce")
	code := c.Query("code")
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing_code")
	}
	redirectURL := a.App.AppBaseURL() + "/api/auth/web/" + provider + "/callback"

	profile, err := p.Exchange(ctx, code, verifier, redirectURL)
	if err != nil {
		a.Log.Warn("web_auth_exchange_failed", zap.String("provider", provider), zap.Error(err))
		return fiber.NewError(fiber.StatusBadRequest, "exchange_failed")
	}
	if _, perr := mail.ParseAddress(profile.Email); profile.Email == "" || perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_email")
	}

	user, err := a.App.UpsertWebIdentity(ctx, profile.Email, profile.Name, profile.AvatarURL, provider)
	if err != nil {
		a.Log.Error("web_upsert_identity_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "upsert_failed")
	}
	if _, err := a.App.AcceptInvitesForEmail(ctx, profile.Email, user.ID); err != nil {
		a.Log.Warn("web_accept_invites_failed", zap.Error(err))
	}

	raw, err := a.App.CreateWebSession(ctx, user.ID, c.Get("User-Agent"), c.IP())
	if err != nil {
		a.Log.Error("web_create_session_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "session_failed")
	}
	csrf, err := authweb.NewState(nil)
	if err != nil {
		return err
	}
	a.setSessionCookies(c, raw, csrf)
	a.clearCookie(c, "lc_oauth_state", true)
	a.clearCookie(c, "lc_pkce", true)

	return c.Redirect(a.App.AppBaseURL()+a.postLoginDest(ctx, user.ID), fiber.StatusFound)
}

// WebMagicRequest issues a one-time sign-in link. Always returns 204 to avoid
// account enumeration.
func (a *API) WebMagicRequest(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(fiber.StatusNoContent)
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if err := a.App.RequestMagicLink(c.UserContext(), email); err != nil {
		a.Log.Warn("web_magic_request_failed", zap.Error(err))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// WebMagicVerify consumes a magic-link token and issues a session.
func (a *API) WebMagicVerify(c *fiber.Ctx) error {
	ctx := c.UserContext()
	token := c.Query("token")
	email, err := a.App.VerifyMagicLink(ctx, token)
	if err != nil {
		return c.Redirect(a.App.AppBaseURL()+"/login?error=invalid_link", fiber.StatusFound)
	}
	user, err := a.App.UpsertWebIdentity(ctx, email, "", "", application.AuthMethodMagicLink)
	if err != nil {
		a.Log.Error("web_upsert_identity_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "upsert_failed")
	}
	if _, err := a.App.AcceptInvitesForEmail(ctx, email, user.ID); err != nil {
		a.Log.Warn("web_accept_invites_failed", zap.Error(err))
	}
	raw, err := a.App.CreateWebSession(ctx, user.ID, c.Get("User-Agent"), c.IP())
	if err != nil {
		a.Log.Error("web_create_session_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "session_failed")
	}
	csrf, err := authweb.NewState(nil)
	if err != nil {
		return err
	}
	a.setSessionCookies(c, raw, csrf)
	return c.Redirect(a.App.AppBaseURL()+a.postLoginDest(ctx, user.ID), fiber.StatusFound)
}

// WebLogout revokes the current session and clears cookies.
func (a *API) WebLogout(c *fiber.Ctx) error {
	raw := c.Cookies("lc_session")
	if err := a.App.RevokeWebSession(c.UserContext(), raw); err != nil {
		a.Log.Warn("web_revoke_session_failed", zap.Error(err))
	}
	a.clearSessionCookies(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// WebMe returns the authenticated web account and its organizations.
func (a *API) WebMe(c *fiber.Ctx) error {
	user := c.Locals("web_user").(postgres.PlatformUser)
	orgs, err := a.App.ListOrganizationsForUser(c.UserContext(), user.ID)
	if err != nil {
		a.Log.Error("web_list_organizations_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_organizations_failed")
	}
	out := make([]fiber.Map, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, fiber.Map{
			"id":   o.ID,
			"name": o.Name,
			"slug": o.Slug,
		})
	}
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"email":       user.Email,
			"avatar_url":  user.AvatarURL,
			"auth_method": user.AuthMethod,
		},
		"organizations": out,
	})
}

// postLoginDest decides where to send the user after sign-in: onboarding when
// they belong to no organization, otherwise the app root.
func (a *API) postLoginDest(ctx context.Context, userID uuid.UUID) string {
	orgs, _ := a.App.ListOrganizationsForUser(ctx, userID)
	if len(orgs) == 0 {
		return "/onboarding"
	}
	return "/"
}
