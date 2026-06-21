package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders sets baseline security response headers on every API response.
// HSTS is emitted only in production (it requires TLS to be meaningful); auth
func SecurityHeaders(prod bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if prod {
			c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		if strings.HasPrefix(c.Path(), "/api/auth/") {
			c.Set("Cache-Control", "no-store")
		}
		return c.Next()
	}
}
