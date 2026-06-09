// Package middleware contains Fiber-level middleware shared across handlers.
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// RequireBotAdmin asserts that the request was authenticated as a bot user
// whose role is "admin". Returns 403 otherwise. Must be mounted AFTER the TMA
// JWT middleware that sets c.Locals("bot_user").
func RequireBotAdmin(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok || bu.Role != "admin" {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	return c.Next()
}
