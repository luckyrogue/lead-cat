# Slice 1b — Microsoft Graph Calendar Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Microsoft a real calendar source — connect via OAuth, create/update/cancel the organizer's meetings on their MS calendar as Teams online meetings, resolve provider per organizer (MS / Google / SA), and read MS free/busy (for 1c).

**Architecture:** A new `infrastructure/calendar/microsoft` package implements the provider-neutral `docalendar.Service` over raw Graph REST on an `oauth2`-authed `*http.Client` (base URL injectable for httptest). A new MS `CalendarConnector` (mirroring 1a's Google one) plugs into the already-`:provider`-parameterized connect flow. A composite resolver in `infrastructure/calendar/resolver` makes `CalendarProvider.For` pick MS / Google / SA by most-recently-updated connection.

**Tech Stack:** Go 1.26, `golang.org/x/oauth2`, `net/http` + `encoding/json` (no msgraph SDK), httptest; React Router v7 / shadcn frontends (`apps/admin`, `apps/mini-app`).

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-18-slice-1b-microsoft-calendar-adapter-design.md`.
- depguard: `application` imports zero `internal/infrastructure`; the MS calendar pkg must NOT import the oauth pkg (use local interfaces, wire concretes in `main.go`); the resolver pkg implements `application.CalendarProvider` structurally (return `docalendar.Service`).
- No code comments in new Go/TS files. gofmt every Go file; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues.
- Provider string is the literal `"microsoft"`. MS meetings are **Teams** (`isOnlineMeeting:true`, `onlineMeetingProvider:"teamsForBusiness"`); `MeetLink` = `onlineMeeting.joinUrl`.
- **MS OAuth differs from Google:** refresh tokens come from the **`offline_access` scope** (NOT `oauth2.AccessTypeOffline`); consent via `oauth2.SetAuthURLParam("prompt","consent")` (NOT `oauth2.ApprovalForce`). Endpoint: `https://login.microsoftonline.com/common/oauth2/v2.0/{authorize,token}`.
- SA-fallback invariant (from 1a) is sacred: no per-user path may hard-fail meeting create/update/cancel — any MS error falls through to the Google/SA path.
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; add i18n keys to ALL THREE dicts (en/ru/kk); admin formal "вы", mini-app informal "ты". Never run repo-wide prettier; keep edits additive (watch the known `entities/meeting/api.ts` reflow gotcha).
- Work on `main`; never `git add -A` (stage explicit paths); **verify `HEAD` before each commit** (the user commits in parallel). Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference — 1a's Google connector to mirror** (`internal/infrastructure/oauth/google/calendar_connector.go`): struct `{clientID, clientSecret string; endpoint oauth2.Endpoint}`; methods `Name`, `OAuthConfig(redirectURL)`, `cfg(redirectURL)`, `AuthURL`, `Exchange`; `var _ application.CalendarConnector = (*CalendarConnector)(nil)`. `application.CalendarToken{AccessToken,RefreshToken string; Expiry time.Time; Scopes string}`.

**Reference — domain `docalendar` (`internal/domain/calendar/calendar.go`):**
```go
type CalendarEvent struct { Title, Description string; Start, End time.Time; AttendeeEmails []string }
type CalendarResult struct { EventID, MeetLink string }
type Service interface {
    CreateEvent(ctx, CalendarEvent) (CalendarResult, error)
    UpdateEvent(ctx, eventID string, e CalendarEvent) error
    UpdateAttendees(ctx, eventID string, emails []string) error
    DeleteEvent(ctx, eventID string) error
}
```

---

### Task 1: Microsoft calendar OAuth connector

**Files:**
- Create: `apps/backend/internal/infrastructure/oauth/microsoft/calendar_connector.go`
- Test: `apps/backend/internal/infrastructure/oauth/microsoft/calendar_connector_test.go`

**Interfaces:**
- Produces: `microsoft.NewCalendarConnector(clientID, clientSecret string) *CalendarConnector` implementing `application.CalendarConnector` (`Name`→`"microsoft"`, `AuthURL`, `Exchange`) + `OAuthConfig(redirectURL string) *oauth2.Config`. Endpoint field injectable for tests.
- Consumes: `golang.org/x/oauth2`, `application.CalendarToken`.

- [ ] **Step 1: Write the failing test** — `calendar_connector_test.go` (`package microsoft`):
```go
func TestCalendarConnector_AuthURL(t *testing.T) {
	c := NewCalendarConnector("cid", "secret")
	u := c.AuthURL("st", "chal", "https://app.example.com/cb")
	q, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	v := q.Query()
	if v.Get("prompt") != "consent" {
		t.Errorf("prompt=%q want consent", v.Get("prompt"))
	}
	if v.Get("code_challenge") != "chal" || v.Get("code_challenge_method") != "S256" {
		t.Errorf("pkce missing: %v", v)
	}
	if v.Get("state") != "st" {
		t.Errorf("state=%q", v.Get("state"))
	}
	scope := v.Get("scope")
	for _, want := range []string{"Calendars.ReadWrite", "OnlineMeetings.ReadWrite", "offline_access"} {
		if !strings.Contains(scope, want) {
			t.Errorf("scope %q missing %q", scope, want)
		}
	}
}
```

- [ ] **Step 2: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/oauth/microsoft/ -run TestCalendarConnector -v`

- [ ] **Step 3: Implement** — `calendar_connector.go`:
```go
package microsoft

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application"
)

var msEndpoint = oauth2.Endpoint{
	AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
	TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
}

type CalendarConnector struct {
	clientID, clientSecret string
	endpoint               oauth2.Endpoint
}

func NewCalendarConnector(clientID, clientSecret string) *CalendarConnector {
	return &CalendarConnector{clientID: clientID, clientSecret: clientSecret, endpoint: msEndpoint}
}

func (c *CalendarConnector) Name() string { return "microsoft" }

func (c *CalendarConnector) OAuthConfig(redirectURL string) *oauth2.Config { return c.cfg(redirectURL) }

func (c *CalendarConnector) cfg(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     c.endpoint,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://graph.microsoft.com/Calendars.ReadWrite",
			"https://graph.microsoft.com/OnlineMeetings.ReadWrite",
			"offline_access", "openid", "email", "profile",
		},
	}
}

func (c *CalendarConnector) AuthURL(state, challenge, redirectURL string) string {
	return c.cfg(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (c *CalendarConnector) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.CalendarToken, error) {
	tok, err := c.cfg(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return application.CalendarToken{}, err
	}
	scopes, _ := tok.Extra("scope").(string)
	return application.CalendarToken{AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, Expiry: tok.Expiry, Scopes: scopes}, nil
}

var _ application.CalendarConnector = (*CalendarConnector)(nil)
```

- [ ] **Step 4: Add an Exchange httptest** — append a test that sets `c.endpoint.TokenURL = srv.URL`, where `srv` returns `{"access_token":"at","refresh_token":"rt","expires_in":3600,"scope":"...Calendars.ReadWrite offline_access","token_type":"Bearer"}`; assert `Exchange` returns those tokens + scopes. (Mirror 1a's google `calendar_connector_test.go` Exchange test.)

- [ ] **Step 5: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/oauth/microsoft/ -v`

- [ ] **Step 6: gofmt + commit**
```bash
gofmt -w internal/infrastructure/oauth/microsoft/calendar_connector.go internal/infrastructure/oauth/microsoft/calendar_connector_test.go
git add apps/backend/internal/infrastructure/oauth/microsoft/calendar_connector.go apps/backend/internal/infrastructure/oauth/microsoft/calendar_connector_test.go
git commit -m "feat(calendar/ms): Microsoft calendar OAuth connector"
```

---

### Task 2: MS event adapter (Teams) + free/busy `BusyReader`

**Files:**
- Create: `apps/backend/internal/infrastructure/calendar/microsoft/adapter.go`
- Create: `apps/backend/internal/infrastructure/calendar/microsoft/graph.go` (typed request/response structs + HTTP helper)
- Modify: `apps/backend/internal/domain/calendar/calendar.go` — add `Interval` + `BusyReader`
- Test: `apps/backend/internal/infrastructure/calendar/microsoft/adapter_test.go`

**Interfaces:**
- Produces:
  - `microsoft.newAdapter(httpClient *http.Client, baseURL string) *adapter` (unexported; constructed by the factory in Task 3) implementing `docalendar.Service`.
  - `(*adapter).BusyTimes(ctx, emails []string, from, to time.Time) (map[string][]docalendar.Interval, error)`.
  - `docalendar.Interval{ Start, End time.Time }`; `docalendar.BusyReader interface { BusyTimes(ctx, emails []string, from, to time.Time) (map[string][]Interval, error) }`.
- Consumes: `docalendar.CalendarEvent`, `docalendar.CalendarResult`.

- [ ] **Step 1: Add the domain interface** — append to `internal/domain/calendar/calendar.go`:
```go
type Interval struct {
	Start time.Time
	End   time.Time
}

type BusyReader interface {
	BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]Interval, error)
}
```

- [ ] **Step 2: Write the failing adapter test** — `adapter_test.go` (`package microsoft`):
```go
func newTestAdapter(t *testing.T, h http.HandlerFunc) (*adapter, *httptest.Server) {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newAdapter(srv.Client(), srv.URL), srv
}

func TestCreateEvent_Teams(t *testing.T) {
	var gotPath, gotBody string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"evt1","onlineMeeting":{"joinUrl":"https://teams.microsoft.com/l/xyz"}}`))
	})
	res, err := a.CreateEvent(context.Background(), docalendar.CalendarEvent{
		Title: "Sync", Description: "d",
		Start: time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 20, 9, 30, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.EventID != "evt1" || res.MeetLink != "https://teams.microsoft.com/l/xyz" {
		t.Fatalf("bad result: %+v", res)
	}
	if gotPath != "/me/events" {
		t.Errorf("path=%q", gotPath)
	}
	for _, want := range []string{`"isOnlineMeeting":true`, `"teamsForBusiness"`, `"a@x.com"`, `"Sync"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body missing %q: %s", want, gotBody)
		}
	}
}

func TestCreateEvent_GraphError(t *testing.T) {
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":{"code":"ErrorAccessDenied","message":"no"}}`))
	})
	if _, err := a.CreateEvent(context.Background(), docalendar.CalendarEvent{}); err == nil {
		t.Fatal("expected error on 403")
	}
}

func TestUpdateEvent_Patch(t *testing.T) {
	var m, p string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) { m, p = r.Method, r.URL.Path })
	if err := a.UpdateEvent(context.Background(), "evt1", docalendar.CalendarEvent{Title: "X",
		Start: time.Now().UTC(), End: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if m != http.MethodPatch || p != "/me/events/evt1" {
		t.Fatalf("got %s %s", m, p)
	}
}

func TestDeleteEvent(t *testing.T) {
	var m, p string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) { m, p = r.Method, r.URL.Path; w.WriteHeader(204) })
	if err := a.DeleteEvent(context.Background(), "evt1"); err != nil {
		t.Fatal(err)
	}
	if m != http.MethodDelete || p != "/me/events/evt1" {
		t.Fatalf("got %s %s", m, p)
	}
}

func TestBusyTimes(t *testing.T) {
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendar/getSchedule" {
			t.Errorf("path=%q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"value":[{"scheduleId":"a@x.com","scheduleItems":[{"status":"busy","start":{"dateTime":"2026-06-20T09:00:00.0000000","timeZone":"UTC"},"end":{"dateTime":"2026-06-20T09:30:00.0000000","timeZone":"UTC"}}]}]}`))
	})
	busy, err := a.BusyTimes(context.Background(), []string{"a@x.com"},
		time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(busy["a@x.com"]) != 1 {
		t.Fatalf("expected 1 busy block, got %v", busy)
	}
}
```

- [ ] **Step 3: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/calendar/microsoft/ -v`

- [ ] **Step 4: Implement** — `graph.go` (structs + a `doJSON` helper that marshals a body, sends `method base+path`, and on non-2xx returns `fmt.Errorf("graph %s: %s", resp.Status, errBody.Error.Code)`), and `adapter.go`:
```go
package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

const graphTimeLayout = "2006-01-02T15:04:05"

type adapter struct {
	httpClient *http.Client
	baseURL    string
}

func newAdapter(httpClient *http.Client, baseURL string) *adapter {
	return &adapter{httpClient: httpClient, baseURL: baseURL}
}

func graphTime(t time.Time) graphDateTime {
	return graphDateTime{DateTime: t.UTC().Format(graphTimeLayout), TimeZone: "UTC"}
}

func attendees(emails []string) []graphAttendee {
	out := make([]graphAttendee, 0, len(emails))
	for _, e := range emails {
		out = append(out, graphAttendee{EmailAddress: graphEmail{Address: e}, Type: "required"})
	}
	return out
}

func (a *adapter) CreateEvent(ctx context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	body := graphEvent{
		Subject:               e.Title,
		Body:                  &graphBody{ContentType: "text", Content: e.Description},
		Start:                 graphTime(e.Start),
		End:                   graphTime(e.End),
		Attendees:             attendees(e.AttendeeEmails),
		IsOnlineMeeting:       true,
		OnlineMeetingProvider: "teamsForBusiness",
	}
	var resp graphEvent
	if err := a.doJSON(ctx, http.MethodPost, "/me/events", body, &resp); err != nil {
		return docalendar.CalendarResult{}, err
	}
	link := ""
	if resp.OnlineMeeting != nil {
		link = resp.OnlineMeeting.JoinURL
	}
	return docalendar.CalendarResult{EventID: resp.ID, MeetLink: link}, nil
}

func (a *adapter) UpdateEvent(ctx context.Context, eventID string, e docalendar.CalendarEvent) error {
	body := graphEvent{Subject: e.Title, Body: &graphBody{ContentType: "text", Content: e.Description}, Start: graphTime(e.Start), End: graphTime(e.End)}
	return a.doJSON(ctx, http.MethodPatch, "/me/events/"+eventID, body, nil)
}

func (a *adapter) UpdateAttendees(ctx context.Context, eventID string, emails []string) error {
	return a.doJSON(ctx, http.MethodPatch, "/me/events/"+eventID, map[string]any{"attendees": attendees(emails)}, nil)
}

func (a *adapter) DeleteEvent(ctx context.Context, eventID string) error {
	return a.doJSON(ctx, http.MethodDelete, "/me/events/"+eventID, nil, nil)
}

func (a *adapter) BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]docalendar.Interval, error) {
	body := map[string]any{
		"schedules":                emails,
		"startTime":                graphTime(from),
		"endTime":                  graphTime(to),
		"availabilityViewInterval": 30,
	}
	var resp graphScheduleResponse
	if err := a.doJSON(ctx, http.MethodPost, "/me/calendar/getSchedule", body, &resp); err != nil {
		return nil, err
	}
	out := make(map[string][]docalendar.Interval, len(resp.Value))
	for _, s := range resp.Value {
		for _, it := range s.ScheduleItems {
			start, _ := time.Parse(graphTimeLayout, trimFraction(it.Start.DateTime))
			end, _ := time.Parse(graphTimeLayout, trimFraction(it.End.DateTime))
			out[s.ScheduleID] = append(out[s.ScheduleID], docalendar.Interval{Start: start, End: end})
		}
	}
	return out, nil
}

func (a *adapter) doJSON(ctx context.Context, method, path string, in, out any) error {
	var reader io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reader)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var ge graphErrorEnvelope
		raw, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(raw, &ge)
		return fmt.Errorf("graph %s: %s", resp.Status, ge.Error.Code)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

var _ docalendar.Service = (*adapter)(nil)
var _ docalendar.BusyReader = (*adapter)(nil)
```
In `graph.go` add `trimFraction(s string) string` (Graph returns `...09:00:00.0000000`; strip a trailing `.` + digits so the `graphTimeLayout` parse succeeds) and all the structs: `graphDateTime{DateTime,TimeZone string}`, `graphEmail{Address string}`, `graphAttendee{EmailAddress graphEmail; Type string}`, `graphBody{ContentType,Content string}`, `graphEvent{ID string \`json:"id,omitempty"\`; Subject string; Body *graphBody; Start,End graphDateTime; Attendees []graphAttendee; IsOnlineMeeting bool \`json:"isOnlineMeeting,omitempty"\`; OnlineMeetingProvider string \`json:"onlineMeetingProvider,omitempty"\`; OnlineMeeting *struct{JoinURL string \`json:"joinUrl"\`} \`json:"onlineMeeting,omitempty"\`}`, `graphScheduleResponse{Value []struct{ScheduleID string \`json:"scheduleId"\`; ScheduleItems []struct{Status string; Start,End graphDateTime}}}`, `graphErrorEnvelope{Error struct{Code,Message string}}`. Use exact JSON tags matching Graph (`subject`, `body`, `start`, `end`, `attendees`, `emailAddress`, `address`, `type`, `contentType`, `content`, `dateTime`, `timeZone`).

- [ ] **Step 5: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/calendar/microsoft/ -v`

- [ ] **Step 6: gofmt + lint + commit**
```bash
gofmt -w internal/infrastructure/calendar/microsoft/*.go internal/domain/calendar/calendar.go
golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/calendar/microsoft/... ./internal/domain/calendar/...
git add apps/backend/internal/infrastructure/calendar/microsoft/ apps/backend/internal/domain/calendar/calendar.go
git commit -m "feat(calendar/ms): Graph event adapter (Teams) + free/busy BusyReader"
```

---

### Task 3: Composite resolver + MS factory + token source + main.go wiring

**Files:**
- Create: `apps/backend/internal/infrastructure/calendar/microsoft/factory.go` (+ `usersource.go` — self-persisting token source)
- Create: `apps/backend/internal/infrastructure/calendar/resolver/resolver.go`
- Test: `apps/backend/internal/infrastructure/calendar/resolver/resolver_test.go`
- Modify: `apps/backend/cmd/server/main.go` (wire MS connector + factory + resolver; register `connectors["microsoft"]`)

**Interfaces:**
- Produces:
  - `microsoft.NewFactory(conns connStore, connector oauthConfigProvider) *Factory` with `(*Factory).For(ctx, conn model.CalendarConnection) (docalendar.Service, bool)` — builds the adapter from the connection's tokens via a self-persisting source; `bool=false` on any build error.
  - `resolver.New(lister connLister, google calProvider, ms msFactory) *Resolver` implementing `application.CalendarProvider`: `For(ctx, orgID uuid.UUID, organizerEmail string) (docalendar.Service, error)`.
- Consumes: `model.CalendarConnection`, `model.IsNotFound`; the 1a Google `*google.Provider` (as the `calProvider` interface `For(ctx, uuid.UUID, string) (docalendar.Service, error)`); `microsoft.NewCalendarConnector` (Task 1, satisfies `oauthConfigProvider = OAuthConfig(string) *oauth2.Config`).

- [ ] **Step 1: MS self-persisting source** — `microsoft/usersource.go` (duplicate of 1a's google `savingSource`, package-local; the helper is ~10 lines — acceptable duplication, candidate for extraction in 1c):
```go
package microsoft

import "golang.org/x/oauth2"

type savingSource struct {
	base oauth2.TokenSource
	last string
	save func(*oauth2.Token)
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		s.last = tok.AccessToken
		if s.save != nil {
			s.save(tok)
		}
	}
	return tok, nil
}
```

- [ ] **Step 2: MS factory** — `microsoft/factory.go`:
```go
package microsoft

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

type connStore interface {
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
}

type oauthConfigProvider interface {
	OAuthConfig(redirectURL string) *oauth2.Config
}

type Factory struct {
	conns     connStore
	connector oauthConfigProvider
	baseURL   string
}

func NewFactory(conns connStore, connector oauthConfigProvider) *Factory {
	return &Factory{conns: conns, connector: connector, baseURL: graphBaseURL}
}

func (f *Factory) For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool) {
	if f.connector == nil {
		return nil, false
	}
	cfg := f.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{AccessToken: conn.AccessToken, RefreshToken: conn.RefreshToken, Expiry: conn.Expiry})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = f.conns.UpsertCalendarConnection(ctx, conn)
	}}
	return newAdapter(oauth2.NewClient(ctx, src), f.baseURL), true
}
```

- [ ] **Step 3: Write the failing resolver test** — `resolver/resolver_test.go` (`package resolver`):
```go
type fakeLister struct{ conns []model.CalendarConnection }

func (f fakeLister) ListCalendarConnections(_ context.Context, _ string) ([]model.CalendarConnection, error) {
	return f.conns, nil
}

type stubService struct{ tag string }

func (stubService) CreateEvent(context.Context, docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	return docalendar.CalendarResult{}, nil
}
func (stubService) UpdateEvent(context.Context, string, docalendar.CalendarEvent) error { return nil }
func (stubService) UpdateAttendees(context.Context, string, []string) error            { return nil }
func (stubService) DeleteEvent(context.Context, string) error                          { return nil }

type fakeGoogle struct{ called bool }

func (g *fakeGoogle) For(context.Context, uuid.UUID, string) (docalendar.Service, error) {
	g.called = true
	return stubService{tag: "google"}, nil
}

type fakeMS struct{ built bool }

func (m *fakeMS) For(context.Context, model.CalendarConnection) (docalendar.Service, bool) {
	m.built = true
	return stubService{tag: "ms"}, true
}

func TestResolve_MicrosoftWins_MostRecent(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()
	lister := fakeLister{conns: []model.CalendarConnection{
		{Provider: "google", UpdatedAt: old}, {Provider: "microsoft", UpdatedAt: newer},
	}}
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(lister, g, m)
	svc, err := r.For(context.Background(), uuid.New(), "u@x.com")
	if err != nil || svc.(stubService).tag != "ms" {
		t.Fatalf("expected ms, got %+v err=%v", svc, err)
	}
	if g.called {
		t.Error("google should not be called when MS is most recent")
	}
}

func TestResolve_GoogleDelegate_WhenNoMS(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "google", UpdatedAt: time.Now()}}}
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(lister, g, m)
	svc, err := r.For(context.Background(), uuid.New(), "u@x.com")
	if err != nil || svc.(stubService).tag != "google" || !g.called {
		t.Fatalf("expected google delegate, got %+v err=%v", svc, err)
	}
}

func TestResolve_NoConnections_Delegates(t *testing.T) {
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(fakeLister{}, g, m)
	if _, err := r.For(context.Background(), uuid.New(), "u@x.com"); err != nil || !g.called {
		t.Fatalf("expected google/SA delegate, err=%v called=%v", err, g.called)
	}
}
```

- [ ] **Step 4: Implement the resolver** — `resolver/resolver.go`:
```go
package resolver

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type connLister interface {
	ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error)
}

type calProvider interface {
	For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error)
}

type msFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool)
}

type Resolver struct {
	lister connLister
	google calProvider
	ms     msFactory
}

func New(lister connLister, google calProvider, ms msFactory) *Resolver {
	return &Resolver{lister: lister, google: google, ms: ms}
}

func (r *Resolver) For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error) {
	if organizerEmail != "" && r.lister != nil {
		if conns, err := r.lister.ListCalendarConnections(ctx, organizerEmail); err == nil {
			if best, ok := mostRecent(conns); ok && best.Provider == "microsoft" && r.ms != nil {
				if svc, built := r.ms.For(ctx, best); built {
					return svc, nil
				}
			}
		}
	}
	return r.google.For(ctx, organizationID, organizerEmail)
}

func mostRecent(conns []model.CalendarConnection) (model.CalendarConnection, bool) {
	var best model.CalendarConnection
	found := false
	for _, c := range conns {
		if !found || c.UpdatedAt.After(best.UpdatedAt) {
			best, found = c, true
		}
	}
	return best, found
}
```

- [ ] **Step 5: Run resolver test; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/calendar/resolver/ -v`

- [ ] **Step 6: Wire `main.go`** — replace the calendar-provider construction so the Google provider is wrapped by the resolver, and register the MS connector:
```go
var calProvider application.CalendarProvider
if cfg.CalendarStub() { // keep whatever the existing stub guard is
	calProvider = calendarstub.NewProvider()
} else {
	var gconn *oauthgoogle.CalendarConnector
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		gconn = oauthgoogle.NewCalendarConnector(cfg.GoogleClientID, cfg.GoogleClientSecret)
	}
	googleProvider := calendargoogle.NewProvider(store, store, cipher, gconn) // gconn may be nil
	var msFactory *calendarms.Factory
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		msConn := oauthms.NewCalendarConnector(cfg.MicrosoftClientID, cfg.MicrosoftClientSecret)
		msFactory = calendarms.NewFactory(store, msConn)
		connectors["microsoft"] = msConn
	}
	calProvider = calendarresolver.New(store, googleProvider, msFactory)
}
```
Add imports `calendarms "….../infrastructure/calendar/microsoft"`, `calendarresolver "….../infrastructure/calendar/resolver"`, `oauthms "….../infrastructure/oauth/microsoft"` (the existing google import aliases stay). Keep the existing `connectors["google"] = …` registration and `ConfigureCalendarConnectors(connectors)` call. Note: a nil `*calendarms.Factory` passed as the `msFactory` interface is a typed-nil — the resolver guards `r.ms != nil` which would be TRUE for a typed-nil; so pass an untyped nil when MS creds are unset (declare `var msFactory msFactoryIface` or pass `nil` explicitly via a branch). Simplest: in `resolver.New`, also guard each MS call with a check that the concrete is usable — but cleanest is to pass real-nil: build `calProvider` with `resolver.New(store, googleProvider, nilOrFactory)` where `nilOrFactory` is `nil` (untyped) when creds unset. Implement by making the `main.go` variable an interface type set to nil, OR add a `Factory == nil` guard inside `Factory.For`. **Do the latter is impossible (nil receiver method ok but r.ms!=nil still true).** So: declare `var msFactory interface{ For(context.Context, model.CalendarConnection)(docalendar.Service,bool) }` and only assign when creds set; pass it to `resolver.New`. Confirm `go vet` / build is green and the typed-nil trap is avoided.

- [ ] **Step 7: Full verify** — `env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green.

- [ ] **Step 8: gofmt + commit**
```bash
gofmt -w internal/infrastructure/calendar/microsoft/*.go internal/infrastructure/calendar/resolver/*.go cmd/server/main.go
git add apps/backend/internal/infrastructure/calendar/microsoft/factory.go apps/backend/internal/infrastructure/calendar/microsoft/usersource.go \
        apps/backend/internal/infrastructure/calendar/resolver/ apps/backend/cmd/server/main.go
git commit -m "feat(calendar): composite MS/Google/SA resolver + MS factory + wiring"
```

---

### Task 4: Admin — Microsoft connect in the calendar card + i18n

**Files:**
- Modify: `apps/admin/app/features/calendar-connections/components/calendar-connections-card.tsx`
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts`

**Interfaces:** Consumes the 1a entity (`useCalendarConnections`, `useStartConnect`, `useDisconnect` — all already take a `provider` arg). No new entity code.

- [ ] **Step 1:** In the card, derive `const microsoft = data.find((c) => c.provider === "microsoft")` alongside the existing `google`. Render a second control block: connected → "Connected as {email}" + Disconnect(`"microsoft"`); else → "Connect Microsoft" button calling `start.mutate("microsoft")`. Use a lucide icon. Keep the file ≤300 lines, no comments.
- [ ] **Step 2:** Add i18n keys to en/ru/kk: `settings.calendars.connectMicrosoft` (EN "Connect Microsoft" / RU formal "Подключить Microsoft" / KK "Microsoft қосу"). Reuse the existing `disconnect` / `connected` keys. Ensure all three dicts updated (parity is compile-enforced).
- [ ] **Step 3: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/features/calendar-connections/components/calendar-connections-card.tsx apps/admin/app/shared/i18n/dictionaries
git commit -m "feat(admin): Microsoft calendar connect in settings card + i18n"
```

---

### Task 5: Mini-app — Microsoft connect in the calendar row + i18n

**Files:**
- Modify: `apps/mini-app/app/features/profile/components/calendar-connection-row.tsx`
- Modify: `apps/mini-app/app/shared/i18n/dictionaries/{en,ru,kk}.ts`

**Interfaces:** Consumes the 1a mini-app entity hooks (provider-parameterized). No new entity code.

- [ ] **Step 1:** In the row, derive `microsoft` from the connections list; render a Microsoft connect/disconnect control next to Google. Connect calls `start.mutateAsync("microsoft")` then `getWebApp()?.openLink?.(res.auth_url)` (reuse the existing connect handler, parameterized by provider — refactor the Google handler to take a `provider` arg rather than duplicating). Keep ≤300 lines, no comments, no emoji.
- [ ] **Step 2:** Add i18n keys to en/ru/kk: `profile.calendar.connectMicrosoft` (EN "Connect Microsoft" / RU informal "Подключить Microsoft" / KK "Microsoft қосу"); reuse `disconnect`/`connected`. All three dicts.
- [ ] **Step 3: Verify + commit** (watch the `entities/meeting/api.ts` reflow gotcha — do not reformat unrelated files)
```bash
pnpm --filter mini-app typecheck && pnpm --filter mini-app lint && pnpm --filter mini-app build
git add apps/mini-app/app/features/profile/components/calendar-connection-row.tsx apps/mini-app/app/shared/i18n/dictionaries
git commit -m "feat(mini-app): Microsoft calendar connect in profile row + i18n"
```

---

### Task 6: Whole-slice verification

**Files:** none (verification only)

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green; MS adapter + connector + resolver tests run and PASS (httptest, no Docker needed).
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck lint build` and `pnpm --filter mini-app typecheck lint build`. Green; i18n parity holds.
- [ ] **Step 3: Manual/ops note (documented, not automated)** — record in the verification: MS OAuth app must expose Graph scopes `Calendars.ReadWrite` + `OnlineMeetings.ReadWrite` (the latter may need tenant admin consent); reuse the existing Microsoft SSO client id/secret. No real-Graph test (deferred to WS4 E2E).
- [ ] **Step 4: Tree clean** — verify `HEAD`, `git status` shows no stray staged files; the user's parallel WIP (if any) untouched.

---

## Notes for the executor

- **SA-fallback invariant** (Task 3): any MS factory failure returns `built=false` → resolver delegates to the Google/SA path. Meeting creation must never hard-fail. This is the top correctness property.
- **Typed-nil trap** (Task 3 Step 6): if MS creds are unset, pass a real nil `msFactory` (interface var left nil), not a typed-nil `*microsoft.Factory`, so the resolver's `r.ms != nil` guard works.
- **MS OAuth ≠ Google OAuth:** `offline_access` scope (not `AccessTypeOffline`), `prompt=consent` (not `ApprovalForce`).
- **`savingSource` duplication** in the microsoft pkg is intentional (tiny, avoids a shared pkg for now) — candidate for extraction in 1c when Google free/busy also lands.
- **Deferred to 1c/2a/3:** free/busy engine wiring (the `BusyReader` is built but unconsumed), onboarding, booking links.
