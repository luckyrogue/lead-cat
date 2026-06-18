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
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
)

var (
	joinUserID = uuid.MustParse("dddddddd-0000-0000-0000-000000000001")
	joinOrgID  = uuid.MustParse("eeeeeeee-0000-0000-0000-000000000002")
	joinReqID  = uuid.MustParse("ffffffff-0000-0000-0000-000000000003")
)

type joinFakeRepo struct {
	stubRepo
	getOrgBySlugFn         func(ctx context.Context, slug string) (model.Organization, error)
	getOrgMemberFn         func(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	createJoinRequestFn    func(ctx context.Context, orgID, userID uuid.UUID) error
	listJoinRequestsUserFn func(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error)
	listPendingJoinReqsFn  func(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)
	acceptJoinRequestFn    func(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
	declineJoinRequestFn   func(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
}

func (r *joinFakeRepo) GetOrganizationBySlug(ctx context.Context, slug string) (model.Organization, error) {
	return r.getOrgBySlugFn(ctx, slug)
}

func (r *joinFakeRepo) GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error) {
	return r.getOrgMemberFn(ctx, orgID, userID)
}

func (r *joinFakeRepo) CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error {
	return r.createJoinRequestFn(ctx, orgID, userID)
}

func (r *joinFakeRepo) ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	return r.listJoinRequestsUserFn(ctx, userID)
}

func (r *joinFakeRepo) ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return r.listPendingJoinReqsFn(ctx, orgID)
}

func (r *joinFakeRepo) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return r.acceptJoinRequestFn(ctx, orgID, requestID, deciderID)
}

func (r *joinFakeRepo) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return r.declineJoinRequestFn(ctx, orgID, requestID, deciderID)
}

func newJoinRepo() *joinFakeRepo {
	return &joinFakeRepo{
		stubRepo: stubRepo{calendarFakeRepo: calendarFakeRepo{
			states: make(map[string]model.CalendarOAuthState),
			conns:  make(map[string][]model.CalendarConnection),
		}},
		getOrgBySlugFn: func(_ context.Context, _ string) (model.Organization, error) {
			return model.Organization{ID: joinOrgID, Slug: "acme", Name: "Acme"}, nil
		},
		getOrgMemberFn: func(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
			return model.Member{}, false, nil
		},
		createJoinRequestFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
		listJoinRequestsUserFn: func(_ context.Context, _ uuid.UUID) ([]model.JoinRequestView, error) {
			return []model.JoinRequestView{{OrganizationID: joinOrgID, OrgName: "Acme", Status: "pending"}}, nil
		},
		listPendingJoinReqsFn: func(_ context.Context, _ uuid.UUID) ([]model.JoinRequestAdminView, error) {
			return []model.JoinRequestAdminView{{RequestID: joinReqID}}, nil
		},
		acceptJoinRequestFn: func(_ context.Context, _, _, _ uuid.UUID) error {
			return nil
		},
		declineJoinRequestFn: func(_ context.Context, _, _, _ uuid.UUID) error {
			return nil
		},
	}
}

func buildJoinApp(t *testing.T, repo *joinFakeRepo) *fiber.App {
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
		c.Locals("web_user", model.PlatformUser{ID: joinUserID, Email: "user@example.com"})
		return c.Next()
	})

	app.Post("/me/join-requests", api.WebRequestToJoin)
	app.Get("/me/join-requests", api.WebMyJoinRequests)
	app.Get("/orgs/:id/join-requests", api.OrgJoinRequests)
	app.Post("/orgs/:id/join-requests/:rid/accept", api.AcceptJoinRequest)
	app.Post("/orgs/:id/join-requests/:rid/decline", api.DeclineJoinRequest)

	return app
}

func TestWebRequestToJoin_Pending(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	body, _ := json.Marshal(map[string]string{"slug": "acme"})
	req := httptest.NewRequest(http.MethodPost, "/me/join-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["status"] != "pending" {
		t.Fatalf("expected status=pending, got %+v", result)
	}
	if result["organization_id"] == "" {
		t.Fatal("missing organization_id")
	}
}

func TestWebRequestToJoin_AlreadyMember(t *testing.T) {
	repo := newJoinRepo()
	repo.getOrgMemberFn = func(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
		return model.Member{}, true, nil
	}
	app := buildJoinApp(t, repo)

	body, _ := json.Marshal(map[string]string{"slug": "acme"})
	req := httptest.NewRequest(http.MethodPost, "/me/join-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["already_member"] != true {
		t.Fatalf("expected already_member=true, got %+v", result)
	}
}

func TestWebRequestToJoin_OrgNotFound_Returns404(t *testing.T) {
	repo := newJoinRepo()
	repo.getOrgBySlugFn = func(_ context.Context, _ string) (model.Organization, error) {
		return model.Organization{}, sql.ErrNoRows
	}
	app := buildJoinApp(t, repo)

	body, _ := json.Marshal(map[string]string{"slug": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/me/join-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, b)
	}
}

func TestWebRequestToJoin_MissingSlug_Returns400(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/me/join-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, b)
	}
}

func TestWebMyJoinRequests_List(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	req := httptest.NewRequest(http.MethodGet, "/me/join-requests", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var views []model.JoinRequestView
	if err := json.NewDecoder(resp.Body).Decode(&views); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(views) != 1 || views[0].OrganizationID != joinOrgID {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestOrgJoinRequests_List(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+joinOrgID.String()+"/join-requests", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var views []model.JoinRequestAdminView
	if err := json.NewDecoder(resp.Body).Decode(&views); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(views) != 1 || views[0].RequestID != joinReqID {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestAcceptJoinRequest_Returns204(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	req := httptest.NewRequest(http.MethodPost, "/orgs/"+joinOrgID.String()+"/join-requests/"+joinReqID.String()+"/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, b)
	}
}

func TestDeclineJoinRequest_Returns204(t *testing.T) {
	app := buildJoinApp(t, newJoinRepo())

	req := httptest.NewRequest(http.MethodPost, "/orgs/"+joinOrgID.String()+"/join-requests/"+joinReqID.String()+"/decline", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, b)
	}
}

func TestAcceptJoinRequest_NotFound_Returns404(t *testing.T) {
	repo := newJoinRepo()
	repo.acceptJoinRequestFn = func(_ context.Context, _, _, _ uuid.UUID) error {
		return sql.ErrNoRows
	}
	app := buildJoinApp(t, repo)

	req := httptest.NewRequest(http.MethodPost, "/orgs/"+joinOrgID.String()+"/join-requests/"+joinReqID.String()+"/accept", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, b)
	}
}
