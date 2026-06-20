package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
)

type fakeCalendarConnector struct {
	token application.CalendarToken
}

func (f *fakeCalendarConnector) Name() string { return "google" }

func (f *fakeCalendarConnector) AuthURL(state, _, _ string) string {
	return "https://accounts.google.com/o/oauth2/auth?state=" + state
}

func (f *fakeCalendarConnector) Exchange(_ context.Context, _, _, _ string) (application.CalendarToken, error) {
	return f.token, nil
}

type calendarFakeRepo struct {
	mu     sync.Mutex
	states map[string]model.CalendarOAuthState
	conns  map[string][]model.CalendarConnection
}

func (r *calendarFakeRepo) CreateCalendarOAuthState(_ context.Context, st model.CalendarOAuthState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[st.State] = st
	return nil
}

func (r *calendarFakeRepo) ConsumeCalendarOAuthState(_ context.Context, state string) (model.CalendarOAuthState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.states[state]
	if !ok {
		return model.CalendarOAuthState{}, sql.ErrNoRows
	}
	delete(r.states, state)
	return st, nil
}

func (r *calendarFakeRepo) UpsertCalendarConnection(_ context.Context, conn model.CalendarConnection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	email := conn.Email
	for i, c := range r.conns[email] {
		if c.Provider == conn.Provider {
			r.conns[email][i] = conn
			return nil
		}
	}
	r.conns[email] = append(r.conns[email], conn)
	return nil
}

func (r *calendarFakeRepo) ListCalendarConnections(_ context.Context, email string) ([]model.CalendarConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]model.CalendarConnection{}, r.conns[email]...), nil
}

func (r *calendarFakeRepo) DeleteCalendarConnection(_ context.Context, email, provider string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cs := r.conns[email]
	out := cs[:0]
	for _, c := range cs {
		if c.Provider != provider {
			out = append(out, c)
		}
	}
	r.conns[email] = out
	return nil
}

type stubRepo struct{ calendarFakeRepo }

func (s *stubRepo) Ping(_ context.Context) error { return nil }
func (s *stubRepo) GetUserByID(_ context.Context, _ uuid.UUID) (model.User, error) {
	return model.User{}, nil
}
func (s *stubRepo) GetUserTelegramID(_ context.Context, _ uuid.UUID) (int64, bool, error) {
	return 0, false, nil
}
func (s *stubRepo) UpsertUserIdentity(_ context.Context, _, _ string) (model.User, error) {
	return model.User{}, nil
}
func (s *stubRepo) UpsertWebIdentity(_ context.Context, _, _, _, _ string) (model.PlatformUser, error) {
	return model.PlatformUser{}, nil
}
func (s *stubRepo) GetPlatformUserByID(_ context.Context, _ uuid.UUID) (model.PlatformUser, bool, error) {
	return model.PlatformUser{}, false, nil
}

func (s *stubRepo) GetPlatformUserLanguageByEmail(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (s *stubRepo) GetPlatformUserIDByTelegramID(_ context.Context, _ int64) (uuid.UUID, bool, error) {
	return uuid.UUID{}, false, nil
}
func (s *stubRepo) LinkTelegram(_ context.Context, _ uuid.UUID, _ int64) error { return nil }
func (s *stubRepo) LinkMemberUserIDsByTelegram(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (s *stubRepo) GetBotUserByTelegramID(_ context.Context, _ int64) (model.BotUser, error) {
	return model.BotUser{}, nil
}
func (s *stubRepo) SetReminderMinutes(_ context.Context, _ int64, _ string) error { return nil }
func (s *stubRepo) SetBotUserPrefs(_ context.Context, _ int64, _, _ string) error { return nil }
func (s *stubRepo) SetPlatformUserPrefs(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (s *stubRepo) GetOrganization(_ context.Context, _ uuid.UUID) (model.Organization, error) {
	return model.Organization{}, nil
}
func (s *stubRepo) CreateOrganization(_ context.Context, _, _ string, _ uuid.UUID) (model.Organization, error) {
	return model.Organization{}, nil
}
func (s *stubRepo) EnsureDefaultOrganizationID(_ context.Context, _, _ string, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
func (s *stubRepo) UpdateOrganization(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (s *stubRepo) ListOrganizationsForUser(_ context.Context, _ uuid.UUID) ([]model.Organization, error) {
	return nil, nil
}
func (s *stubRepo) ListOrganizationsWithGoogle(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}
func (s *stubRepo) LinkChat(_ context.Context, _ uuid.UUID, _ int64) error { return nil }
func (s *stubRepo) GetGoogleConfig(_ context.Context, _ uuid.UUID) ([]byte, string, string, error) {
	return nil, "", "", nil
}
func (s *stubRepo) SetGoogleConfig(_ context.Context, _ uuid.UUID, _ []byte, _, _ string) error {
	return nil
}
func (s *stubRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]model.Member, error) {
	return nil, nil
}
func (s *stubRepo) ListOrgMembers(_ context.Context, _ uuid.UUID) ([]model.Member, error) {
	return nil, nil
}
func (s *stubRepo) AddMember(_ context.Context, _ uuid.UUID, _, _ string) (model.Member, error) {
	return model.Member{}, nil
}
func (s *stubRepo) DeleteMember(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *stubRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *stubRepo) UpdateMemberRole(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (s *stubRepo) CreateInvite(_ context.Context, _ uuid.UUID, _, _ string, _ []byte, _ time.Time, _ uuid.UUID) (model.OrganizationInvite, error) {
	return model.OrganizationInvite{}, nil
}
func (s *stubRepo) ListInvites(_ context.Context, _ uuid.UUID) ([]model.OrganizationInvite, error) {
	return nil, nil
}
func (s *stubRepo) DeleteInvite(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *stubRepo) AcceptInvitesForEmail(_ context.Context, _ string, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubRepo) ListPendingInvitesForEmail(_ context.Context, _ string) ([]model.InviteView, error) {
	return nil, nil
}
func (s *stubRepo) AcceptInvite(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (s *stubRepo) DeclineInvite(_ context.Context, _ uuid.UUID, _ string) error   { return nil }
func (s *stubRepo) ListEmployees(_ context.Context, _ uuid.UUID) ([]model.Employee, error) {
	return nil, nil
}
func (s *stubRepo) SearchEmployeesGlobal(_ context.Context, _ string) ([]model.Employee, error) {
	return nil, nil
}
func (s *stubRepo) GetMeeting(_ context.Context, _, _ uuid.UUID) (model.Meeting, error) {
	return model.Meeting{}, nil
}
func (s *stubRepo) CreateMeeting(_ context.Context, _ model.Meeting) (model.Meeting, error) {
	return model.Meeting{}, nil
}
func (s *stubRepo) UpdateMeeting(_ context.Context, _, _ uuid.UUID, _ model.Meeting) error {
	return nil
}
func (s *stubRepo) UpdateMeetingsTx(_ context.Context, _ uuid.UUID, _ []model.Meeting) error {
	return nil
}
func (s *stubRepo) CancelMeeting(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *stubRepo) ListMeetings(_ context.Context, _ uuid.UUID) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListMeetingsByOrganizer(_ context.Context, _, _ uuid.UUID) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListMeetingsFiltered(_ context.Context, _ uuid.UUID, _ model.MeetingFilter) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListMeetingsByOrganizerTelegram(_ context.Context, _ int64) ([]model.MeetingWithTZ, error) {
	return nil, nil
}
func (s *stubRepo) ListMeetingsOverlapping(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListScheduleForEmail(_ context.Context, _ string, _, _ time.Time) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) CreateMeetingSeries(_ context.Context, _ []model.Meeting, _ []model.MeetingParticipant) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListSeriesOccurrences(_ context.Context, _, _ uuid.UUID, _ time.Time) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListSeriesAllOccurrences(_ context.Context, _, _ uuid.UUID) ([]model.Meeting, error) {
	return nil, nil
}
func (s *stubRepo) ListSeriesOccurrenceStarts(_ context.Context, _, _ uuid.UUID) ([]time.Time, error) {
	return nil, nil
}
func (s *stubRepo) CancelSeriesOccurrences(_ context.Context, _, _ uuid.UUID, _ time.Time) (int, error) {
	return 0, nil
}
func (s *stubRepo) CancelAllSeriesOccurrences(_ context.Context, _, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubRepo) SetSeriesRecurrenceUntil(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (s *stubRepo) AddParticipants(_ context.Context, _ uuid.UUID, _ []model.MeetingParticipant) error {
	return nil
}
func (s *stubRepo) RemoveParticipant(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (s *stubRepo) ListParticipants(_ context.Context, _ uuid.UUID) ([]model.MeetingParticipant, error) {
	return nil, nil
}
func (s *stubRepo) InsertAuditEntry(_ context.Context, _ model.AuditEntry) error { return nil }
func (s *stubRepo) ListAuditEntries(_ context.Context, _ model.AuditFilter) ([]model.AuditEntry, error) {
	return nil, nil
}
func (s *stubRepo) InsertMagicLink(_ context.Context, _ string, _ []byte, _ time.Time) error {
	return nil
}
func (s *stubRepo) ConsumeMagicLink(_ context.Context, _ []byte, _ time.Time) (string, bool, error) {
	return "", false, nil
}
func (s *stubRepo) CreateWebSession(_ context.Context, _ []byte, _ uuid.UUID, _ time.Time, _, _ string) (model.WebSession, error) {
	return model.WebSession{}, nil
}
func (s *stubRepo) ResolveWebSession(_ context.Context, _ []byte, _ time.Time) (model.WebSession, bool, error) {
	return model.WebSession{}, false, nil
}
func (s *stubRepo) TouchWebSession(_ context.Context, _ uuid.UUID, _, _ time.Time) error {
	return nil
}
func (s *stubRepo) RevokeWebSession(_ context.Context, _ []byte, _ time.Time) error { return nil }
func (s *stubRepo) GetOrganizationBySlug(_ context.Context, _ string) (model.Organization, error) {
	return model.Organization{}, nil
}
func (s *stubRepo) GetOrgMember(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
	return model.Member{}, false, nil
}
func (s *stubRepo) CreateJoinRequest(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *stubRepo) ListJoinRequestsForUser(_ context.Context, _ uuid.UUID) ([]model.JoinRequestView, error) {
	return nil, nil
}
func (s *stubRepo) ListPendingJoinRequests(_ context.Context, _ uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return nil, nil
}
func (s *stubRepo) AcceptJoinRequest(_ context.Context, _, _, _ uuid.UUID) error  { return nil }
func (s *stubRepo) DeclineJoinRequest(_ context.Context, _, _, _ uuid.UUID) error { return nil }
func (s *stubRepo) CreateBookingEventType(_ context.Context, et model.BookingEventType) (model.BookingEventType, error) {
	return et, nil
}
func (s *stubRepo) GetBookingEventType(_ context.Context, _ uuid.UUID) (model.BookingEventType, error) {
	return model.BookingEventType{}, nil
}
func (s *stubRepo) GetBookingEventTypeBySlug(_ context.Context, _ string) (model.BookingEventType, error) {
	return model.BookingEventType{}, nil
}
func (s *stubRepo) ListBookingEventTypesForUser(_ context.Context, _ uuid.UUID) ([]model.BookingEventType, error) {
	return nil, nil
}
func (s *stubRepo) UpdateBookingEventType(_ context.Context, _ model.BookingEventType) error {
	return nil
}
func (s *stubRepo) DeleteBookingEventType(_ context.Context, _ uuid.UUID) error { return nil }

func buildFakeServices(t *testing.T, repo *stubRepo, connector *fakeCalendarConnector) *application.Services {
	t.Helper()
	svc := &application.Services{
		Store: repo,
		Log:   zap.NewNop(),
	}
	svc.ConfigureCalendarConnectors(map[string]application.CalendarConnector{
		"google": connector,
	})
	svc.ConfigureWebAuth(nil, nil, "http://localhost", "", time.Hour, time.Hour)
	return svc
}

func buildTestApp(t *testing.T, svc *application.Services, email string) *fiber.App {
	t.Helper()
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
		c.Locals("web_user", model.PlatformUser{Email: email})
		return c.Next()
	})

	app.Post("/api/calendar/connect/:provider/start", api.CalendarConnectStart)
	app.Get("/api/calendar/connect/:provider/callback", api.CalendarConnectCallback)
	app.Get("/api/calendar/connections", api.CalendarConnectionsList)
	app.Delete("/api/calendar/connections/:provider", api.CalendarDisconnect)

	return app
}

func TestCalendarConnect_Start_ReturnsAuthURL(t *testing.T) {
	repo := &stubRepo{calendarFakeRepo: calendarFakeRepo{
		states: make(map[string]model.CalendarOAuthState),
		conns:  make(map[string][]model.CalendarConnection),
	}}
	connector := &fakeCalendarConnector{
		token: application.CalendarToken{
			AccessToken: "at", RefreshToken: "rt",
			Expiry: time.Now().Add(time.Hour), Scopes: "calendar",
		},
	}
	svc := buildFakeServices(t, repo, connector)
	app := buildTestApp(t, svc, "u@x.com")

	req := httptest.NewRequest(http.MethodPost, "/api/calendar/connect/google/start", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AuthURL string `json:"auth_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.AuthURL == "" {
		t.Fatal("auth_url is empty")
	}
	repo.mu.Lock()
	stateCount := len(repo.states)
	repo.mu.Unlock()
	if stateCount == 0 {
		t.Fatal("no pending state row was created")
	}
}

func TestCalendarConnect_Callback_PersistsConnection(t *testing.T) {
	repo := &stubRepo{calendarFakeRepo: calendarFakeRepo{
		states: make(map[string]model.CalendarOAuthState),
		conns:  make(map[string][]model.CalendarConnection),
	}}
	connector := &fakeCalendarConnector{
		token: application.CalendarToken{
			AccessToken: "at", RefreshToken: "rt",
			Expiry: time.Now().Add(time.Hour), Scopes: "calendar",
		},
	}
	svc := buildFakeServices(t, repo, connector)
	app := buildTestApp(t, svc, "u@x.com")

	startReq := httptest.NewRequest(http.MethodPost, "/api/calendar/connect/google/start", nil)
	startResp, err := app.Test(startReq, -1)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	startResp.Body.Close()
	if startResp.StatusCode != http.StatusOK {
		t.Fatalf("start: expected 200, got %d", startResp.StatusCode)
	}

	repo.mu.Lock()
	var capturedState string
	for k := range repo.states {
		capturedState = k
	}
	repo.mu.Unlock()

	if capturedState == "" {
		t.Fatal("no state captured from start")
	}

	callbackReq := httptest.NewRequest(http.MethodGet,
		"/api/calendar/connect/google/callback?state="+capturedState+"&code=fake-code", nil)
	callbackResp, err := app.Test(callbackReq, -1)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer callbackResp.Body.Close()
	if callbackResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(callbackResp.Body)
		t.Fatalf("callback: expected 200, got %d: %s", callbackResp.StatusCode, body)
	}

	ct := callbackResp.Header.Get("Content-Type")
	if ct == "" {
		t.Fatal("Content-Type not set on callback response")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/calendar/connections", nil)
	listResp, err := app.Test(listReq, -1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("list: expected 200, got %d: %s", listResp.StatusCode, body)
	}

	var views []application.CalendarConnectionView
	if err := json.NewDecoder(listResp.Body).Decode(&views); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(views) == 0 {
		t.Fatal("expected at least one connection")
	}
	if views[0].Provider != "google" || !views[0].Connected {
		t.Fatalf("unexpected view: %+v", views[0])
	}
}

func TestCalendarConnect_Callback_BadState_Returns400(t *testing.T) {
	repo := &stubRepo{calendarFakeRepo: calendarFakeRepo{
		states: make(map[string]model.CalendarOAuthState),
		conns:  make(map[string][]model.CalendarConnection),
	}}
	connector := &fakeCalendarConnector{}
	svc := buildFakeServices(t, repo, connector)
	app := buildTestApp(t, svc, "u@x.com")

	req := httptest.NewRequest(http.MethodGet,
		"/api/calendar/connect/google/callback?state=bogus&code=x", nil)
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
