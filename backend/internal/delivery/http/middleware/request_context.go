package middleware

import (
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
	"github.com/gofiber/fiber/v2"
)

func RequestContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid, _ := c.Locals("requestid").(string)
		ctx := log.WithRequestID(c.UserContext(), rid)
		c.SetUserContext(ctx)
		return c.Next()
	}
}
