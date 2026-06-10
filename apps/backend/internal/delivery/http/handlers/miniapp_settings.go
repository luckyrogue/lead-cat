package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// miniAppSettingsBotUser extracts the authed bot user identity for settings handlers.
func miniAppSettingsBotUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// MiniAppGetSettings — GET /api/miniapp/settings
func (a *API) MiniAppGetSettings(c *fiber.Ctx) error {
	bu, ok := miniAppSettingsBotUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	s, err := a.App.GetUserSettings(c.Context(), bu.TelegramID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(s)
}

// MiniAppPatchSettings — PATCH /api/miniapp/settings
func (a *API) MiniAppPatchSettings(c *fiber.Ctx) error {
	bu, ok := miniAppSettingsBotUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var body struct {
		ReminderMinutes *[]int `json:"reminder_minutes"`
	}
	if err := c.BodyParser(&body); err != nil || body.ReminderMinutes == nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if err := a.App.SetUserReminderMinutes(c.Context(), bu.TelegramID, *body.ReminderMinutes); err != nil {
		if errors.Is(err, application.ErrInvalidReminderMinute) {
			return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
