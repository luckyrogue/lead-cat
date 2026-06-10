package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres" //nolint:depguard
)

type OrgMemberResolver interface {
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (postgres.Member, bool, error)
}

func RequireOrgMember(r OrgMemberResolver) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("web_user").(postgres.PlatformUser)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		idStr := c.Params("id")
		if idStr == "" {
			idStr = c.Get("X-Org-Id")
		}
		orgID, err := uuid.Parse(idStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "org_required")
		}
		m, found, err := r.GetOrgMember(c.UserContext(), orgID, user.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "internal_error")
		}
		if !found {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		c.Locals("org_member", m)
		return c.Next()
	}
}

func RequireOrgRole(minRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		m, ok := c.Locals("org_member").(postgres.Member)
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		if !application.RoleAtLeast(m.Role, minRole) {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		return c.Next()
	}
}
