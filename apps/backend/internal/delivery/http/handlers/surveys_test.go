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

type surveyFakeRepo struct {
	publicBookingFakeRepo
	getSurveyResponseByTokenFn func(ctx context.Context, token string) (model.SurveyResponse, error)
	getSurveyFn                func(ctx context.Context, id uuid.UUID) (model.Survey, error)
	createSurveyResponseFn     func(ctx context.Context, r model.SurveyResponse) (model.SurveyResponse, error)
	completeSurveyResponseFn   func(ctx context.Context, id uuid.UUID, answers []model.Answer) error
}

func (r *surveyFakeRepo) GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error) {
	if r.getSurveyResponseByTokenFn != nil {
		return r.getSurveyResponseByTokenFn(ctx, token)
	}
	return model.SurveyResponse{}, sql.ErrNoRows
}

func (r *surveyFakeRepo) GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error) {
	if r.getSurveyFn != nil {
		return r.getSurveyFn(ctx, id)
	}
	return model.Survey{}, sql.ErrNoRows
}

func (r *surveyFakeRepo) CreateSurveyResponse(ctx context.Context, resp model.SurveyResponse) (model.SurveyResponse, error) {
	if r.createSurveyResponseFn != nil {
		return r.createSurveyResponseFn(ctx, resp)
	}
	resp.Token = "generated-survey-token"
	return resp, nil
}

func (r *surveyFakeRepo) CompleteSurveyResponse(ctx context.Context, id uuid.UUID, answers []model.Answer) error {
	if r.completeSurveyResponseFn != nil {
		return r.completeSurveyResponseFn(ctx, id, answers)
	}
	return nil
}

func newSurveyFakeRepo() *surveyFakeRepo {
	repo := &surveyFakeRepo{
		publicBookingFakeRepo: publicBookingFakeRepo{
			stubRepo: stubRepo{calendarFakeRepo: calendarFakeRepo{
				states: make(map[string]model.CalendarOAuthState),
				conns:  make(map[string][]model.CalendarConnection),
			}},
			getBySlugFn: func(_ context.Context, _ string) (model.BookingEventType, error) {
				return model.BookingEventType{}, sql.ErrNoRows
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
		},
	}
	return repo
}

func buildSurveyApp(t *testing.T, repo *surveyFakeRepo) *fiber.App {
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
	app.Get("/api/survey/:token", api.PublicSurveyGet)
	app.Post("/api/survey/:token", api.PublicSurveySubmit)
	app.Post("/api/book/:slug", api.PublicBookingSubmit)
	return app
}

func TestPublicSurveyGet_UnknownToken(t *testing.T) {
	app := buildSurveyApp(t, newSurveyFakeRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/survey/nope", nil)
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

func TestPublicSurveySubmit_AlreadyCompleted(t *testing.T) {
	repo := newSurveyFakeRepo()
	completedAt := time.Now()
	repo.getSurveyResponseByTokenFn = func(_ context.Context, _ string) (model.SurveyResponse, error) {
		return model.SurveyResponse{
			ID:          uuid.New(),
			Status:      "completed",
			CompletedAt: &completedAt,
		}, nil
	}
	app := buildSurveyApp(t, repo)

	body := `{"answers":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/survey/completed-token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, b)
	}
}

func TestPublicSurveyGet_AlreadyCompleted(t *testing.T) {
	repo := newSurveyFakeRepo()
	completedAt := time.Now()
	repo.getSurveyResponseByTokenFn = func(_ context.Context, _ string) (model.SurveyResponse, error) {
		return model.SurveyResponse{
			ID:          uuid.New(),
			Status:      "completed",
			CompletedAt: &completedAt,
		}, nil
	}
	app := buildSurveyApp(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/survey/completed-token", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, b)
	}
}

func TestDeclineBookingWithActiveSurvey(t *testing.T) {
	repo := newSurveyFakeRepo()

	surveyID := uuid.MustParse("dddddddd-0000-0000-0000-000000000004")
	etID := uuid.MustParse("eeeeeeee-0000-0000-0000-000000000005")

	repo.getBySlugFn = func(_ context.Context, _ string) (model.BookingEventType, error) {
		return model.BookingEventType{
			ID:               etID,
			HostUserID:       pubBookingHostID,
			OrganizationID:   pubBookingOrgID,
			Slug:             "declined-event",
			Title:            "Declined Event",
			DurationMins:     30,
			Active:           true,
			Timezone:         "Asia/Almaty",
			AvailWeekdays:    []int{1, 2, 3, 4, 5},
			AvailStartMinute: 540,
			AvailEndMinute:   1020,
			SurveyID:         &surveyID,
		}, nil
	}

	repo.listOverlappingFn = func(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
		loc, _ := time.LoadLocation("Asia/Almaty")
		slotStart := time.Date(2099, 6, 22, 10, 0, 0, 0, loc).UTC()
		hostID := pubBookingHostID
		return []model.Meeting{{
			ID:              uuid.New(),
			StartsAt:        slotStart,
			EndsAt:          slotStart.Add(30 * time.Minute),
			OrganizerUserID: &hostID,
		}}, nil
	}

	repo.getSurveyFn = func(_ context.Context, _ uuid.UUID) (model.Survey, error) {
		return model.Survey{
			ID:       surveyID,
			Name:     "Post-decline survey",
			IsActive: true,
			Questions: []model.SurveyQuestion{
				{
					ID:       uuid.New(),
					Prompt:   "Why?",
					Type:     model.QuestionText,
					Required: false,
				},
			},
		}, nil
	}

	app := buildSurveyApp(t, repo)

	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2099, 6, 22, 10, 0, 0, 0, loc)
	bodyStr := `{"name":"Bo","email":"bo@example.com","start":"` + start.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/book/declined-event",
		bytes.NewReader([]byte(bodyStr)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
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
	tok, ok := result["survey_token"]
	if !ok {
		t.Fatalf("expected survey_token in body, got: %+v", result)
	}
	if tokStr, _ := tok.(string); tokStr == "" {
		t.Errorf("expected non-empty survey_token, got: %+v", result)
	}
}

func TestDeclineBookingNoSurvey(t *testing.T) {
	repo := newSurveyFakeRepo()

	repo.getBySlugFn = func(_ context.Context, _ string) (model.BookingEventType, error) {
		return model.BookingEventType{
			ID:               uuid.New(),
			HostUserID:       pubBookingHostID,
			OrganizationID:   pubBookingOrgID,
			Slug:             "no-survey-event",
			Title:            "No Survey Event",
			DurationMins:     30,
			Active:           true,
			Timezone:         "Asia/Almaty",
			AvailWeekdays:    []int{1, 2, 3, 4, 5},
			AvailStartMinute: 540,
			AvailEndMinute:   1020,
			SurveyID:         nil,
		}, nil
	}

	repo.listOverlappingFn = func(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
		loc, _ := time.LoadLocation("Asia/Almaty")
		slotStart := time.Date(2099, 6, 22, 10, 0, 0, 0, loc).UTC()
		hostID := pubBookingHostID
		return []model.Meeting{{
			ID:              uuid.New(),
			StartsAt:        slotStart,
			EndsAt:          slotStart.Add(30 * time.Minute),
			OrganizerUserID: &hostID,
		}}, nil
	}

	app := buildSurveyApp(t, repo)

	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2099, 6, 22, 10, 0, 0, 0, loc)
	bodyStr := `{"name":"Bo","email":"bo@example.com","start":"` + start.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/book/no-survey-event",
		bytes.NewReader([]byte(bodyStr)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
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
	if _, ok := result["survey_token"]; ok {
		t.Errorf("expected no survey_token in body when no survey assigned, got: %+v", result)
	}
}
