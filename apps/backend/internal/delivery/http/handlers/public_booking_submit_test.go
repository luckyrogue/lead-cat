package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/stub"
)

func (r *publicBookingFakeRepo) GetUserByID(_ context.Context, id uuid.UUID) (model.User, error) {
	if id == pubBookingHostID {
		return model.User{ID: id, Email: "host@example.com"}, nil
	}
	return model.User{ID: id}, nil
}

func (r *publicBookingFakeRepo) CreateMeeting(_ context.Context, m model.Meeting) (model.Meeting, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return m, nil
}

func (r *publicBookingFakeRepo) AddParticipants(_ context.Context, _ uuid.UUID, _ []model.MeetingParticipant) error {
	return nil
}

func buildPublicBookingSubmitApp(t *testing.T, repo *publicBookingFakeRepo) *fiber.App {
	t.Helper()
	svc := &application.Services{
		Store: repo,
		Log:   zap.NewNop(),
		Commands: &command.Meetings{
			Store:    repo,
			Calendar: stub.NewProvider(),
		},
	}
	api := &handlers.API{App: svc, Log: zap.NewNop()}
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
	app.Post("/api/book/:slug", api.PublicBookingSubmit)
	return app
}

func futureMondayStart() string {
	loc, _ := time.LoadLocation("Asia/Almaty")
	return time.Date(2099, 6, 22, 10, 0, 0, 0, loc).Format(time.RFC3339)
}

func postBooking(t *testing.T, app *fiber.App, slug, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/book/"+slug, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return resp
}

func TestPublicBookingSubmit_Returns200WithMeetLink(t *testing.T) {
	app := buildPublicBookingSubmitApp(t, newPublicBookingRepo())
	body := `{"name":"Visitor V","email":"visitor@example.com","start":"` + futureMondayStart() + `"}`
	resp := postBooking(t, app, "intro-call-abc123", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}
	var conf struct {
		MeetLink string `json:"meet_link"`
		Start    string `json:"start"`
		End      string `json:"end"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&conf); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(conf.MeetLink, "https://meet.google.com/") {
		t.Errorf("expected stub meet link, got %q", conf.MeetLink)
	}
	if conf.Start == "" || conf.End == "" {
		t.Errorf("expected start/end in confirmation, got %+v", conf)
	}
}

func TestPublicBookingSubmit_UnknownSlug_Returns404(t *testing.T) {
	repo := newPublicBookingRepo()
	repo.getBySlugFn = func(_ context.Context, _ string) (model.BookingEventType, error) {
		return model.BookingEventType{}, sql.ErrNoRows
	}
	app := buildPublicBookingSubmitApp(t, repo)
	body := `{"name":"V","email":"visitor@example.com","start":"` + futureMondayStart() + `"}`
	resp := postBooking(t, app, "no-such", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, b)
	}
}

func TestPublicBookingSubmit_Conflict_Returns409(t *testing.T) {
	repo := newPublicBookingRepo()
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2099, 6, 22, 10, 0, 0, 0, loc)
	repo.listOverlappingFn = func(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
		hostID := pubBookingHostID
		return []model.Meeting{{
			ID:              uuid.New(),
			StartsAt:        start.UTC(),
			EndsAt:          start.Add(30 * time.Minute).UTC(),
			OrganizerUserID: &hostID,
		}}, nil
	}
	app := buildPublicBookingSubmitApp(t, repo)
	body := `{"name":"V","email":"visitor@example.com","start":"` + futureMondayStart() + `"}`
	resp := postBooking(t, app, "intro-call-abc123", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, b)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if result["message"] != "slot_taken" {
		t.Errorf("expected message=slot_taken in decline body, got: %+v", result)
	}
}

func TestPublicBookingSubmit_BadEmail_Returns400(t *testing.T) {
	app := buildPublicBookingSubmitApp(t, newPublicBookingRepo())
	body := `{"name":"V","email":"not-an-email","start":"` + futureMondayStart() + `"}`
	resp := postBooking(t, app, "intro-call-abc123", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for bad email, got %d: %s", resp.StatusCode, b)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if result["message"] != "invalid_booking" {
		t.Errorf("expected message=invalid_booking in decline body, got: %+v", result)
	}
}

func TestPublicBookingSubmit_BadBody_Returns400(t *testing.T) {
	app := buildPublicBookingSubmitApp(t, newPublicBookingRepo())
	resp := postBooking(t, app, "intro-call-abc123", `{not json`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for bad body, got %d: %s", resp.StatusCode, b)
	}
}

func TestPublicBookingSubmit_BadStart_Returns400(t *testing.T) {
	app := buildPublicBookingSubmitApp(t, newPublicBookingRepo())
	body := `{"name":"V","email":"visitor@example.com","start":"not-a-time"}`
	resp := postBooking(t, app, "intro-call-abc123", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for bad start, got %d: %s", resp.StatusCode, b)
	}
}
