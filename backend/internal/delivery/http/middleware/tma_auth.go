package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
)

type tmaStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
}

// TMAAuth guards /api/tma/* with a TMA session JWT and resolves the bot_users
// row each request (so role/email changes and de-registration take effect).
type TMAAuth struct {
	token *auth.TMAToken
	store tmaStore
}

func NewTMAAuth(token *auth.TMAToken, store *postgres.Store) *TMAAuth {
	return &TMAAuth{token: token, store: store}
}

func (m *TMAAuth) Middleware(c *fiber.Ctx) error {
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
