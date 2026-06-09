package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// MiniAppMe returns the authenticated Telegram Mini App user's identity.
func (a *API) MiniAppMe(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return c.JSON(miniappUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role})
}
