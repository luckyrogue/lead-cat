package middleware

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubWorkspaceAccess struct {
	allowed bool
	err     error
}

func (s *stubWorkspaceAccess) UserCanAccessWorkspace(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return s.allowed, s.err
}

func TestRequireWorkspaceAccess_AllowOwner(t *testing.T) {
	wid := uuid.New()
	uid := uuid.New()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uid)
		return c.Next()
	})
	app.Get("/workspaces/:id", RequireWorkspaceAccess(&stubWorkspaceAccess{allowed: true}), func(c *fiber.Ctx) error {
		if got, ok := c.Locals("workspace_id").(uuid.UUID); !ok || got != wid {
			t.Fatalf("workspace_id not set")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/workspaces/"+wid.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body %s", resp.StatusCode, b)
	}
}

func TestRequireWorkspaceAccess_DenyStranger(t *testing.T) {
	wid := uuid.New()
	uid := uuid.New()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uid)
		return c.Next()
	})
	app.Get("/workspaces/:id", RequireWorkspaceAccess(&stubWorkspaceAccess{allowed: false}), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/workspaces/"+wid.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("want 403 got %d", resp.StatusCode)
	}
}

func TestRequireWorkspaceAccess_InvalidUUID(t *testing.T) {
	uid := uuid.New()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uid)
		return c.Next()
	})
	app.Get("/workspaces/:id", RequireWorkspaceAccess(&stubWorkspaceAccess{allowed: true}), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/workspaces/not-a-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

func TestRequireWorkspaceAccess_StoreError(t *testing.T) {
	wid := uuid.New()
	uid := uuid.New()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uid)
		return c.Next()
	})
	app.Get("/workspaces/:id", RequireWorkspaceAccess(&stubWorkspaceAccess{err: errors.New("db")}), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/workspaces/"+wid.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("want 500 got %d", resp.StatusCode)
	}
}
