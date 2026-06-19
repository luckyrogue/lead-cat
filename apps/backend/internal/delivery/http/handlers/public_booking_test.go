package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
)

var (
	pubBookingHostID = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	pubBookingOrgID  = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

type publicBookingFakeRepo struct {
	stubRepo
	getBySlugFn           func(ctx context.Context, slug string) (model.BookingEventType, error)
	getPlatformUserByIDFn func(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error)
	listOverlappingFn     func(ctx context.Context, emails []string, from, to time.Time) ([]model.Meeting, error)
	getOrganizationFn     func(ctx context.Context, id uuid.UUID) (model.Organization, error)
}

func (r *publicBookingFakeRepo) GetBookingEventTypeBySlug(ctx context.Context, slug string) (model.BookingEventType, error) {
	return r.getBySlugFn(ctx, slug)
}

func (r *publicBookingFakeRepo) GetPlatformUserByID(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error) {
	return r.getPlatformUserByIDFn(ctx, id)
}

func (r *publicBookingFakeRepo) ListMeetingsOverlapping(ctx context.Context, emails []string, from, to time.Time) ([]model.Meeting, error) {
	return r.listOverlappingFn(ctx, emails, from, to)
}

func (r *publicBookingFakeRepo) GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error) {
	return r.getOrganizationFn(ctx, id)
}

func activeEventType() model.BookingEventType {
	return model.BookingEventType{
		ID:               uuid.MustParse("cccccccc-0000-0000-0000-000000000003"),
		HostUserID:       pubBookingHostID,
		OrganizationID:   pubBookingOrgID,
		Slug:             "intro-call-abc123",
		Title:            "Intro Call",
		Description:      "30 min intro",
		DurationMins:     30,
		Active:           true,
		Timezone:         "Asia/Almaty",
		AvailWeekdays:    []int{1, 2, 3, 4, 5},
		AvailStartMinute: 540,
		AvailEndMinute:   1020,
	}
}

func newPublicBookingRepo() *publicBookingFakeRepo {
	return &publicBookingFakeRepo{
		stubRepo: stubRepo{calendarFakeRepo: calendarFakeRepo{
			states: make(map[string]model.CalendarOAuthState),
			conns:  make(map[string][]model.CalendarConnection),
		}},
		getBySlugFn: func(_ context.Context, _ string) (model.BookingEventType, error) {
			return activeEventType(), nil
		},
		getPlatformUserByIDFn: func(_ context.Context, _ uuid.UUID) (model.PlatformUser, bool, error) {
			return model.PlatformUser{Email: "host@example.com"}, true, nil
		},
		listOverlappingFn: func(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
			return nil, nil
		},
		getOrganizationFn: func(_ context.Context, _ uuid.UUID) (model.Organization, error) {
			return model.Organization{ID: pubBookingOrgID, Name: "Acme Corp"}, nil
		},
	}
}

func buildPublicBookingApp(t *testing.T, repo *publicBookingFakeRepo) *fiber.App {
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
	app.Get("/api/book/:slug", api.PublicBooking)
	return app
}

func TestPublicBooking_Returns200WithEventAndSlots(t *testing.T) {
	app := buildPublicBookingApp(t, newPublicBookingRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/book/intro-call-abc123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var view struct {
		Event struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			DurationMins int    `json:"duration_mins"`
			OrgName      string `json:"org_name"`
			Timezone     string `json:"timezone"`
		} `json:"event"`
		Slots []struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Event.Title != "Intro Call" {
		t.Errorf("unexpected title: %q", view.Event.Title)
	}
	if view.Event.DurationMins != 30 {
		t.Errorf("unexpected duration: %d", view.Event.DurationMins)
	}
	if view.Event.OrgName != "Acme Corp" {
		t.Errorf("unexpected org_name: %q", view.Event.OrgName)
	}
	if view.Event.Timezone != "Asia/Almaty" {
		t.Errorf("unexpected timezone: %q", view.Event.Timezone)
	}
}

func TestPublicBooking_UnknownSlug_Returns404(t *testing.T) {
	repo := newPublicBookingRepo()
	repo.getBySlugFn = func(_ context.Context, _ string) (model.BookingEventType, error) {
		return model.BookingEventType{}, sql.ErrNoRows
	}
	app := buildPublicBookingApp(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/book/no-such-slug", nil)
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

func TestPublicBooking_InactiveEvent_Returns404(t *testing.T) {
	repo := newPublicBookingRepo()
	repo.getBySlugFn = func(_ context.Context, _ string) (model.BookingEventType, error) {
		et := activeEventType()
		et.Active = false
		return et, nil
	}
	app := buildPublicBookingApp(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/book/intro-call-abc123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404 for inactive event, got %d: %s", resp.StatusCode, b)
	}
}

func TestPublicBooking_ResponseHasNoHostEmail(t *testing.T) {
	app := buildPublicBookingApp(t, newPublicBookingRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/book/intro-call-abc123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rawBytes, _ := json.Marshal(raw)
	rawStr := string(rawBytes)
	if contains(rawStr, "host@example.com") {
		t.Errorf("response must NOT contain host email, but got: %s", rawStr)
	}
	if contains(rawStr, "host_user_id") {
		t.Errorf("response must NOT contain host_user_id, but got: %s", rawStr)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
