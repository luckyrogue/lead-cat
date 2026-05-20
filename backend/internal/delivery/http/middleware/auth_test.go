package middleware

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/config"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubAuthStore struct {
	users map[string]postgres.User
}

func (s *stubAuthStore) GetUserByID(_ context.Context, id uuid.UUID) (postgres.User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return postgres.User{}, fiber.ErrUnauthorized
}

func (s *stubAuthStore) UpsertUserIdentity(_ context.Context, authSub, email, phone string) (postgres.User, error) {
	if u, ok := s.users[authSub]; ok {
		return u, nil
	}
	u := postgres.User{ID: uuid.New(), AuthSub: authSub, Email: email, Phone: phone}
	s.users[authSub] = u
	return u, nil
}

func TestAuthMiddleware_NoBearer(t *testing.T) {
	jwtSvc, _ := platformauth.NewJWT("test-secret-key-32chars-min", "test", time.Hour)
	a := &Auth{cfg: config.Config{}, jwt: jwtSvc, store: &stubAuthStore{users: map[string]postgres.User{}}, log: zap.NewNop()}
	app := fiber.New()
	app.Use(a.Middleware)
	app.Get("/api/me", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/api/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("want 401 got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_DevMode(t *testing.T) {
	a := &Auth{
		cfg:   config.Config{AuthDevMode: true, AuthDevSub: "dev-user", AuthDevEmail: "dev@test"},
		store: &stubAuthStore{users: map[string]postgres.User{}},
		log:   zap.NewNop(),
	}
	app := fiber.New()
	app.Use(a.Middleware)
	app.Get("/api/me", func(c *fiber.Ctx) error {
		_, ok := c.Locals("user_id").(uuid.UUID)
		if !ok {
			t.Fatal("user_id missing")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer dev")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("want 200 got %d %s", resp.StatusCode, b)
	}
}

func TestAuthMiddleware_JWT(t *testing.T) {
	jwtSvc, _ := platformauth.NewJWT("test-secret-key-32chars-min", "test", time.Hour)
	uid := uuid.New()
	store := &stubAuthStore{users: map[string]postgres.User{
		"email:test@x.com": {ID: uid, AuthSub: "email:test@x.com", Email: "test@x.com"},
	}}
	tok, _ := jwtSvc.Issue(uid, "email:test@x.com", "test@x.com", "")
	a := &Auth{cfg: config.Config{}, jwt: jwtSvc, store: store, log: zap.NewNop()}
	app := fiber.New()
	app.Use(a.Middleware)
	app.Get("/api/me", func(c *fiber.Ctx) error {
		if got, _ := c.Locals("user_id").(uuid.UUID); got != uid {
			t.Fatalf("uid mismatch")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}
}
