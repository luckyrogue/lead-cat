package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
)

func TestRequireBotAdmin_AllowsListedID(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", model.BotUser{TelegramID: 42, Role: "admin"})
		return c.Next()
	})
	app.Get("/", middleware.RequireBotAdmin([]int64{42}), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestRequireBotAdmin_RejectsUnlistedID(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", model.BotUser{TelegramID: 99, Role: "admin"})
		return c.Next()
	})
	app.Get("/", middleware.RequireBotAdmin([]int64{42}), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
