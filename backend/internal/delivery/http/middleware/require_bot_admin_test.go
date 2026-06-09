package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestRequireBotAdmin_NoLocal_Returns403(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireBotAdmin_NonAdmin_Returns403(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", postgres.BotUser{Role: "user"})
		return c.Next()
	}, middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireBotAdmin_Admin_Passes(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", postgres.BotUser{Role: "admin"})
		return c.Next()
	}, middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
