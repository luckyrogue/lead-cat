package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/metrics"
)

func PrometheusHTTP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}
		status := c.Response().StatusCode()
		if err != nil {
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			} else if status < 400 {
				status = fiber.StatusInternalServerError
			}
		}
		metrics.IncHTTP(c.Method(), path, status)
		return err
	}
}
