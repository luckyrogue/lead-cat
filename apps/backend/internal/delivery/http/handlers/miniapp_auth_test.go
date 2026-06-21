package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

type authBotRepo struct {
	stubRepo
}

func (r *authBotRepo) GetBotUserByTelegramID(_ context.Context, tid int64) (model.BotUser, error) {
	if tid == 12345 {
		return model.BotUser{TelegramID: 12345, FullName: "Dev", Email: "dev@test.com", Role: "user"}, nil
	}
	return model.BotUser{}, sql.ErrNoRows
}

func newAuthAPI(t *testing.T, devMode bool, appEnv string) *handlers.API {
	t.Helper()
	tok, err := platformauth.NewMiniAppToken("test-jwt-secret-min-16", "lead-cat", 0)
	if err != nil {
		t.Fatal(err)
	}
	return &handlers.API{
		App:               &application.Services{Store: &authBotRepo{}},
		Log:               zap.NewNop(),
		InitData:          telegram.NewInitDataValidator("000000000:AAFakeDevTokenForLocalOnly"),
		MiniAppToken:      tok,
		AuthDevMode:       devMode,
		AppEnv:            appEnv,
		TrustProxyHeaders: true,
	}
}

func postMiniAppAuth(t *testing.T, api *handlers.API, body string, clientIP ...string) (int, map[string]any) {
	t.Helper()
	app := fiber.New(fiber.Config{
		EnableTrustedProxyCheck: true,
		ProxyHeader:             "X-Real-IP",
	})
	app.Post("/auth", api.MiniAppAuth)
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ip := "127.0.0.1"
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
	}
	req.Header.Set("X-Real-IP", ip)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

func TestMiniAppAuth_DevBypass_LocalDevelopment(t *testing.T) {
	api := newAuthAPI(t, true, "development")
	status, body := postMiniAppAuth(t, api, `{"init_data":"12345"}`)
	if status != http.StatusOK {
		t.Fatalf("status %d body %v", status, body)
	}
	if body["token"] == nil || body["token"] == "" {
		t.Fatal("expected token")
	}
}

func TestMiniAppAuth_DevBypass_RejectedOnStaging(t *testing.T) {
	api := newAuthAPI(t, true, "staging")
	status, body := postMiniAppAuth(t, api, `{"init_data":"12345"}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
	if body["code"] != "invalid_init_data" {
		t.Fatalf("body %v", body)
	}
}

func TestMiniAppAuth_DevBypass_RejectedFromNonLoopback(t *testing.T) {
	api := newAuthAPI(t, true, "development")
	status, body := postMiniAppAuth(t, api, `{"init_data":"12345"}`, "203.0.113.1")
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
	if body["code"] != "invalid_init_data" {
		t.Fatalf("body %v", body)
	}
}
