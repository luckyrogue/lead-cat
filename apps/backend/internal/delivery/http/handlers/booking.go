package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type eventTypeBody struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	DurationMins     int    `json:"duration_mins"`
	Timezone         string `json:"timezone"`
	AvailWeekdays    []int  `json:"avail_weekdays"`
	AvailStartMinute int    `json:"avail_start_minute"`
	AvailEndMinute   int    `json:"avail_end_minute"`
	Active           bool   `json:"active"`
}

func (b eventTypeBody) toInput() application.EventTypeInput {
	return application.EventTypeInput{
		Title:            b.Title,
		Description:      b.Description,
		DurationMins:     b.DurationMins,
		Timezone:         b.Timezone,
		AvailWeekdays:    b.AvailWeekdays,
		AvailStartMinute: b.AvailStartMinute,
		AvailEndMinute:   b.AvailEndMinute,
		Active:           b.Active,
	}
}

func bookingErr(err error) error {
	if errors.Is(err, application.ErrInvalidEventType) {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if errors.Is(err, model.ErrForbidden) {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	if model.IsNotFound(err) {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}

func (a *API) BookingListEventTypes(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	list, err := a.App.ListMyEventTypes(c.UserContext(), user.ID)
	if err != nil {
		return bookingErr(err)
	}
	return c.JSON(list)
}

func (a *API) BookingCreateEventType(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	orgRaw := c.Get("X-Org-Id")
	orgID, err := uuid.Parse(orgRaw)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	var body eventTypeBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	et, err := a.App.CreateEventType(c.UserContext(), user.ID, orgID, body.toInput())
	if err != nil {
		return bookingErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(et)
}

func (a *API) BookingUpdateEventType(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	var body eventTypeBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	if err := a.App.UpdateEventType(c.UserContext(), user.ID, id, body.toInput()); err != nil {
		return bookingErr(err)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (a *API) BookingDeleteEventType(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if err := a.App.DeleteEventType(c.UserContext(), user.ID, id); err != nil {
		return bookingErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
