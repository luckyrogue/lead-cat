package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/copy"
)

func (a *API) ListEmployees(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	list, err := a.App.ListEmployees(c.Context(), wid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (a *API) ListMeetings(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	uid, _ := c.Locals("user_id").(uuid.UUID)
	list, err := a.App.ListMeetings(c.Context(), wid, uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (a *API) CreateMeeting(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	uid, _ := c.Locals("user_id").(uuid.UUID)
	var body struct {
		Dept            string                        `json:"dept"`
		Type            string                        `json:"type"`
		Host            string                        `json:"host"`
		Date            string                        `json:"date"`
		Start           string                        `json:"start"`
		End             string                        `json:"end"`
		Recurrence      string                        `json:"recurrence"`
		RecurrenceUntil string                        `json:"recurrence_until"`
		Description     string                        `json:"description"`
		Participants    []postgres.MeetingParticipant `json:"participants"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	m, err := a.App.CreateMeeting(c.Context(), wid, uid, application.CreateMeetingInput{
		Dept: body.Dept, Type: body.Type, Host: body.Host,
		Date: body.Date, Start: body.Start, End: body.End,
		Recurrence: body.Recurrence, RecurrenceUntil: body.RecurrenceUntil, Description: body.Description,
		Participants: body.Participants,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) || errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

func (a *API) GetMeeting(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	mid, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	m, err := a.App.GetMeeting(c.Context(), wid, mid)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, copy.APIError("not_found"))
	}
	return c.JSON(m)
}

func (a *API) DeleteMeeting(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	uid, _ := c.Locals("user_id").(uuid.UUID)
	mid, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	if err := a.App.CancelMeeting(c.Context(), wid, uid, mid); err != nil {
		if errors.Is(err, application.ErrForbidden) {
			return fiber.NewError(fiber.StatusForbidden, copy.APIError("forbidden"))
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
