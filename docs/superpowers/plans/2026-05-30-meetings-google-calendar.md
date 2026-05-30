# Meetings: Real Google Calendar Adapter (Increment 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the stubbed `CalendarService` with a real Google Calendar adapter that creates events with Meet links, selected per-workspace from encrypted service-account credentials; keep the stub for local/CI via a `CALENDAR_STUB` flag.

**Architecture:** A `CalendarProvider.For(ctx, workspaceID)` resolves a `CalendarService` per workspace. Two providers: `stub` (always the fake) and `google` (loads encrypted SA JSON from the workspace, impersonates a subject via domain-wide delegation, creates events with `conferenceData`). No creds → `ErrGoogleNotConfigured` → HTTP 400 (the existing CreateMeeting handler already maps service errors to 400).

**Tech Stack:** Go 1.26, pgx, `golang.org/x/oauth2/google` (already vendored), new `google.golang.org/api/calendar/v3` + `google.golang.org/api/option`. Spec: `docs/superpowers/specs/2026-05-30-meetings-google-calendar-design.md`.

**Run from:** `backend/` with `env -u GOROOT go ...`. Tasks build green individually; the type-flip happens atomically in Task 7.

---

### Task 1: Migration — workspace Google config columns

**Files:**
- Create: `backend/migrations/20260530130000_meetings_google.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_sa_json_enc BYTEA;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_subject TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_calendar_id TEXT NOT NULL DEFAULT 'primary';

-- +goose Down
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_calendar_id;
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_subject;
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_sa_json_enc;
```

- [ ] **Step 2: Apply and verify**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && set -a && . ./.env && set +a && cd backend && env -u GOROOT go run ./cmd/migrate up`
Expected: `OK 20260530130000_meetings_google.sql` and `successfully migrated database to version: 20260530130000`.

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260530130000_meetings_google.sql
git commit -m "feat(meetings): workspace google calendar config columns"
```

---

### Task 2: Store — Get/SetGoogleConfig

**Files:**
- Create: `backend/internal/infrastructure/persistence/postgres/google_config_repo.go`

(No DB unit test — the package has no DB harness; covered manually/by build.)

- [ ] **Step 1: Write the repo**

`backend/internal/infrastructure/persistence/postgres/google_config_repo.go`:
```go
package postgres

import (
	"context"

	"github.com/google/uuid"
)

// SetGoogleConfig stores the (already-encrypted) service-account JSON plus the
// impersonation subject and target calendar for a workspace.
func (s *Store) SetGoogleConfig(ctx context.Context, id uuid.UUID, encJSON []byte, subject, calendarID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workspaces
		SET google_sa_json_enc = $2, google_subject = $3, google_calendar_id = $4, updated_at = now()
		WHERE id = $1`, id, encJSON, subject, calendarID)
	return err
}

// GetGoogleConfig returns the encrypted SA JSON, subject, and calendar id.
func (s *Store) GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT google_sa_json_enc, google_subject, google_calendar_id
		FROM workspaces WHERE id = $1`, id).Scan(&encJSON, &subject, &calendarID)
	return encJSON, subject, calendarID, err
}
```

- [ ] **Step 2: Build**

Run: `cd backend && env -u GOROOT go build ./...`
Expected: builds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/google_config_repo.go
git commit -m "feat(meetings): store get/set workspace google config"
```

---

### Task 3: Config — CALENDAR_STUB flag

**Files:**
- Modify: `backend/internal/platform/config/config.go` (add field + load)

- [ ] **Step 1: Add the field**

In `config.go`, add to the `Config` struct (next to `AutoMigrate bool`):
```go
	CalendarStub bool
```

- [ ] **Step 2: Load it**

In `config.go` `Load()`, next to `cfg.AutoMigrate = strings.EqualFold(...)`, add:
```go
	cfg.CalendarStub = strings.EqualFold(os.Getenv("CALENDAR_STUB"), "true")
```

- [ ] **Step 3: Build and commit**

Run: `cd backend && env -u GOROOT go build ./...` → builds.
```bash
git add backend/internal/platform/config/config.go
git commit -m "feat(meetings): CALENDAR_STUB config flag"
```

---

### Task 4: Application — CalendarProvider port + ErrGoogleNotConfigured (additive)

**Files:**
- Modify: `backend/internal/application/calendar.go` (append; do NOT change Services yet)

- [ ] **Step 1: Append the port + error**

Append to `backend/internal/application/calendar.go`. Add `"errors"` and `"github.com/google/uuid"` to its imports, then add:
```go
// ErrGoogleNotConfigured is returned when a workspace has no Google credentials.
var ErrGoogleNotConfigured = errors.New("google not configured")

// CalendarProvider resolves the CalendarService to use for a given workspace.
type CalendarProvider interface {
	For(ctx context.Context, workspaceID uuid.UUID) (CalendarService, error)
}
```

- [ ] **Step 2: Build**

Run: `cd backend && env -u GOROOT go build ./...`
Expected: builds (purely additive; `Services.Calendar` is still `CalendarService`).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application/calendar.go
git commit -m "feat(meetings): CalendarProvider port + ErrGoogleNotConfigured"
```

---

### Task 5: Stub provider

**Files:**
- Create: `backend/internal/infrastructure/calendar/stub/provider.go`
- Test: `backend/internal/infrastructure/calendar/stub/provider_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/infrastructure/calendar/stub/provider_test.go`:
```go
package stub

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestProviderFor(t *testing.T) {
	cal, err := NewProvider().For(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if cal == nil {
		t.Fatal("expected a CalendarService")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/stub/ -run TestProviderFor -v`
Expected: FAIL — `NewProvider` undefined.

- [ ] **Step 3: Write the provider**

`backend/internal/infrastructure/calendar/stub/provider.go`:
```go
package stub

import (
	"context"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

// Provider always resolves to the stub Service, regardless of workspace.
// Used when CALENDAR_STUB=true (local/CI).
type Provider struct{ svc *Service }

func NewProvider() *Provider { return &Provider{svc: New()} }

func (p *Provider) For(_ context.Context, _ uuid.UUID) (application.CalendarService, error) {
	return p.svc, nil
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/stub/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/calendar/stub/provider.go backend/internal/infrastructure/calendar/stub/provider_test.go
git commit -m "feat(meetings): stub calendar provider"
```

---

### Task 6: Google adapter + provider

**Files:**
- Create: `backend/internal/infrastructure/calendar/google/adapter.go`
- Create: `backend/internal/infrastructure/calendar/google/provider.go`
- Test: `backend/internal/infrastructure/calendar/google/adapter_test.go`

- [ ] **Step 1: Add the dependency**

Run:
```bash
cd backend && env -u GOROOT go get google.golang.org/api/calendar/v3 google.golang.org/api/option
```
Expected: go.mod/go.sum updated, no error.

- [ ] **Step 2: Write the failing test (pure buildEvent)**

`backend/internal/infrastructure/calendar/google/adapter_test.go`:
```go
package google

import (
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

func TestBuildEvent(t *testing.T) {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	ev := buildEvent(application.CalendarEvent{
		Title:          "Sync",
		Description:    "desc",
		Start:          start,
		End:            start.Add(time.Hour),
		AttendeeEmails: []string{"a@example.com", "b@example.com"},
	}, "req-123")

	if ev.Summary != "Sync" || ev.Description != "desc" {
		t.Fatalf("summary/desc wrong: %+v", ev)
	}
	if ev.Start.DateTime != "2025-06-02T10:00:00Z" || ev.End.DateTime != "2025-06-02T11:00:00Z" {
		t.Fatalf("times wrong: %s / %s", ev.Start.DateTime, ev.End.DateTime)
	}
	if len(ev.Attendees) != 2 || ev.Attendees[0].Email != "a@example.com" {
		t.Fatalf("attendees wrong: %+v", ev.Attendees)
	}
	if ev.ConferenceData == nil || ev.ConferenceData.CreateRequest == nil {
		t.Fatal("missing conference create request")
	}
	if ev.ConferenceData.CreateRequest.RequestId != "req-123" {
		t.Fatalf("request id wrong: %s", ev.ConferenceData.CreateRequest.RequestId)
	}
	if ev.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
		t.Fatalf("solution key wrong: %+v", ev.ConferenceData.CreateRequest.ConferenceSolutionKey)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestBuildEvent -v`
Expected: FAIL — `buildEvent` undefined.

- [ ] **Step 4: Write the adapter**

`backend/internal/infrastructure/calendar/google/adapter.go`:
```go
// Package google is the real Google Calendar adapter. It impersonates a
// workspace's configured subject (domain-wide delegation) and creates events
// with a Google Meet conference link.
package google

import (
	"context"
	"time"

	"github.com/google/uuid"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

type adapter struct {
	svc        *calendar.Service
	calendarID string
}

// buildEvent maps a transport-agnostic CalendarEvent to a Google event with a
// Meet create-request. requestID must be unique per insert.
func buildEvent(e application.CalendarEvent, requestID string) *calendar.Event {
	var attendees []*calendar.EventAttendee
	for _, em := range e.AttendeeEmails {
		attendees = append(attendees, &calendar.EventAttendee{Email: em})
	}
	return &calendar.Event{
		Summary:     e.Title,
		Description: e.Description,
		Start:       &calendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
		Attendees:   attendees,
		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId:             requestID,
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		},
	}
}

func (a *adapter) CreateEvent(ctx context.Context, e application.CalendarEvent) (application.CalendarResult, error) {
	created, err := a.svc.Events.
		Insert(a.calendarID, buildEvent(e, uuid.NewString())).
		ConferenceDataVersion(1).
		Context(ctx).
		Do()
	if err != nil {
		return application.CalendarResult{}, err
	}
	link := created.HangoutLink
	if link == "" && created.ConferenceData != nil {
		for _, ep := range created.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				link = ep.Uri
				break
			}
		}
	}
	return application.CalendarResult{EventID: created.Id, MeetLink: link}, nil
}

func (a *adapter) DeleteEvent(ctx context.Context, eventID string) error {
	return a.svc.Events.Delete(a.calendarID, eventID).Context(ctx).Do()
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestBuildEvent -v`
Expected: PASS.

- [ ] **Step 6: Write the provider**

`backend/internal/infrastructure/calendar/google/provider.go`:
```go
package google

import (
	"context"

	"github.com/google/uuid"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Provider builds a per-workspace Google Calendar client from the workspace's
// encrypted service-account credentials.
type Provider struct {
	store  *postgres.Store
	cipher *crypto.TokenCipher
}

func NewProvider(store *postgres.Store, cipher *crypto.TokenCipher) *Provider {
	return &Provider{store: store, cipher: cipher}
}

func (p *Provider) For(ctx context.Context, workspaceID uuid.UUID) (application.CalendarService, error) {
	enc, subject, calendarID, err := p.store.GetGoogleConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 || subject == "" {
		return nil, application.ErrGoogleNotConfigured
	}
	saJSON, err := p.cipher.Decrypt(enc)
	if err != nil {
		return nil, err
	}
	jwtCfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
	if err != nil {
		return nil, err
	}
	jwtCfg.Subject = subject // domain-wide delegation
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(jwtCfg.Client(ctx)))
	if err != nil {
		return nil, err
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return &adapter{svc: svc, calendarID: calendarID}, nil
}
```

- [ ] **Step 7: Build + tidy**

Run: `cd backend && env -u GOROOT go mod tidy && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/infrastructure/calendar/google/ -v`
Expected: builds; buildEvent test passes.

- [ ] **Step 8: Commit**

```bash
git add backend/go.mod backend/go.sum backend/internal/infrastructure/calendar/google/
git commit -m "feat(meetings): real Google Calendar adapter + per-workspace provider"
```

---

### Task 7: Flip Services to CalendarProvider + wire selection

**Files:**
- Modify: `backend/internal/application/services.go` (Calendar field type)
- Modify: `backend/internal/application/meeting_service.go` (resolve per workspace)
- Modify: `backend/internal/delivery/http/app.go` (select provider by flag)

This task changes the `Calendar` field type and all its uses together so the build stays green.

- [ ] **Step 1: Change the Services field type**

In `backend/internal/application/services.go`, change the `Calendar` field:
```go
	Calendar CalendarProvider
```
(was `Calendar CalendarService`).

- [ ] **Step 2: Resolve the calendar per workspace in CreateMeeting**

In `backend/internal/application/meeting_service.go`, inside `CreateMeeting`, replace the existing calendar call block:
```go
	cal, err := s.Calendar.CreateEvent(ctx, CalendarEvent{
		Title: name, Description: in.Description,
		Start: startsAt, End: endsAt, AttendeeEmails: emails,
	})
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("calendar: %w", err)
	}
```
with:
```go
	calSvc, err := s.Calendar.For(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	cal, err := calSvc.CreateEvent(ctx, CalendarEvent{
		Title: name, Description: in.Description,
		Start: startsAt, End: endsAt, AttendeeEmails: emails,
	})
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("calendar: %w", err)
	}
```
Note: returning `err` directly (not wrapped) preserves `ErrGoogleNotConfigured`; the CreateMeeting HTTP handler already maps any service error to HTTP 400, so an unconfigured workspace yields `400 "google not configured"`.

- [ ] **Step 3: Resolve the calendar per workspace in CancelMeeting**

In `meeting_service.go`, inside `CancelMeeting`, replace:
```go
	if m.GoogleEventID != "" {
		_ = s.Calendar.DeleteEvent(ctx, m.GoogleEventID) // best-effort; real adapter increment will log/retry
	}
```
with:
```go
	if m.GoogleEventID != "" {
		if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr == nil {
			_ = calSvc.DeleteEvent(ctx, m.GoogleEventID) // best-effort
		}
	}
```

- [ ] **Step 4: Wire provider selection in app.go**

In `backend/internal/delivery/http/app.go`, add the google import alongside the existing `calendarstub` import (line ~21):
```go
	calendargoogle "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/calendar/google"
```
Then, just before the `api := &handlers.API{...}` literal, add:
```go
	var calProvider application.CalendarProvider
	if cfg.CalendarStub {
		calProvider = calendarstub.NewProvider()
	} else {
		calProvider = calendargoogle.NewProvider(store, cipher)
	}
```
And change the `App:` line to use it:
```go
		App:     &application.Services{Store: store, Cipher: cipher, Queue: queue, Calendar: calProvider},
```

- [ ] **Step 5: Build, vet, test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -count=1 ./...`
Expected: all green. (`go vet -tags=smoke ./test/smoke/...` should also pass — unchanged.)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/application/services.go backend/internal/application/meeting_service.go backend/internal/delivery/http/app.go
git commit -m "feat(meetings): select calendar provider per workspace (stub|google)"
```

---

### Task 8: Config endpoint — set/read Google config via integrations

**Files:**
- Modify: `backend/internal/application/services.go` (add `SetGoogleConfig`, extend `IntegrationsView` + `GetIntegrations`)
- Modify: `backend/internal/delivery/http/handlers/handlers.go` (extend `PatchIntegrations` body)

- [ ] **Step 1: Add SetGoogleConfig + extend the integrations view**

In `backend/internal/application/services.go`:

(a) Extend the `IntegrationsView` struct with three fields:
```go
	HasGoogle        bool   `json:"has_google"`
	GoogleSubject    string `json:"google_subject"`
	GoogleCalendarID string `json:"google_calendar_id"`
```

(b) In `GetIntegrations`, before the `return IntegrationsView{...}`, load the google config and include it. Replace the final `return IntegrationsView{...}, nil` with:
```go
	encG, subjectG, calIDG, _ := s.Store.GetGoogleConfig(ctx, workspaceID)
	return IntegrationsView{
		VCSProvider:      w.VCSProvider,
		VCSNamespace:     w.VCSNamespace,
		VCSBaseURL:       base,
		HasVCSToken:      w.HasVCSToken,
		MeetLink:         w.MeetLink,
		TZ:               w.TZ,
		HasGoogle:        len(encG) > 0 && subjectG != "",
		GoogleSubject:    subjectG,
		GoogleCalendarID: calIDG,
	}, nil
```

(c) Add a new method (next to `PatchIntegrations`):
```go
// SetGoogleConfig encrypts and stores per-workspace Google credentials. An empty
// saJSON keeps the existing key (so subject/calendar can be updated alone).
func (s *Services) SetGoogleConfig(ctx context.Context, workspaceID uuid.UUID, saJSON, subject, calendarID string) error {
	var enc []byte
	if saJSON != "" {
		e, err := s.Cipher.Encrypt(saJSON)
		if err != nil {
			return err
		}
		enc = e
	} else {
		e, _, _, err := s.Store.GetGoogleConfig(ctx, workspaceID)
		if err != nil {
			return err
		}
		enc = e
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return s.Store.SetGoogleConfig(ctx, workspaceID, enc, subject, calendarID)
}
```

- [ ] **Step 2: Extend the PatchIntegrations handler**

In `backend/internal/delivery/http/handlers/handlers.go`, in `PatchIntegrations`, add the three fields to the `body` struct:
```go
		GoogleSAJSON     string `json:"google_sa_json"`
		GoogleSubject    string `json:"google_subject"`
		GoogleCalendarID string `json:"google_calendar_id"`
```
Then, after the existing `a.App.PatchIntegrations(...)` call succeeds (before `return c.SendStatus(...)`), add:
```go
	if body.GoogleSAJSON != "" || body.GoogleSubject != "" || body.GoogleCalendarID != "" {
		if err := a.App.SetGoogleConfig(c.Context(), id, body.GoogleSAJSON, body.GoogleSubject, body.GoogleCalendarID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
```

- [ ] **Step 3: Build, vet, test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -count=1 ./...`
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/application/services.go backend/internal/delivery/http/handlers/handlers.go
git commit -m "feat(meetings): set/read per-workspace google config via integrations"
```

---

### Task 9: Stub flag for local/CI + verify smoke

**Files:**
- Modify: `deploy/.env.example` (add `CALENDAR_STUB=true`)
- Modify: `.github/workflows/_smoke.yml` (add `CALENDAR_STUB: "true"` to the server step env)

- [ ] **Step 1: Add to `deploy/.env.example`**

Append a line under the existing flags (e.g., after `AUTO_MIGRATE=true`):
```
CALENDAR_STUB=true
```

- [ ] **Step 2: Add to the smoke workflow**

In `.github/workflows/_smoke.yml`, in the "start server" step's `env:` block (which already has `AUTH_DEV_MODE`, `AUTO_MIGRATE`, etc.), add:
```yaml
          CALENDAR_STUB: "true"
```

- [ ] **Step 3: Verify smoke locally**

From repo root, ensure `.env` has `CALENDAR_STUB=true` (copy from deploy/.env.example or export it), start DB + server, then run the meetings smoke:
```bash
make up
set -a && . ./.env && set +a
CALENDAR_STUB=true bash -c 'cd backend && env -u GOROOT go run ./cmd/server' &
# wait for health, then:
cd backend && SMOKE_BASE_URL=http://localhost:8080 SMOKE_TOKEN="Bearer smoke-owner" SMOKE_TOKEN_B="Bearer smoke-stranger" \
  env -u GOROOT go test -tags=smoke -count=1 -run TestSmokeMeetings ./test/smoke/...
```
Expected: `TestSmokeMeetings` PASSES (stub provider active because `CALENDAR_STUB=true`). Stop the server afterward.

- [ ] **Step 4: Commit**

```bash
git add deploy/.env.example .github/workflows/_smoke.yml
git commit -m "chore(meetings): default CALENDAR_STUB=true for local + CI smoke"
```

---

### Task 10: Docs

**Files:**
- Modify: `docs/MEETINGS.md` (update backend status + Google config)
- Modify: `docs/REQUIREMENTS.md` (note Google env/config now wired)

- [ ] **Step 1: Update `docs/MEETINGS.md`**

In the "Backend" section, update the increment note to add:
```markdown
> **Increment 2 (done):** real Google Calendar adapter (per-workspace encrypted service-account creds + domain-wide delegation, Meet link via conferenceData). Selected per workspace; `CALENDAR_STUB=true` forces the stub for local/CI. Configure via `PATCH /api/workspaces/:id/integrations` (`google_sa_json`, `google_subject`, `google_calendar_id`); a workspace without creds returns 400 on meeting create.
```

- [ ] **Step 2: Update `docs/REQUIREMENTS.md`**

In §5 (Meetings), under the backend bullet, append:
```markdown
- **Google (per-workspace):** encrypted service-account JSON + subject (domain-wide delegation) + calendar id, set via the integrations endpoint. `CALENDAR_STUB=true` uses the stub (local/CI).
```

- [ ] **Step 3: Format and commit**

Run `make fmt-check` (run `make fmt` first if docs reflow, then stage only these two docs).
```bash
git add docs/MEETINGS.md docs/REQUIREMENTS.md
git commit -m "docs(meetings): document increment-2 google calendar adapter"
```

---

## Done criteria

- `make lint` → 0 issues; `make test` → all pass (incl. `google` buildEvent + `stub` provider tests); `make typecheck` → 0; `make fmt-check` → green; `make build`.
- With `CALENDAR_STUB=true`: `TestSmokeMeetings` passes (stub).
- With real per-workspace creds (manual, out of CI): creating a meeting yields a real Google Calendar event + Meet link; cancel deletes it. A workspace without creds returns `400 "google not configured"` on create.
