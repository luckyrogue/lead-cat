package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) PublicBooking(c *fiber.Ctx) error {
	slug := c.Params("slug")
	view, err := a.App.PublicBooking(c.UserContext(), slug, time.Now())
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
	}
	return c.JSON(view)
}
