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

func IsLoopback(ip string) bool {
	ip = strings.TrimSpace(ip)
	return ip == "127.0.0.1" || ip == "::1" || ip == "0:0:0:0:0:0:0:1"
}
