package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
)

var (
	bobID   = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	orgID   = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
	inviteA = uuid.MustParse("cccccccc-0000-0000-0000-000000000003")
)

type inviteFakeRepo struct {
	stubRepo
	listFn    func(ctx context.Context, email string) ([]model.InviteView, error)
	acceptFn  func(ctx context.Context, inviteID, userID uuid.UUID, email string) error
	declineFn func(ctx context.Context, inviteID uuid.UUID, email string) error
}

func (r *inviteFakeRepo) ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error) {
	return r.listFn(ctx, email)
}

func (r *inviteFakeRepo) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	return r.acceptFn(ctx, inviteID, userID, email)
}

func (r *inviteFakeRepo) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	return r.declineFn(ctx, inviteID, email)
}

func buildInviteApp(t *testing.T, repo *inviteFakeRepo) *fiber.App {
	t.Helper()
	svc := &application.Services{
		Store: repo,
		Log:   zap.NewNop(),
	}

	api := &handlers.API{
		App: svc,
		Log: zap.NewNop(),
	}

	app := fiber.New(fiber.Config{
		Immutable: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("web_user", model.PlatformUser{ID: bobID, Email: "bob@x.com"})
		return c.Next()
	})

	app.Get("/api/auth/web/me/invites", api.WebMyInvites)
	app.Post("/api/auth/web/me/invites/:iid/accept", api.WebAcceptInvite)
	app.Post("/api/auth/web/me/invites/:iid/decline", api.WebDeclineInvite)

	return app
}

func newInviteRepo() *inviteFakeRepo {
	return &inviteFakeRepo{
		stubRepo: stubRepo{calendarFakeRepo: calendarFakeRepo{
			states: make(map[string]model.CalendarOAuthState),
			conns:  make(map[string][]model.CalendarConnection),
		}},
		listFn: func(_ context.Context, _ string) ([]model.InviteView, error) {
			return []model.InviteView{{InviteID: inviteA, OrganizationID: orgID, OrgName: "Acme", Role: "member"}}, nil
		},
		acceptFn:  func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil },
		declineFn: func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
	}
}

func TestWebMyInvites_List(t *testing.T) {
	app := buildInviteApp(t, newInviteRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/auth/web/me/invites", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var views []model.InviteView
	if err := json.NewDecoder(resp.Body).Decode(&views); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(views) != 1 || views[0].InviteID != inviteA {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestWebAcceptInvite_OK(t *testing.T) {
	app := buildInviteApp(t, newInviteRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/me/invites/"+inviteA.String()+"/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, body)
	}
}

func TestWebAcceptInvite_EmailMismatch_Returns403(t *testing.T) {
	repo := newInviteRepo()
	repo.acceptFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return model.ErrInviteEmailMismatch
	}
	app := buildInviteApp(t, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/me/invites/"+inviteA.String()+"/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, body)
	}
}

func TestWebDeclineInvite_OK(t *testing.T) {
	app := buildInviteApp(t, newInviteRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/me/invites/"+inviteA.String()+"/decline", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, body)
	}
}

func TestWebAcceptInvite_NotFound_Returns404(t *testing.T) {
	repo := newInviteRepo()
	repo.acceptFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return sql.ErrNoRows
	}
	app := buildInviteApp(t, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/me/invites/"+inviteA.String()+"/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, body)
	}
}

func TestWebAcceptInvite_BadUUID_Returns400(t *testing.T) {
	app := buildInviteApp(t, newInviteRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/web/me/invites/not-a-uuid/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}
}
