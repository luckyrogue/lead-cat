package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/openapi"
)

func OpenAPI(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/json")
	return c.Send(openapi.Spec)
}
