package handlers

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func internalAPIError(log *zap.Logger, event string, err error) error {
	if err != nil && log != nil {
		log.Error(event, zap.Error(err))
	}
	return fiber.NewError(fiber.StatusInternalServerError, "internal")
}
