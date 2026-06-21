package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func RequireBotAdmin(adminTelegramIDs []int64) fiber.Handler {
	allowed := make(map[int64]struct{}, len(adminTelegramIDs))
	for _, id := range adminTelegramIDs {
		allowed[id] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		bu, ok := c.Locals("bot_user").(model.BotUser)
		if !ok || bu.Role != "admin" {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		if len(allowed) > 0 {
			if _, ok := allowed[bu.TelegramID]; !ok {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
		}
		return c.Next()
	}
}
