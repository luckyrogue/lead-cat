package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/config"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/copy"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
)

type ctxKey string

const UserSubKey ctxKey = "user_sub"
const UserIDKey ctxKey = "user_id"

type authStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.User, error)
	UpsertUserIdentity(ctx context.Context, authSub, email, phone string) (postgres.User, error)
}

type Auth struct {
	cfg   config.Config
	jwt   *auth.JWT
	store authStore
	log   *zap.Logger
}

func NewAuth(cfg config.Config, jwtSvc *auth.JWT, store *postgres.Store, log *zap.Logger) *Auth {
	return &Auth{cfg: cfg, jwt: jwtSvc, store: store, log: log}
}

func (a *Auth) Middleware(c *fiber.Ctx) error {
	if c.Path() == "/api/health" || c.Path() == "/metrics" || strings.HasPrefix(c.Path(), "/api/auth/") {
		return c.Next()
	}
	hdr := c.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
	}
	token := strings.TrimPrefix(hdr, "Bearer ")

	if a.cfg.AuthDevMode {
		sub := token
		if sub == "" || sub == "dev" {
			sub = a.cfg.AuthDevSub
		}
		devUser, err := a.store.UpsertUserIdentity(c.UserContext(), sub, a.cfg.AuthDevEmail, "")
		if err != nil {
			a.log.Error("dev upsert user", append(log.FieldsFromContext(c.UserContext()), zap.Error(err))...)
			return fiber.NewError(fiber.StatusInternalServerError, copy.APIError("internal"))
		}
		c.Locals("user_sub", sub)
		c.Locals("user_id", devUser.ID)
		return c.Next()
	}

	claims, err := a.jwt.Parse(token)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
	}
	u, err := a.store.GetUserByID(c.UserContext(), uid)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
	}
	c.Locals("user_sub", u.AuthSub)
	c.Locals("user_id", u.ID)
	return c.Next()
}
