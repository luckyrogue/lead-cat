package handlers

import "github.com/gofiber/fiber/v2"

// PlatformGone marks retired platform bootstrap routes (workspace JWT, OTP, passkey, OAuth).
func PlatformGone(c *fiber.Ctx) error {
	c.Set("Deprecation", "true")
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"error":   "deprecated",
		"message": "Platform API retired; use Telegram Mini App and /api/miniapp/admin/* for workspace setup",
	})
}
