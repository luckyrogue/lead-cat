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
		Name     string `json:"name"`
		Email    string `json:"email"`
		Start    string `json:"start"`
		Language string `json:"language"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request")
	}
	start, err := time.Parse(time.RFC3339, body.Start)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_start")
	}
	conf, err := a.App.SubmitBooking(c.UserContext(), slug, application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
	if err != nil {
		switch {
		case model.IsNotFound(err):
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		case errors.Is(err, model.ErrSlotTaken):
			return a.declineWithSurvey(c, slug, "slot_taken", fiber.StatusConflict,
				application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
		case errors.Is(err, model.ErrInvalidBooking):
			return a.declineWithSurvey(c, slug, "invalid_booking", fiber.StatusBadRequest,
				application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
		}
	}
	return c.JSON(conf)
}

func (a *API) declineWithSurvey(c *fiber.Ctx, slug, reason string, status int, req application.BookingRequest) error {
	body := fiber.Map{"error": "error", "message": reason}
	et, err := a.App.GetBookingEventTypeBySlugPublic(c.UserContext(), slug)
	if err == nil {
		if token, terr := a.App.CreatePendingResponse(c.UserContext(), et, reason, req); terr == nil && token != "" {
			body["survey_token"] = token
		}
	}
	return c.Status(status).JSON(body)
}
