package handlers

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) withWebAuditActor(c *fiber.Ctx) {
	u, ok := c.Locals("web_user").(model.PlatformUser)
	if !ok {
		return
	}
	c.SetUserContext(application.WithWebAuditActor(c.UserContext(), u.ID, u.Email))
}

var validRoles = map[string]struct{}{
	"member": {},
	"admin":  {},
	"owner":  {},
}

func (a *API) CreateOrg(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	org, err := a.App.CreateOrganizationForOwner(c.UserContext(), body.Name, user.ID)
	if err != nil {
		a.Log.Error("org_create_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "create_failed")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":   org.ID,
		"name": org.Name,
		"slug": org.Slug,
	})
}

func (a *API) ListMyOrgs(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	orgs, err := a.App.ListOrganizationsForUser(c.UserContext(), user.ID)
	if err != nil {
		a.Log.Error("org_list_failed", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	out := make([]fiber.Map, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, fiber.Map{
			"id":   o.ID,
			"name": o.Name,
			"slug": o.Slug,
		})
	}
	return c.JSON(fiber.Map{"organizations": out})
}

func (a *API) ListOrgMembers(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	members, err := a.App.ListOrgMembers(c.UserContext(), orgID)
	if err != nil {
		a.Log.Error("org_members_list_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	out := make([]fiber.Map, 0, len(members))
	for _, m := range members {
		email, name, status := memberView(m)
		out = append(out, fiber.Map{
			"user_id":           m.UserID,
			"role":              m.Role,
			"email":             email,
			"name":              name,
			"status":            status,
			"invited_email":     m.InvitedEmail,
			"telegram_username": m.TelegramUsername,
		})
	}
	return c.JSON(fiber.Map{"members": out})
}

func memberView(m model.Member) (email, name, status string) {
	email = m.Email
	if email == "" && m.InvitedEmail != nil {
		email = *m.InvitedEmail
	}
	status = "invited"
	if m.UserID != nil {
		status = "active"
	}
	switch {
	case m.TelegramUsername != "":
		name = "@" + strings.TrimPrefix(m.TelegramUsername, "@")
	case email != "":
		name = email[:strings.IndexByte(email+"@", '@')]
	}
	return email, name, status
}

func (a *API) InviteMember(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if _, err := mail.ParseAddress(body.Email); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_email")
	}
	if body.Role == "" {
		body.Role = "member"
	}
	if _, ok := validRoles[body.Role]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_role")
	}
	inv, err := a.App.InviteToOrg(c.UserContext(), orgID, body.Email, body.Role, user.ID)
	if err != nil {
		a.Log.Error("org_invite_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "invite_failed")
	}
	a.withWebAuditActor(c)
	a.App.Audit(c.UserContext(), "org_invited", "organization", orgID.String(), map[string]any{"email": body.Email, "role": body.Role})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":    inv.ID,
		"email": inv.Email,
		"role":  inv.Role,
	})
}

func (a *API) ListInvites(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	invites, err := a.App.ListOrgInvites(c.UserContext(), orgID)
	if err != nil {
		a.Log.Error("org_invites_list_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	out := make([]fiber.Map, 0, len(invites))
	for _, inv := range invites {
		out = append(out, fiber.Map{
			"id":         inv.ID,
			"email":      inv.Email,
			"role":       inv.Role,
			"expires_at": inv.ExpiresAt,
		})
	}
	return c.JSON(fiber.Map{"invites": out})
}

func (a *API) DeleteInvite(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	inviteID, err := uuid.Parse(c.Params("iid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_invite_id")
	}
	if err := a.App.DeleteOrgInvite(c.UserContext(), orgID, inviteID); err != nil {
		a.Log.Error("org_invite_delete_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "delete_failed")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) UpdateMemberRole(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	targetUID, err := uuid.Parse(c.Params("uid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_user_id")
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil || body.Role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if _, ok := validRoles[body.Role]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_role")
	}
	if err := a.App.SetOrgMemberRole(c.UserContext(), orgID, targetUID, body.Role); err != nil {
		if errors.Is(err, application.ErrLastOwner) {
			return fiber.NewError(fiber.StatusConflict, "last_owner")
		}
		a.Log.Error("org_member_role_update_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "update_failed")
	}
	a.withWebAuditActor(c)
	a.App.Audit(c.UserContext(), "org_member_role_changed", "organization", orgID.String(), map[string]any{"target_user_id": targetUID.String(), "role": body.Role})
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) RemoveMember(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	targetUID, err := uuid.Parse(c.Params("uid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_user_id")
	}
	if err := a.App.RemoveOrgMember(c.UserContext(), orgID, targetUID); err != nil {
		if errors.Is(err, application.ErrLastOwner) {
			return fiber.NewError(fiber.StatusConflict, "last_owner")
		}
		a.Log.Error("org_member_remove_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "remove_failed")
	}
	a.withWebAuditActor(c)
	a.App.Audit(c.UserContext(), "org_member_removed", "organization", orgID.String(), map[string]any{"target_user_id": targetUID.String()})
	return c.SendStatus(fiber.StatusNoContent)
}
