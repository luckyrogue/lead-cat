package handlers

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/auth/oauth"
	webauthnsvc "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/auth/webauthn"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AuthAPI struct {
	Store    *postgres.Store
	JWT      *platformauth.JWT
	OTP      *platformauth.OTP
	WebAuthn *webauthnsvc.Service
	OAuth    *oauth.Service
	Webapp   string
}

func (h *AuthAPI) tokenResponse(c *fiber.Ctx, u postgres.User) error {
	tok, err := h.JWT.Issue(u.ID, u.AuthSub, u.Email, u.Phone)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "token issue failed")
	}
	return c.JSON(fiber.Map{
		"access_token": tok,
		"token_type":   "Bearer",
		"user": fiber.Map{
			"id":    u.ID,
			"email": u.Email,
			"phone": u.Phone,
		},
	})
}

func (h *AuthAPI) Config(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"github_enabled":   h.OAuth != nil && h.OAuth.GitHubEnabled(),
		"gitlab_enabled":   h.OAuth != nil && h.OAuth.GitLabEnabled(),
		"passkey_enabled":  h.WebAuthn != nil,
		"email_enabled":    true,
		"phone_enabled":    true,
	})
}

func (h *AuthAPI) SendEmailCode(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil || body.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email required")
	}
	if _, err := h.OTP.Send(c.Context(), "email", body.Email); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "send failed")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthAPI) VerifyEmailCode(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || body.Email == "" || body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and code required")
	}
	ok, err := h.OTP.Verify(c.Context(), "email", body.Email, body.Code)
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid code")
	}
	sub := platformauth.SubEmail(body.Email)
	u, err := h.Store.UpsertUserIdentity(c.Context(), sub, body.Email, "")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "user create failed")
	}
	return h.tokenResponse(c, u)
}

func (h *AuthAPI) SendPhoneCode(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&body); err != nil || body.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone required")
	}
	if _, err := h.OTP.Send(c.Context(), "phone", body.Phone); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "send failed")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthAPI) VerifyPhoneCode(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || body.Phone == "" || body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone and code required")
	}
	ok, err := h.OTP.Verify(c.Context(), "phone", body.Phone, body.Code)
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid code")
	}
	phoneNorm := strings.TrimPrefix(platformauth.SubPhone(body.Phone), "phone:")
	u, err := h.Store.UpsertUserIdentity(c.Context(), platformauth.SubPhone(body.Phone), "", phoneNorm)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "user create failed")
	}
	return h.tokenResponse(c, u)
}

func (h *AuthAPI) PasskeyLoginBegin(c *fiber.Ctx) error {
	if h.WebAuthn == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "passkey not configured")
	}
	opts, sid, err := h.WebAuthn.BeginLogin(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"options": opts, "session_id": sid})
}

func (h *AuthAPI) PasskeyLoginFinish(c *fiber.Ctx) error {
	var body struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := c.BodyParser(&body); err != nil || body.SessionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "session_id and credential required")
	}
	u, err := h.WebAuthn.FinishLogin(c.Context(), body.SessionID, body.Credential)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	return h.tokenResponse(c, u)
}

func (h *AuthAPI) PasskeyRegisterBegin(c *fiber.Ctx) error {
	uid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "login required")
	}
	opts, sid, err := h.WebAuthn.BeginRegistration(c.Context(), uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"options": opts, "session_id": sid})
}

func (h *AuthAPI) PasskeyRegisterFinish(c *fiber.Ctx) error {
	uid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "login required")
	}
	var body struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
		DeviceName string          `json:"device_name"`
	}
	if err := c.BodyParser(&body); err != nil || body.SessionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "session_id and credential required")
	}
	if err := h.WebAuthn.FinishRegistration(c.Context(), uid, body.SessionID, body.Credential, body.DeviceName); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AuthAPI) OAuthStart(c *fiber.Ctx) error {
	provider := c.Params("provider")
	urlStr, _, err := h.OAuth.AuthURL(c.Context(), provider)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Redirect(urlStr, fiber.StatusFound)
}

func (h *AuthAPI) OAuthCallback(c *fiber.Ctx) error {
	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		return h.oauthRedirect(c, "", "missing_params")
	}
	u, err := h.OAuth.Exchange(c.Context(), state, code)
	if err != nil {
		return h.oauthRedirect(c, "", "oauth_failed")
	}
	tok, err := h.JWT.Issue(u.ID, u.AuthSub, u.Email, u.Phone)
	if err != nil {
		return h.oauthRedirect(c, "", "token_failed")
	}
	return h.oauthRedirect(c, tok, "")
}

func (h *AuthAPI) oauthRedirect(c *fiber.Ctx, token, errCode string) error {
	base := strings.TrimSuffix(h.Webapp, "/")
	u, _ := url.Parse(base + "/login")
	q := u.Query()
	if token != "" {
		q.Set("access_token", token)
	}
	if errCode != "" {
		q.Set("error", errCode)
	}
	u.RawQuery = q.Encode()
	return c.Redirect(u.String(), fiber.StatusFound)
}
