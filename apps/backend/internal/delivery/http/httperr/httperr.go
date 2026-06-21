package httperr

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

func PublicMessage(err error) string {
	if err == nil {
		return ""
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		if fe.Code >= fiber.StatusInternalServerError {
			return "internal_error"
		}
		if fe.Message != "" {
			return fe.Message
		}
	}
	return "internal_error"
}
