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
	bookingUserID = uuid.MustParse("11111111-0000-0000-0000-000000000001")
	bookingOrgID  = uuid.MustParse("22222222-0000-0000-0000-000000000002")
	bookingETID   = uuid.MustParse("33333333-0000-0000-0000-000000000003")
)

type bookingFakeRepo struct {
	stubRepo
	getPlatformUserFn        func(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error)
	getOrgMemberFn           func(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	createBookingEventTypeFn func(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error)
	getBookingEventTypeFn    func(ctx context.Context, id uuid.UUID) (model.BookingEventType, error)
	listBookingEventTypesFn  func(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error)
	updateBookingEventTypeFn func(ctx context.Context, et model.BookingEventType) error
	deleteBookingEventTypeFn func(ctx context.Context, id uuid.UUID) error
}

func (r *bookingFakeRepo) GetPlatformUserByID(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error) {
	return r.getPlatformUserFn(ctx, id)
}

func (r *bookingFakeRepo) GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error) {
	return r.getOrgMemberFn(ctx, orgID, userID)
}

func (r *bookingFakeRepo) CreateBookingEventType(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error) {
	return r.createBookingEventTypeFn(ctx, et)
}

func (r *bookingFakeRepo) GetBookingEventType(ctx context.Context, id uuid.UUID) (model.BookingEventType, error) {
	return r.getBookingEventTypeFn(ctx, id)
}

func (r *bookingFakeRepo) ListBookingEventTypesForUser(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	return r.listBookingEventTypesFn(ctx, hostUserID)
}

func (r *bookingFakeRepo) UpdateBookingEventType(ctx context.Context, et model.BookingEventType) error {
	return r.updateBookingEventTypeFn(ctx, et)
}

func (r *bookingFakeRepo) DeleteBookingEventType(ctx context.Context, id uuid.UUID) error {
	return r.deleteBookingEventTypeFn(ctx, id)
}

func newBookingRepo() *bookingFakeRepo {
	return &bookingFakeRepo{
		stubRepo: stubRepo{calendarFakeRepo: calendarFakeRepo{
			states: make(map[string]model.CalendarOAuthState),
			conns:  make(map[string][]model.CalendarConnection),
		}},
		getPlatformUserFn: func(_ context.Context, _ uuid.UUID) (model.PlatformUser, bool, error) {
			return model.PlatformUser{Timezone: "Asia/Almaty"}, true, nil
		},
		getOrgMemberFn: func(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
			return model.Member{}, true, nil
		},
		createBookingEventTypeFn: func(_ context.Context, et model.BookingEventType) (model.BookingEventType, error) {
			et.ID = bookingETID
			return et, nil
		},
		getBookingEventTypeFn: func(_ context.Context, _ uuid.UUID) (model.BookingEventType, error) {
			return model.BookingEventType{
				ID:               bookingETID,
				HostUserID:       bookingUserID,
				OrganizationID:   bookingOrgID,
				Slug:             "test-abc123",
				Title:            "Test Event",
				DurationMins:     30,
				Timezone:         "Asia/Almaty",
				AvailWeekdays:    []int{1, 2, 3, 4, 5},
				AvailStartMinute: 540,
				AvailEndMinute:   1080,
				Active:           true,
			}, nil
		},
		listBookingEventTypesFn: func(_ context.Context, _ uuid.UUID) ([]model.BookingEventType, error) {
			return []model.BookingEventType{
				{ID: bookingETID, Title: "Test Event", Slug: "test-abc123"},
			}, nil
		},
		updateBookingEventTypeFn: func(_ context.Context, _ model.BookingEventType) error {
			return nil
		},
		deleteBookingEventTypeFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}
}

func buildBookingApp(t *testing.T, repo *bookingFakeRepo) *fiber.App {
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
		c.Locals("web_user", model.PlatformUser{ID: bookingUserID, Email: "host@example.com"})
		return c.Next()
	})

	app.Get("/api/booking/event-types", api.BookingListEventTypes)
	app.Post("/api/booking/event-types", api.BookingCreateEventType)
	app.Patch("/api/booking/event-types/:id", api.BookingUpdateEventType)
	app.Delete("/api/booking/event-types/:id", api.BookingDeleteEventType)

	return app
}

func validCreateBody() map[string]interface{} {
	return map[string]interface{}{
		"title":              "Intro Call",
		"description":        "30 min intro",
		"duration_mins":      30,
		"timezone":           "",
		"avail_weekdays":     []int{1, 2, 3, 4, 5},
		"avail_start_minute": 540,
		"avail_end_minute":   1080,
		"active":             true,
	}
}

func TestBookingListEventTypes(t *testing.T) {
	app := buildBookingApp(t, newBookingRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/booking/event-types", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	var list []model.BookingEventType
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0].ID != bookingETID {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestBookingCreateEventType_Member_Returns201(t *testing.T) {
	app := buildBookingApp(t, newBookingRepo())

	body, _ := json.Marshal(validCreateBody())
	req := httptest.NewRequest(http.MethodPost, "/api/booking/event-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Org-Id", bookingOrgID.String())

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}

	var et model.BookingEventType
	if err := json.NewDecoder(resp.Body).Decode(&et); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if et.Slug == "" {
		t.Fatal("slug should not be empty")
	}
	if et.Timezone == "" {
		t.Fatal("timezone should be defaulted")
	}
}

func TestBookingCreateEventType_NotMember_Returns403(t *testing.T) {
	repo := newBookingRepo()
	repo.getOrgMemberFn = func(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
		return model.Member{}, false, nil
	}
	app := buildBookingApp(t, repo)

	body, _ := json.Marshal(validCreateBody())
	req := httptest.NewRequest(http.MethodPost, "/api/booking/event-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Org-Id", bookingOrgID.String())

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, b)
	}
}

func TestBookingCreateEventType_BadInput_Returns400(t *testing.T) {
	app := buildBookingApp(t, newBookingRepo())

	bad := validCreateBody()
	bad["duration_mins"] = 0
	body, _ := json.Marshal(bad)
	req := httptest.NewRequest(http.MethodPost, "/api/booking/event-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Org-Id", bookingOrgID.String())

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

func TestBookingUpdateEventType_Owner_Returns200(t *testing.T) {
	app := buildBookingApp(t, newBookingRepo())

	body, _ := json.Marshal(validCreateBody())
	req := httptest.NewRequest(http.MethodPatch, "/api/booking/event-types/"+bookingETID.String(), bytes.NewReader(body))
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
}

func TestBookingUpdateEventType_OtherHost_Returns403(t *testing.T) {
	repo := newBookingRepo()
	otherUser := uuid.MustParse("99999999-0000-0000-0000-000000000009")
	repo.getBookingEventTypeFn = func(_ context.Context, _ uuid.UUID) (model.BookingEventType, error) {
		return model.BookingEventType{
			ID:               bookingETID,
			HostUserID:       otherUser,
			OrganizationID:   bookingOrgID,
			Title:            "Other's event",
			DurationMins:     30,
			Timezone:         "UTC",
			AvailWeekdays:    []int{1},
			AvailStartMinute: 0,
			AvailEndMinute:   60,
		}, nil
	}
	app := buildBookingApp(t, repo)

	body, _ := json.Marshal(validCreateBody())
	req := httptest.NewRequest(http.MethodPatch, "/api/booking/event-types/"+bookingETID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, b)
	}
}

func TestBookingDeleteEventType_Returns204(t *testing.T) {
	app := buildBookingApp(t, newBookingRepo())

	req := httptest.NewRequest(http.MethodDelete, "/api/booking/event-types/"+bookingETID.String(), nil)
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

func TestBookingDeleteEventType_OtherHost_Returns403(t *testing.T) {
	repo := newBookingRepo()
	otherUser := uuid.MustParse("99999999-0000-0000-0000-000000000009")
	repo.getBookingEventTypeFn = func(_ context.Context, _ uuid.UUID) (model.BookingEventType, error) {
		return model.BookingEventType{
			ID:               bookingETID,
			HostUserID:       otherUser,
			OrganizationID:   bookingOrgID,
			Title:            "Other's event",
			DurationMins:     30,
			Timezone:         "UTC",
			AvailWeekdays:    []int{1},
			AvailStartMinute: 0,
			AvailEndMinute:   60,
		}, nil
	}
	app := buildBookingApp(t, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/booking/event-types/"+bookingETID.String(), nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, b)
	}
}

func TestBookingUpdateEventType_NotFound_Returns404(t *testing.T) {
	repo := newBookingRepo()
	repo.getBookingEventTypeFn = func(_ context.Context, _ uuid.UUID) (model.BookingEventType, error) {
		return model.BookingEventType{}, sql.ErrNoRows
	}
	app := buildBookingApp(t, repo)

	body, _ := json.Marshal(validCreateBody())
	req := httptest.NewRequest(http.MethodPatch, "/api/booking/event-types/"+bookingETID.String(), bytes.NewReader(body))
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
