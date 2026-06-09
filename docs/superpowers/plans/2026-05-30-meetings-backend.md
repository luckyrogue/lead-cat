# Meetings Backend (Increment 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first vertical slice of the Google Meet meetings feature on the backend — create/list/get/cancel meetings over REST (serving the Mini App), with Google Calendar behind a stubbed port.

**Architecture:** Within the existing clean-architecture Go monolith and multi-tenant workspace model. A `CalendarService` port is defined in `application`; a `stub` adapter lives in `infrastructure/calendar/stub`. Meetings/employees are new workspace-scoped Postgres tables. Reuses existing JWT + `RequireWorkspaceAccess` middleware. Integration verified via the build-tagged Go smoke E2E test.

**Tech Stack:** Go 1.26, Fiber v2, pgx/pgxpool, goose migrations, google/uuid. Spec: `docs/superpowers/specs/2026-05-30-meetings-backend-design.md`.

**Run from:** `backend/` (commands use `env -u GOROOT go ...` to match the repo Makefile).

---

### Task 1: Migration — meetings, employees, participants tables

**Files:**

- Create: `backend/migrations/20260530120000_meetings.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL,
    dept TEXT NOT NULL DEFAULT '',
    has_telegram BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, email)
);

CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    organizer_user_id UUID REFERENCES platform_users(id),
    dept TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    recurrence TEXT NOT NULL DEFAULT 'once',
    recurrence_until DATE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    google_event_id TEXT NOT NULL DEFAULT '',
    meet_link TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'scheduled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX meetings_workspace_id_idx ON meetings (workspace_id);
CREATE INDEX meetings_organizer_idx ON meetings (organizer_user_id);

CREATE TABLE meeting_participants (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees(id) ON DELETE SET NULL,
    email TEXT NOT NULL
);

CREATE INDEX meeting_participants_meeting_idx ON meeting_participants (meeting_id);

-- +goose Down
DROP TABLE IF EXISTS meeting_participants;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS employees;
```

- [ ] **Step 2: Apply and verify migration**

Run: `cd backend && set -a && . ../.env && set +a && env -u GOROOT go run ./cmd/migrate up`
(or from repo root: `make migrate`)
Expected: `OK 20260530120000_meetings.sql` and `successfully migrated database to version: 20260530120000`.

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260530120000_meetings.sql
git commit -m "feat(meetings): add meetings/employees/participants tables"
```

---

### Task 2: Domain — meeting input, recurrence enum, validation

**Files:**

- Create: `backend/internal/domain/meeting/meeting.go`
- Create: `backend/internal/domain/meeting/validate.go`
- Test: `backend/internal/domain/meeting/validate_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/domain/meeting/validate_test.go`:

```go
package meeting

import (
	"testing"
	"time"
)

func base() Input {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	return Input{
		Dept:       "Разработка",
		Type:       "Планёрка",
		Host:       "Иванов А.А.",
		StartsAt:   start,
		EndsAt:     start.Add(time.Hour),
		Recurrence: Once,
	}
}

func TestValidate_OK(t *testing.T) {
	if err := base().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidate_EndBeforeStart(t *testing.T) {
	in := base()
	in.EndsAt = in.StartsAt.Add(-time.Minute)
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for end <= start")
	}
}

func TestValidate_MissingDept(t *testing.T) {
	in := base()
	in.Dept = ""
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for missing dept")
	}
}

func TestValidate_BadRecurrence(t *testing.T) {
	in := base()
	in.Recurrence = "yearly"
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for unknown recurrence")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run TestValidate -v`
Expected: FAIL — package does not compile (`Input`, `Once` undefined).

- [ ] **Step 3: Write the domain types**

`backend/internal/domain/meeting/meeting.go`:

```go
// Package meeting holds the meeting domain: types, recurrence rules,
// validation, and the meeting-name standard. No persistence or transport here.
package meeting

import "time"

// Recurrence is how often a meeting repeats.
type Recurrence string

const (
	Once     Recurrence = "once"
	Daily    Recurrence = "daily"
	Weekly   Recurrence = "weekly"
	Biweekly Recurrence = "biweekly"
	Monthly  Recurrence = "monthly"
)

var recurrenceLabels = map[Recurrence]string{
	Once:     "",
	Daily:    "Ежедневно",
	Weekly:   "Еженедельно",
	Biweekly: "Раз в 2 недели",
	Monthly:  "Ежемесячно",
}

// Valid reports whether r is a known recurrence value.
func (r Recurrence) Valid() bool {
	_, ok := recurrenceLabels[r]
	return ok
}

// Label is the human-readable Russian label used in the meeting name.
func (r Recurrence) Label() string { return recurrenceLabels[r] }

// Input is the validated, parsed payload for creating a meeting.
type Input struct {
	Dept        string
	Type        string
	Host        string
	StartsAt    time.Time
	EndsAt      time.Time
	Recurrence  Recurrence
	Description string
}
```

- [ ] **Step 4: Write the validation**

`backend/internal/domain/meeting/validate.go`:

```go
package meeting

import "fmt"

// Validate checks required fields, time ordering, and recurrence.
func (in Input) Validate() error {
	if in.Dept == "" {
		return fmt.Errorf("dept required")
	}
	if in.Type == "" {
		return fmt.Errorf("type required")
	}
	if in.Host == "" {
		return fmt.Errorf("host required")
	}
	if in.StartsAt.IsZero() || in.EndsAt.IsZero() {
		return fmt.Errorf("start and end required")
	}
	if !in.EndsAt.After(in.StartsAt) {
		return fmt.Errorf("end must be after start")
	}
	if !in.Recurrence.Valid() {
		return fmt.Errorf("unknown recurrence: %q", in.Recurrence)
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run TestValidate -v`
Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/domain/meeting/
git commit -m "feat(meetings): domain input + recurrence + validation"
```

---

### Task 3: Domain — meeting name standard

**Files:**

- Create: `backend/internal/domain/meeting/naming.go`
- Test: `backend/internal/domain/meeting/naming_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/domain/meeting/naming_test.go`:

```go
package meeting

import (
	"testing"
	"time"
)

func TestGenerateName_Once(t *testing.T) {
	d := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	got := GenerateName("Разработка", "Планёрка", "Иванов А.А.", d, Once)
	want := "Разработка | Планёрка | Иванов А.А. | 2025-06-02"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGenerateName_Weekly(t *testing.T) {
	d := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	got := GenerateName("Разработка", "Планёрка", "Иванов А.А.", d, Weekly)
	want := "Разработка | Планёрка | Иванов А.А. | 2025-06-02 | Еженедельно"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run TestGenerateName -v`
Expected: FAIL — `GenerateName` undefined.

- [ ] **Step 3: Write the implementation**

`backend/internal/domain/meeting/naming.go`:

```go
package meeting

import (
	"fmt"
	"time"
)

// GenerateName builds the standard meeting title:
//
//	[Отдел] | [Тип] | [Ведущий] | [Дата YYYY-MM-DD] | [Частота]
//
// For a one-time meeting the frequency segment is omitted.
func GenerateName(dept, mtype, host string, date time.Time, r Recurrence) string {
	name := fmt.Sprintf("%s | %s | %s | %s", dept, mtype, host, date.Format("2006-01-02"))
	if label := r.Label(); label != "" {
		name += " | " + label
	}
	return name
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -v`
Expected: PASS (all meeting domain tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/meeting/naming.go backend/internal/domain/meeting/naming_test.go
git commit -m "feat(meetings): meeting name standard"
```

---

### Task 4: Calendar port + stub adapter

**Files:**

- Create: `backend/internal/application/calendar.go`
- Create: `backend/internal/infrastructure/calendar/stub/stub.go`
- Test: `backend/internal/infrastructure/calendar/stub/stub_test.go`

- [ ] **Step 1: Write the port**

`backend/internal/application/calendar.go`:

```go
package application

import (
	"context"
	"time"
)

// CalendarEvent is a calendar event to create (transport-agnostic).
type CalendarEvent struct {
	Title          string
	Description    string
	Start          time.Time
	End            time.Time
	AttendeeEmails []string
}

// CalendarResult is what the calendar backend returns after creation.
type CalendarResult struct {
	EventID  string
	MeetLink string
}

// CalendarService is the port for the calendar backend (Google Calendar in
// production, a stub in tests/local). Implemented in infrastructure/calendar/*.
type CalendarService interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	DeleteEvent(ctx context.Context, eventID string) error
}
```

- [ ] **Step 2: Write the failing stub test**

`backend/internal/infrastructure/calendar/stub/stub_test.go`:

```go
package stub

import (
	"context"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
)

func TestStubCreateEvent(t *testing.T) {
	res, err := New().CreateEvent(context.Background(), application.CalendarEvent{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.EventID == "" || !strings.HasPrefix(res.MeetLink, "https://meet.google.com/") {
		t.Fatalf("bad result: %+v", res)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/stub/ -v`
Expected: FAIL — `New` undefined.

- [ ] **Step 4: Write the stub**

`backend/internal/infrastructure/calendar/stub/stub.go`:

```go
// Package stub is a no-network CalendarService used locally and in tests until
// the real Google Calendar adapter lands. It fabricates event IDs and Meet links.
package stub

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) CreateEvent(_ context.Context, _ application.CalendarEvent) (application.CalendarResult, error) {
	id := uuid.NewString()
	return application.CalendarResult{
		EventID:  id,
		MeetLink: "https://meet.google.com/stub-" + id[:8],
	}, nil
}

func (s *Service) DeleteEvent(_ context.Context, _ string) error { return nil }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/calendar/stub/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/application/calendar.go backend/internal/infrastructure/calendar/stub/
git commit -m "feat(meetings): CalendarService port + stub adapter"
```

---

### Task 5: Postgres models + repositories

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/models.go` (append types)
- Create: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`
- Create: `backend/internal/infrastructure/persistence/postgres/employee_repo.go`

(Repository SQL is exercised end-to-end by the smoke test in Task 8; no DB unit test — the package has no DB harness.)

- [ ] **Step 1: Append models**

Append to `backend/internal/infrastructure/persistence/postgres/models.go`:

```go
type Employee struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	Dept        string    `json:"dept"`
	HasTelegram bool      `json:"has_telegram"`
}

type MeetingParticipant struct {
	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	Email      string     `json:"email"`
}

type Meeting struct {
	ID              uuid.UUID            `json:"id"`
	WorkspaceID     uuid.UUID            `json:"workspace_id"`
	OrganizerUserID *uuid.UUID           `json:"organizer_user_id,omitempty"`
	Dept            string               `json:"dept"`
	Type            string               `json:"type"`
	Host            string               `json:"host"`
	StartsAt        time.Time            `json:"starts_at"`
	EndsAt          time.Time            `json:"ends_at"`
	Recurrence      string               `json:"recurrence"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	GoogleEventID   string               `json:"google_event_id"`
	MeetLink        string               `json:"meet_link"`
	Status          string               `json:"status"`
	Participants    []MeetingParticipant `json:"participants"`
}
```

- [ ] **Step 2: Write the employee repository**

`backend/internal/infrastructure/persistence/postgres/employee_repo.go`:

```go
package postgres

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) ListEmployees(ctx context.Context, workspaceID uuid.UUID) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, full_name, email, dept, has_telegram
		FROM employees WHERE workspace_id = $1 ORDER BY full_name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEmployee(ctx context.Context, workspaceID uuid.UUID, fullName, email, dept string, hasTelegram bool) (Employee, error) {
	var e Employee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO employees (workspace_id, full_name, email, dept, has_telegram)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, workspace_id, full_name, email, dept, has_telegram`,
		workspaceID, fullName, email, dept, hasTelegram).
		Scan(&e.ID, &e.WorkspaceID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram)
	return e, err
}
```

- [ ] **Step 3: Write the meeting repository**

`backend/internal/infrastructure/persistence/postgres/meeting_repo.go`:

```go
package postgres

import (
	"context"

	"github.com/google/uuid"
)

const meetingCols = `id, workspace_id, organizer_user_id, dept, type, host,
	starts_at, ends_at, recurrence, name, description, google_event_id, meet_link, status`

func scanMeeting(row interface {
	Scan(dest ...any) error
}) (Meeting, error) {
	var m Meeting
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.OrganizerUserID, &m.Dept, &m.Type, &m.Host,
		&m.StartsAt, &m.EndsAt, &m.Recurrence, &m.Name, &m.Description, &m.GoogleEventID, &m.MeetLink, &m.Status)
	return m, err
}

func (s *Store) CreateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO meetings (workspace_id, organizer_user_id, dept, type, host,
			starts_at, ends_at, recurrence, name, description, google_event_id, meet_link)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING `+meetingCols,
		m.WorkspaceID, m.OrganizerUserID, m.Dept, m.Type, m.Host,
		m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description, m.GoogleEventID, m.MeetLink)
	return scanMeeting(row)
}

func (s *Store) AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []MeetingParticipant) error {
	for _, p := range ps {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO meeting_participants (meeting_id, employee_id, email)
			VALUES ($1, $2, $3)`, meetingID, p.EmployeeID, p.Email); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]MeetingParticipant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT employee_id, email FROM meeting_participants WHERE meeting_id = $1`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MeetingParticipant
	for rows.Next() {
		var p MeetingParticipant
		if err := rows.Scan(&p.EmployeeID, &p.Email); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListMeetings(ctx context.Context, workspaceID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE workspace_id = $1 ORDER BY starts_at DESC`, workspaceID)
}

func (s *Store) ListMeetingsByOrganizer(ctx context.Context, workspaceID, organizerID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE workspace_id = $1 AND organizer_user_id = $2 ORDER BY starts_at DESC`, workspaceID, organizerID)
}

func (s *Store) queryMeetings(ctx context.Context, sql string, args ...any) ([]Meeting, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meeting
	for rows.Next() {
		m, err := scanMeeting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMeeting(ctx context.Context, id uuid.UUID) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+meetingCols+` FROM meetings WHERE id = $1`, id)
	return scanMeeting(row)
}

func (s *Store) CancelMeeting(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now() WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd backend && env -u GOROOT go build ./...`
Expected: builds with no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/
git commit -m "feat(meetings): postgres models + meeting/employee repositories"
```

---

### Task 6: Application service — meetings

**Files:**

- Modify: `backend/internal/application/services.go:20-24` (add `Calendar` field)
- Create: `backend/internal/application/meeting_service.go`

- [ ] **Step 1: Add the Calendar port to Services**

Modify the `Services` struct in `backend/internal/application/services.go` (lines 20-24) to:

```go
type Services struct {
	Store    *postgres.Store
	Cipher   *crypto.TokenCipher
	Queue    *asynqqueue.Client
	Calendar CalendarService
}
```

- [ ] **Step 2: Write the meeting service**

`backend/internal/application/meeting_service.go`:

```go
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// CreateMeetingInput is the transport-level payload (strings as received over HTTP).
type CreateMeetingInput struct {
	Dept         string
	Type         string
	Host         string
	Date         string // YYYY-MM-DD
	Start        string // HH:MM
	End          string // HH:MM
	Recurrence   string
	Description  string
	Participants []postgres.MeetingParticipant
}

func (s *Services) ListEmployees(ctx context.Context, workspaceID uuid.UUID) ([]postgres.Employee, error) {
	return s.Store.ListEmployees(ctx, workspaceID)
}

func (s *Services) ListMeetings(ctx context.Context, workspaceID, userID uuid.UUID) ([]postgres.Meeting, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	// Workspace owner sees all meetings; everyone else sees their own (ТЗ §2).
	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return s.Store.ListMeetings(ctx, workspaceID)
	}
	return s.Store.ListMeetingsByOrganizer(ctx, workspaceID, userID)
}

func (s *Services) GetMeeting(ctx context.Context, id uuid.UUID) (postgres.Meeting, error) {
	m, err := s.Store.GetMeeting(ctx, id)
	if err != nil {
		return m, err
	}
	m.Participants, err = s.Store.ListParticipants(ctx, id)
	return m, err
}

func (s *Services) CreateMeeting(ctx context.Context, workspaceID, organizerID uuid.UUID, in CreateMeetingInput) (postgres.Meeting, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	startsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Start, loc)
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("bad start time: %w", err)
	}
	endsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.End, loc)
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("bad end time: %w", err)
	}

	rec := meeting.Recurrence(orDefault(in.Recurrence, string(meeting.Once)))
	dom := meeting.Input{
		Dept: in.Dept, Type: in.Type, Host: in.Host,
		StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: in.Description,
	}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, err
	}

	name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)

	var emails []string
	for _, p := range in.Participants {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	cal, err := s.Calendar.CreateEvent(ctx, CalendarEvent{
		Title: name, Description: in.Description,
		Start: startsAt, End: endsAt, AttendeeEmails: emails,
	})
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("calendar: %w", err)
	}

	m, err := s.Store.CreateMeeting(ctx, postgres.Meeting{
		WorkspaceID: workspaceID, OrganizerUserID: &organizerID,
		Dept: in.Dept, Type: in.Type, Host: in.Host,
		StartsAt: startsAt.UTC(), EndsAt: endsAt.UTC(),
		Recurrence: string(rec), Name: name, Description: in.Description,
		GoogleEventID: cal.EventID, MeetLink: cal.MeetLink,
	})
	if err != nil {
		return postgres.Meeting{}, err
	}
	if len(in.Participants) > 0 {
		if err := s.Store.AddParticipants(ctx, m.ID, in.Participants); err != nil {
			return m, err
		}
		m.Participants = in.Participants
	}
	return m, nil
}

func (s *Services) CancelMeeting(ctx context.Context, id uuid.UUID) error {
	m, err := s.Store.GetMeeting(ctx, id)
	if err != nil {
		return err
	}
	if m.GoogleEventID != "" {
		_ = s.Calendar.DeleteEvent(ctx, m.GoogleEventID)
	}
	return s.Store.CancelMeeting(ctx, id)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd backend && env -u GOROOT go build ./...`
Expected: builds (note: `cmd/server` wiring is updated in Task 7; build of the application package alone succeeds).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/application/
git commit -m "feat(meetings): application service (create/list/get/cancel)"
```

---

### Task 7: HTTP handlers + route wiring + stub injection

**Files:**

- Create: `backend/internal/delivery/http/handlers/meetings.go`
- Modify: `backend/internal/delivery/http/app.go:83-90` (inject Calendar stub) and `:126-132` (routes)

- [ ] **Step 1: Write the handlers**

`backend/internal/delivery/http/handlers/meetings.go`:

```go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/copy"
)

func (a *API) ListEmployees(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	list, err := a.App.ListEmployees(c.Context(), wid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (a *API) ListMeetings(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	uid, _ := c.Locals("user_id").(uuid.UUID)
	list, err := a.App.ListMeetings(c.Context(), wid, uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (a *API) CreateMeeting(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	uid, _ := c.Locals("user_id").(uuid.UUID)
	var body struct {
		Dept         string                        `json:"dept"`
		Type         string                        `json:"type"`
		Host         string                        `json:"host"`
		Date         string                        `json:"date"`
		Start        string                        `json:"start"`
		End          string                        `json:"end"`
		Recurrence   string                        `json:"recurrence"`
		Description  string                        `json:"description"`
		Participants []postgres.MeetingParticipant `json:"participants"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	m, err := a.App.CreateMeeting(c.Context(), wid, uid, application.CreateMeetingInput{
		Dept: body.Dept, Type: body.Type, Host: body.Host,
		Date: body.Date, Start: body.Start, End: body.End,
		Recurrence: body.Recurrence, Description: body.Description,
		Participants: body.Participants,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

func (a *API) GetMeeting(c *fiber.Ctx) error {
	mid, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	m, err := a.App.GetMeeting(c.Context(), mid)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, copy.APIError("not_found"))
	}
	return c.JSON(m)
}

func (a *API) DeleteMeeting(c *fiber.Ctx) error {
	mid, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	if err := a.App.CancelMeeting(c.Context(), mid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 2: Inject the stub Calendar adapter**

In `backend/internal/delivery/http/app.go`, add the import (with the other infrastructure imports near line 19-24):

```go
	calendarstub "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/stub"
```

Then change the `api.App` construction (lines 83-90) so `Services` includes the Calendar:

```go
	api := &handlers.API{
		App:     &application.Services{Store: store, Cipher: cipher, Queue: queue, Calendar: calendarstub.New()},
		Bot:     tg,
		RDB:     rdb,
		Log:     log,
		TMA:     telegram.NewInitDataValidator(cfg.BotToken),
		Version: os.Getenv("APP_VERSION"),
	}
```

- [ ] **Step 3: Register routes**

In `backend/internal/delivery/http/app.go`, after the scenario routes (line 132, `ws.Get("/scenarios/:sid/runs", api.ListRuns)`) add:

```go
	ws.Get("/employees", api.ListEmployees)
	ws.Get("/meetings", api.ListMeetings)
	ws.Post("/meetings", api.CreateMeeting)
	ws.Get("/meetings/:mid", api.GetMeeting)
	ws.Delete("/meetings/:mid", api.DeleteMeeting)
```

- [ ] **Step 4: Build and vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/
git commit -m "feat(meetings): REST handlers + routes + stub calendar wiring"
```

---

### Task 8: End-to-end smoke coverage

**Files:**

- Modify: `backend/test/smoke/smoke_test.go` (append a meetings sub-test)

- [ ] **Step 1: Append the meetings E2E test**

Add this function to `backend/test/smoke/smoke_test.go` (same `package smoke`, build tag already `//go:build smoke`):

```go
func TestSmokeMeetings(t *testing.T) {
	// workspace
	slug := "smoke-mtg-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ws := must(t, http.MethodPost, "/api/workspaces", token,
		fmt.Sprintf(`{"name":"Smoke Meetings","slug":%q}`, slug))
	wid := firstID(t, ws)

	// create meeting (participant by email; stub calendar returns a Meet link)
	body := `{"dept":"Разработка","type":"Планёрка","host":"Иванов А.А.",` +
		`"date":"2025-06-02","start":"10:00","end":"11:00","recurrence":"weekly",` +
		`"participants":[{"email":"a@example.com"}]}`
	created := must(t, http.MethodPost, "/api/workspaces/"+wid+"/meetings", token, body)
	if !strings.Contains(created, `"meet_link":"https://meet.google.com/`) {
		t.Fatalf("no meet link in created meeting: %s", created)
	}
	mid := firstID(t, created)

	// list (owner sees it)
	list := must(t, http.MethodGet, "/api/workspaces/"+wid+"/meetings", token, "")
	if !strings.Contains(list, mid) {
		t.Fatalf("created meeting not in list: %s", list)
	}

	// get
	got := must(t, http.MethodGet, "/api/workspaces/"+wid+"/meetings/"+mid, token, "")
	if !strings.Contains(got, `"name":"Разработка | Планёрка | Иванов А.А. | 2025-06-02 | Еженедельно"`) {
		t.Fatalf("unexpected meeting name: %s", got)
	}

	// cancel
	if code, _ := do(t, http.MethodDelete, "/api/workspaces/"+wid+"/meetings/"+mid, token, ""); code != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", code)
	}

	t.Logf("meetings smoke OK (workspace %s, meeting %s)", wid, mid)
}
```

- [ ] **Step 2: Verify it compiles (build-tagged)**

Run: `cd backend && env -u GOROOT go vet -tags=smoke ./test/smoke/...`
Expected: no errors.

- [ ] **Step 3: Run the smoke test against a live server**

From the repo root, in one terminal: `make up && make migrate && make backend`
In another: `make smoke`
Expected: `TestSmokeMeetings` PASSES (the new workspace's owner is the smoke user, so it appears in the list; stub calendar yields the Meet link). Note: the pre-existing `TestSmoke` may still fail at `chat not linked` — that is a known, separate domain issue and not introduced here.

- [ ] **Step 4: Commit**

```bash
git add backend/test/smoke/smoke_test.go
git commit -m "test(meetings): E2E smoke coverage for meeting CRUD"
```

---

### Task 9: Docs — update API + MEETINGS status

**Files:**

- Modify: `docs/API.md` (add meeting endpoints under the workspace group)
- Modify: `docs/MEETINGS.md` (mark increment-1 backend as done)

- [ ] **Step 1: Add endpoints to `docs/API.md`**

Under the workspace endpoint list, add:

```markdown
- `GET /api/workspaces/:id/employees`
- `GET /api/workspaces/:id/meetings`
- `POST /api/workspaces/:id/meetings`
- `GET /api/workspaces/:id/meetings/:mid`
- `DELETE /api/workspaces/:id/meetings/:mid`
```

- [ ] **Step 2: Update `docs/MEETINGS.md` Backend section**

Change the "Backend (planned)" heading note to record that increment 1 (meeting CRUD over REST + stubbed calendar) is implemented, Google adapter still pending. Add one line:

```markdown
> Increment 1 (done): meeting CRUD over REST (`/api/workspaces/:id/meetings`, `/employees`) with a stubbed `CalendarService`. Real Google Calendar adapter, recurrence series, conflict detection, checker, notifications, and bot registration remain planned.
```

- [ ] **Step 3: Format docs and commit**

Run: `make fmt-check` (expected: green; if docs reflow, run `make fmt` first)

```bash
git add docs/API.md docs/MEETINGS.md
git commit -m "docs(meetings): document increment-1 REST endpoints"
```

---

## Done criteria

- `make lint` → 0 issues; `make test` → all pass; `make typecheck` → 0 errors; `make fmt-check` → green.
- `make smoke` → `TestSmokeMeetings` passes against a live server.
- Endpoints live under `/api/workspaces/:id/meetings` and `/employees`, gated by existing auth + workspace ACL, backed by the stub calendar.
