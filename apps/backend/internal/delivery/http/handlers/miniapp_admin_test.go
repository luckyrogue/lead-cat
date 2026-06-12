package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestMiniappAdminBotUser(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		if _, ok := miniappAdminBotUser(c); ok {
			t.Fatal("expected no bot_user")
		}
		c.Locals("bot_user", model.BotUser{Role: "user"})
		if _, ok := miniappAdminBotUser(c); ok {
			t.Fatal("non-admin must be rejected")
		}
		c.Locals("bot_user", model.BotUser{Role: "admin", TelegramID: 1})
		if _, ok := miniappAdminBotUser(c); !ok {
			t.Fatal("admin must pass")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestDeprecatedAdminWorkspaceSetsDeprecationHeader(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/workspace", DeprecatedAdminWorkspace(func(c *fiber.Ctx) error {
		return c.SendString("ok")
	}))
	req := httptest.NewRequest("GET", "/workspace", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("Deprecation") != "true" {
		t.Fatal("expected Deprecation: true header")
	}
}
