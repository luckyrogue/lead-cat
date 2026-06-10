package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/openapi"
)

// OpenAPI serves the embedded OpenAPI document for frontend codegen.
func OpenAPI(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/json")
	return c.Send(openapi.Spec)
}
