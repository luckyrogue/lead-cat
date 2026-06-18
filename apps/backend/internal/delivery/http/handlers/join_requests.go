package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) WebRequestToJoin(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	var body struct {
		Slug string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Slug) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing_slug")
	}
	res, err := a.App.RequestToJoinBySlug(c.UserContext(), user.ID, strings.TrimSpace(strings.ToLower(body.Slug)))
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "org_not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "join_failed")
	}
	if res.AlreadyMember {
		return c.JSON(fiber.Map{"already_member": true, "organization_id": res.OrganizationID})
	}
	return c.JSON(fiber.Map{"status": "pending", "organization_id": res.OrganizationID})
}

func (a *API) WebMyJoinRequests(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	views, err := a.App.ListMyJoinRequests(c.UserContext(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) OrgJoinRequests(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	views, err := a.App.ListOrgJoinRequests(c.UserContext(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) AcceptJoinRequest(c *fiber.Ctx) error {
	return a.decideJoinRequest(c, true)
}

func (a *API) DeclineJoinRequest(c *fiber.Ctx) error {
	return a.decideJoinRequest(c, false)
}

func (a *API) decideJoinRequest(c *fiber.Ctx, accept bool) error {
	user := c.Locals("web_user").(model.PlatformUser)
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	rid, err := uuid.Parse(c.Params("rid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if accept {
		err = a.App.AcceptJoinRequest(c.UserContext(), orgID, rid, user.ID)
	} else {
		err = a.App.DeclineJoinRequest(c.UserContext(), orgID, rid, user.ID)
	}
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "request_not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "decide_failed")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
