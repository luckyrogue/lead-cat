package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
)

func appWithSec(prod bool) *fiber.App {
	app := fiber.New()
	app.Use(middleware.SecurityHeaders(prod))
	app.Get("/api/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Post("/api/auth/web/magic/request", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func TestSecurityHeaders_Common(t *testing.T) {
	resp, err := appWithSec(false).Test(httptest.NewRequest("GET", "/api/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("nosniff = %q", got)
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if resp.Header.Get("Referrer-Policy") == "" {
		t.Error("Referrer-Policy missing")
	}
	if resp.Header.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must be absent in non-prod")
	}
}

func TestSecurityHeaders_ProdHSTS(t *testing.T) {
	resp, _ := appWithSec(true).Test(httptest.NewRequest("GET", "/api/x", nil))
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("HSTS must be present in prod")
	}
}

func TestSecurityHeaders_AuthNoStore(t *testing.T) {
	resp, _ := appWithSec(false).Test(httptest.NewRequest("POST", "/api/auth/web/magic/request", nil))
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("auth Cache-Control = %q, want no-store", got)
	}
	// non-auth path should NOT be forced no-store
	resp2, _ := appWithSec(false).Test(httptest.NewRequest("GET", "/api/x", nil))
	if resp2.Header.Get("Cache-Control") == "no-store" {
		t.Error("non-auth path should not be no-store")
	}
}
