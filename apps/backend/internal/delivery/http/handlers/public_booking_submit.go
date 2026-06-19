package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) PublicBookingSubmit(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Start string `json:"start"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request")
	}
	start, err := time.Parse(time.RFC3339, body.Start)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_start")
	}
	conf, err := a.App.SubmitBooking(c.UserContext(), slug, application.BookingRequest{Name: body.Name, Email: body.Email, Start: start})
	if err != nil {
		switch {
		case model.IsNotFound(err):
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		case errors.Is(err, model.ErrSlotTaken):
			return fiber.NewError(fiber.StatusConflict, "slot_taken")
		case errors.Is(err, model.ErrInvalidBooking):
			return fiber.NewError(fiber.StatusBadRequest, "invalid_booking")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
		}
	}
	return c.JSON(conf)
}
