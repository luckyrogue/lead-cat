package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"
)

type tmaAuthRequest struct {
	InitData string `json:"init_data"`
}

type tmaUser struct {
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
}

// TMAAuth exchanges Telegram initData for a short-lived TMA JWT. Public route.
// 401 bodies carry a machine-readable {"code": ...} so the Mini App can tell
// not_registered (→ "register in the bot" screen) from invalid_init_data.
func (a *API) TMAAuth(c *fiber.Ctx) error {
	var req tmaAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	var tgID int64
	if a.AuthDevMode {
		// Dev: no Telegram, no HMAC. init_data carries the dev telegram_id.
		id, err := strconv.ParseInt(strings.TrimSpace(req.InitData), 10, 64)
		if err != nil || id == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "dev init_data must be a telegram id")
		}
		tgID = id
	} else {
		u, err := a.TMA.Validate(req.InitData)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		if !telegram.FreshAuthDate(u.AuthDate, time.Now(), 24*time.Hour) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		tgID = u.ID
	}
	bu, err := a.App.Store.GetBotUserByTelegramID(c.Context(), tgID)
	if err != nil {
		if postgres.IsNotFound(err) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "not_registered"})
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	token, err := a.TMAToken.Issue(bu.TelegramID, bu.Email, bu.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "token issue failed")
	}
	return c.JSON(fiber.Map{
		"token": token,
		"user":  tmaUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role},
	})
}
