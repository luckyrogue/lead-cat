package handlers

import "github.com/gofiber/fiber/v2"

func PlatformGone(c *fiber.Ctx) error {
	c.Set("Deprecation", "true")
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"error":   "deprecated",
		"message": "Platform API retired; use /api/auth/web/*, /api/orgs/*, or /api/miniapp/admin/* for organization setup",
	})
}

func DeprecatedAdminWorkspace(next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Deprecation", "true")
		return next(c)
	}
}
