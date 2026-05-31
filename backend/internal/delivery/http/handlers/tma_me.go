package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// TMAMe returns the authenticated Telegram Mini App user's identity.
func (a *API) TMAMe(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return c.JSON(tmaUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role})
}
