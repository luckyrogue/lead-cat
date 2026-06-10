package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres" //nolint:depguard
)

func TestCSRFMatches(t *testing.T) {
	if !csrfMatches("abc", "abc") {
		t.Fatal("equal should match")
	}
	if csrfMatches("abc", "abd") {
		t.Fatal("different must not match")
	}
	if csrfMatches("", "") {
		t.Fatal("empty must not match")
	}
	if csrfMatches("abc", "") {
		t.Fatal("empty cookie must not match")
	}
}

type fakeResolver struct {
	user postgres.PlatformUser
	ok   bool
}

func (f fakeResolver) ResolveWebUser(_ context.Context, _ string) (postgres.PlatformUser, bool, error) {
	return f.user, f.ok, nil
}

func newTestApp(r WebUserResolver) *fiber.App {
	app := fiber.New()
	wa := NewWebAuth(r)
	app.Use(wa.Middleware)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Post("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func TestWebAuthRejectsMissingCookie(t *testing.T) {
	app := newTestApp(fakeResolver{ok: false})
	resp, _ := app.Test(httptest.NewRequest("GET", "/x", nil))
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

func TestWebAuthAllowsValidGet(t *testing.T) {
	app := newTestApp(fakeResolver{ok: true, user: postgres.PlatformUser{Email: "a@b.com"}})
	req := httptest.NewRequest("GET", "/x", nil)
	req.AddCookie(&http.Cookie{Name: "lc_session", Value: "rawtoken"})
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestWebAuthBlocksPostWithoutCSRF(t *testing.T) {
	app := newTestApp(fakeResolver{ok: true, user: postgres.PlatformUser{Email: "a@b.com"}})
	req := httptest.NewRequest("POST", "/x", nil)
	req.AddCookie(&http.Cookie{Name: "lc_session", Value: "rawtoken"})
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("want 403 (csrf), got %d", resp.StatusCode)
	}
}

func TestWebAuthAllowsPostWithMatchingCSRF(t *testing.T) {
	app := newTestApp(fakeResolver{ok: true, user: postgres.PlatformUser{Email: "a@b.com"}})
	req := httptest.NewRequest("POST", "/x", nil)
	req.AddCookie(&http.Cookie{Name: "lc_session", Value: "rawtoken"})
	req.AddCookie(&http.Cookie{Name: "lc_csrf", Value: "tok123"})
	req.Header.Set("X-CSRF-Token", "tok123")
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}
