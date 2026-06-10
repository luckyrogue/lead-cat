package middleware

import (
	"context"
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// WebUserResolver resolves a raw session cookie value to the authenticated
// web account, sliding the session as needed.
type WebUserResolver interface {
	ResolveWebUser(ctx context.Context, rawSessionToken string) (postgres.PlatformUser, bool, error)
}

// WebAuth guards web routes via the lc_session cookie and enforces CSRF
// (double-submit) on state-changing methods.
type WebAuth struct{ resolver WebUserResolver }

func NewWebAuth(r WebUserResolver) *WebAuth { return &WebAuth{resolver: r} }

func (m *WebAuth) Middleware(c *fiber.Ctx) error {
	raw := c.Cookies("lc_session")
	if raw == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	user, ok, err := m.resolver.ResolveWebUser(c.UserContext(), raw)
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		if !csrfMatches(c.Get("X-CSRF-Token"), c.Cookies("lc_csrf")) {
			return fiber.NewError(fiber.StatusForbidden, "csrf")
		}
	}
	c.Locals("web_user", user)
	return c.Next()
}

// csrfMatches does a constant-time compare; empty values never match.
func csrfMatches(header, cookie string) bool {
	if header == "" || cookie == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(cookie)) == 1
}
