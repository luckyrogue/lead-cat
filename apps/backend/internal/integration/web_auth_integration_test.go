package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type captureMailer struct{ lastBody string }

func (m *captureMailer) Send(_ context.Context, _, _, htmlBody string) error {
	m.lastBody = htmlBody
	return nil
}

func extractToken(body string) string {
	const marker = "token="
	i := strings.Index(body, marker)
	if i < 0 {
		return ""
	}
	rest := body[i+len(marker):]
	if j := strings.IndexByte(rest, '"'); j >= 0 {
		return rest[:j]
	}
	return rest
}

func TestMagicLinkRoundTripIssuesSession(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := postgres.New(pool, zap.NewNop())

	mailer := &captureMailer{}
	services := &application.Services{Store: store, Log: zap.NewNop()}
	services.ConfigureWebAuth(map[string]application.SSOProvider{}, mailer,
		"http://localhost:8080", 720*time.Hour, 15*time.Minute)
	services.WireCQRS()

	api := &handlers.API{App: services, Log: zap.NewNop()}
	webAuth := middleware.NewWebAuth(services)

	app := fiber.New()
	web := app.Group("/api/auth/web")
	web.Post("/magic/request", api.WebMagicRequest)
	web.Get("/magic/verify", api.WebMagicVerify)
	web.Get("/me", webAuth.Middleware, api.WebMe)

	reqBody := strings.NewReader(`{"email":"itest@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/magic/request", reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("magic/request status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	token := extractToken(mailer.lastBody)
	if token == "" {
		t.Fatalf("no token captured in mail body: %q", mailer.lastBody)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/web/magic/verify?token="+token, nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("magic/verify status = %d, want 302", resp.StatusCode)
	}
	var sessionCookie *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == "lc_session" && ck.Value != "" {
			sessionCookie = ck
		}
	}
	if sessionCookie == nil {
		t.Fatalf("no lc_session cookie set on verify response: %v", resp.Header.Values("Set-Cookie"))
	}
	_ = resp.Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/api/auth/web/me", nil)
	req.AddCookie(sessionCookie)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d, want 200", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(bodyBytes), "itest@example.com") {
		t.Fatalf("me body missing email: %s", string(bodyBytes))
	}
}
