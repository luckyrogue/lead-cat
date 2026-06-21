package httpclient

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ClientIP(c *fiber.Ctx, trustProxy bool) string {
	if trustProxy {
		if xri := strings.TrimSpace(c.Get("X-Real-IP")); xri != "" {
			return xri
		}
	}
	return c.IP()
}
