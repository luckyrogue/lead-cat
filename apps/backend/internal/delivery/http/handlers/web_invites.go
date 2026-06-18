package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) WebMyInvites(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	views, err := a.App.ListMyInvites(c.UserContext(), user.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) WebAcceptInvite(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	iid, err := uuid.Parse(c.Params("iid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if err := a.App.AcceptInvite(c.UserContext(), iid, user.ID, user.Email); err != nil {
		return inviteError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) WebDeclineInvite(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	iid, err := uuid.Parse(c.Params("iid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if err := a.App.DeclineInvite(c.UserContext(), iid, user.Email); err != nil {
		return inviteError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func inviteError(err error) error {
	if errors.Is(err, model.ErrInviteEmailMismatch) {
		return fiber.NewError(fiber.StatusForbidden, "email_mismatch")
	}
	if model.IsNotFound(err) {
		return fiber.NewError(fiber.StatusNotFound, "invite_not_found")
	}
	return fiber.NewError(fiber.StatusInternalServerError, "invite_failed")
}
