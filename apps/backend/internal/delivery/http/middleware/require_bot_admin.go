package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres" //nolint:depguard
)

func RequireBotAdmin(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok || bu.Role != "admin" {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	return c.Next()
}
