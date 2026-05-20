package handlers

import (
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (a *API) logHTTPError(c *fiber.Ctx, msg string, err error, extra ...zap.Field) {
	fields := append(log.FieldsFromContext(c.UserContext()), zap.Error(err))
	if wid, ok := c.Locals("workspace_id").(uuid.UUID); ok {
		fields = append(fields, zap.String("workspace_id", wid.String()))
	}
	fields = append(fields, extra...)
	a.Log.Error(msg, fields...)
}
