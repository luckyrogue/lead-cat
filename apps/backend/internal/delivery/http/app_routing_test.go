package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/config"
	"go.uber.org/zap"
)

func TestAuthRouting_WebNotGone(t *testing.T) {
	t.Parallel()

	app, err := NewApp(testRoutingConfig(), postgres.New(nil, zap.NewNop()), nil, nil, nil, zap.NewNop(), &application.Services{Log: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/web/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGone {
		t.Fatal("GET /api/auth/web/me must not return 410")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/auth/web/me status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthRouting_MiniAppNotGone(t *testing.T) {
	t.Parallel()

	app, err := NewApp(testRoutingConfig(), postgres.New(nil, zap.NewNop()), nil, nil, nil, zap.NewNop(), &application.Services{Log: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/miniapp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGone {
		t.Fatal("POST /api/auth/miniapp must not return 410")
	}
}

func TestAuthRouting_LegacyPlatformGone(t *testing.T) {
	t.Parallel()

	app, err := NewApp(testRoutingConfig(), postgres.New(nil, zap.NewNop()), nil, nil, nil, zap.NewNop(), &application.Services{Log: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/workspaces"},
		{http.MethodPost, "/api/auth/email/login"},
		{http.MethodPost, "/api/auth/passkey/register"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusGone {
			t.Fatalf("%s %s status = %d, want 410", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func testRoutingConfig() config.Config {
	return config.Config{
		JWTSecret:  "test-jwt-secret-32bytes-min",
		JWTIssuer:  "lead-cat-test",
		BotToken:   "123456:ABC-DEF",
		StaticDir:  "/nonexistent-static-dir",
		AppBaseURL: "http://localhost:8080",
	}
}
