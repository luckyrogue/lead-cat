package httpclient

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestClientIP_TrustedProxy(t *testing.T) {
	app := fiber.New(fiber.Config{
		EnableTrustedProxyCheck: true,
		ProxyHeader:             "X-Real-IP",
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(ClientIP(c, true))
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "203.0.113.1")
	resp, err := app.Test(r)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestClientIP_UntrustedIgnoresHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		ip := ClientIP(c, false)
		if ip == "203.0.113.1" {
			t.Fatal("must not trust X-Real-IP when trustProxy=false")
		}
		return c.SendStatus(200)
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "203.0.113.1")
	if _, err := app.Test(r); err != nil {
		t.Fatal(err)
	}
}

func TestIsLoopback(t *testing.T) {
	if !IsLoopback("127.0.0.1") {
		t.Fatal("127.0.0.1")
	}
	if !IsLoopback("::1") {
		t.Fatal("::1")
	}
	if IsLoopback("203.0.113.1") {
		t.Fatal("public IP")
	}
}
