package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
)

type miniappAuthRequest struct {
	InitData string `json:"init_data"`
}

type miniappUser struct {
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
}

func (a *API) MiniAppAuth(c *fiber.Ctx) error {
	var req miniappAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	var tgID int64
	if a.AuthDevMode && !looksLikeTelegramInitData(req.InitData) {

		id, err := strconv.ParseInt(strings.TrimSpace(req.InitData), 10, 64)
		if err != nil || id == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "dev init_data must be a telegram id")
		}
		tgID = id
	} else {
		u, err := a.InitData.Validate(req.InitData)
		if err != nil {
			a.Log.Warn("miniapp_auth_invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		if !telegram.FreshAuthDate(u.AuthDate, time.Now(), 24*time.Hour) {
			a.Log.Warn("miniapp_auth_invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		tgID = u.ID
	}
	bu, err := a.App.GetBotUserByTelegramID(c.Context(), tgID)
	if err != nil {
		if model.IsNotFound(err) {
			a.Log.Info("miniapp_auth_unregistered", zap.Int64("telegram_id", tgID))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "not_registered"})
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	token, err := a.MiniAppToken.Issue(bu.TelegramID, bu.Email, bu.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "token issue failed")
	}
	a.Log.Info("miniapp_auth_ok", zap.Int64("telegram_id", bu.TelegramID))
	return c.JSON(fiber.Map{
		"token": token,
		"user":  miniappUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role},
	})
}

func looksLikeTelegramInitData(initData string) bool {
	return strings.Contains(initData, "hash=") || strings.Contains(initData, "auth_date=")
}
