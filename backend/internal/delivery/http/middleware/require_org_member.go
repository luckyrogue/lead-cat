package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// OrgMemberResolver resolves a user's membership in an organization.
type OrgMemberResolver interface {
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (postgres.Member, bool, error)
}

// RequireOrgMember resolves the org id (from the :id path param, else the
// X-Org-Id header), verifies the authed web user is a member, and stores the
// membership in c.Locals("org_member"). 400 on missing/invalid org id, 403 if
// the user is not a member. Must run after WebAuth (which sets "web_user").
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

// RequireOrgRole gates on a minimum role; must run AFTER RequireOrgMember
// (reads the membership it stored). 403 if the member's role is insufficient.
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
