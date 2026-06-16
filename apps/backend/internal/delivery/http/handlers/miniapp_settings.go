package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func miniAppSettingsBotUser(c *fiber.Ctx) (model.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(model.BotUser)
	return bu, ok
}

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

func (a *API) MiniAppPatchSettings(c *fiber.Ctx) error {
	bu, ok := miniAppSettingsBotUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var body struct {
		ReminderMinutes *[]int  `json:"reminder_minutes"`
		Timezone        *string `json:"timezone"`
		Language        *string `json:"language"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if body.ReminderMinutes != nil {
		if err := a.App.SetUserReminderMinutes(c.Context(), bu.TelegramID, *body.ReminderMinutes); err != nil {
			if errors.Is(err, application.ErrInvalidReminderMinute) {
				return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if body.Timezone != nil || body.Language != nil {
		cur, err := a.App.GetUserSettings(c.Context(), bu.TelegramID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		tz := cur.Timezone
		if body.Timezone != nil {
			tz = *body.Timezone
		}
		lang := cur.Language
		if body.Language != nil {
			lang = *body.Language
		}
		if err := a.App.SetUserPrefs(c.Context(), bu.TelegramID, tz, lang); err != nil {
			if errors.Is(err, application.ErrInvalidInput) {
				return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}
