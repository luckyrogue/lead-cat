package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres" //nolint:depguard
	"github.com/luckyrogue/lead-cat/internal/platform/auth"
)

type miniappStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
}

type MiniAppAuth struct {
	token *auth.MiniAppToken
	store miniappStore
}

func NewMiniAppAuth(token *auth.MiniAppToken, store *postgres.Store) *MiniAppAuth {
	return &MiniAppAuth{token: token, store: store}
}

func (m *MiniAppAuth) Middleware(c *fiber.Ctx) error {
	hdr := c.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	claims, err := m.token.Parse(strings.TrimPrefix(hdr, "Bearer "))
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	bu, err := m.store.GetBotUserByTelegramID(c.UserContext(), claims.TelegramID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	c.Locals("bot_user", bu)
	return c.Next()
}
