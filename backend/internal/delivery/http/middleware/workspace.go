package middleware

import (
	"context"

	"github.com/Jaryq-Lab/notify-bot/internal/platform/copy"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type workspaceAccessChecker interface {
	UserCanAccessWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) (bool, error)
}

func RequireWorkspaceAccess(checker workspaceAccessChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid workspace id")
		}
		uid, ok := c.Locals("user_id").(uuid.UUID)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
		}
		allowed, err := checker.UserCanAccessWorkspace(c.Context(), uid, wid)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, copy.APIError("internal"))
		}
		if !allowed {
			return fiber.NewError(fiber.StatusForbidden, copy.APIError("forbidden"))
		}
		c.Locals("workspace_id", wid)
		c.SetUserContext(log.WithWorkspaceID(c.UserContext(), wid))
		return c.Next()
	}
}
