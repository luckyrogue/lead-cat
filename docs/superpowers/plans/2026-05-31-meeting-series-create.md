# Recurring-series creation engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make recurrence real — when a meeting is created with `recurrence != once`, materialize it into individual occurrence rows (linked by `series_id`, bounded by a required `recurrence_until`), each its own Google event, persisted atomically.

**Architecture:** A pure domain `Occurrences` expander produces the spans; `Services.CreateMeeting` gains a series branch that creates one Google event per span (with best-effort compensation on failure), then inserts all rows + participants in one DB transaction (`CreateMeetingSeries`), and enqueues the `meeting:created` notification once. `recurrence == once` keeps the existing single path.

**Tech Stack:** Go, pgx (tx), goose, google/uuid, asynq.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-series-create-design.md`

**Conventions:** Run Go from `backend/` with `env -u GOROOT`. Module `github.com/Jaryq-Lab/notify-bot`. Build check: `env -u GOROOT go build ./...`.

---

## Task 1: domain — `Occurrences` expander

**Files:**
- Create: `backend/internal/domain/meeting/recurrence.go`
- Test: `backend/internal/domain/meeting/recurrence_test.go`

- [ ] **Step 1: Write the failing test.** Create `backend/internal/domain/meeting/recurrence_test.go`:

```go
package meeting

import (
	"errors"
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 10, 0, 0, 0, time.UTC)
}

func TestOccurrences_Once(t *testing.T) {
	spans, err := Occurrences(d(2026, 6, 1), d(2026, 6, 1).Add(time.Hour), Once, time.Time{})
	if err != nil || len(spans) != 1 {
		t.Fatalf("once: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Weekly(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 6, 22))
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 4 { // 1, 8, 15, 22
		t.Fatalf("weekly want 4, got %d", len(spans))
	}
	if !spans[1].Start.Equal(d(2026, 6, 8)) {
		t.Fatalf("2nd occurrence = %v", spans[1].Start)
	}
	if spans[0].End.Sub(spans[0].Start) != time.Hour {
		t.Fatalf("duration not preserved")
	}
}

func TestOccurrences_Daily(t *testing.T) {
	start := d(2026, 6, 1)
	spans, _ := Occurrences(start, start.Add(time.Hour), Daily, d(2026, 6, 3))
	if len(spans) != 3 {
		t.Fatalf("daily want 3, got %d", len(spans))
	}
}

func TestOccurrences_Monthly(t *testing.T) {
	start := d(2026, 1, 15)
	spans, _ := Occurrences(start, start.Add(time.Hour), Monthly, d(2026, 4, 15))
	if len(spans) != 4 { // Jan,Feb,Mar,Apr 15
		t.Fatalf("monthly want 4, got %d", len(spans))
	}
}

func TestOccurrences_UntilEqualsStart(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 6, 1))
	if err != nil || len(spans) != 1 {
		t.Fatalf("until==start: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Errors(t *testing.T) {
	start := d(2026, 6, 1)
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, time.Time{}); err == nil {
		t.Fatal("recurring without until must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 5, 1)); err == nil {
		t.Fatal("until before start must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Daily, d(2030, 1, 1)); !errors.Is(err, ErrTooManyOccurrences) {
		t.Fatalf("want ErrTooManyOccurrences, got %v", err)
	}
}
```

- [ ] **Step 2: Run, verify fail.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/domain/meeting/ -run TestOccurrences -v` → FAIL (undefined Occurrences/Span/ErrTooManyOccurrences).

- [ ] **Step 3: Implement.** Create `backend/internal/domain/meeting/recurrence.go`:

```go
package meeting

import (
	"errors"
	"time"
)

// Span is one occurrence's start/end.
type Span struct {
	Start time.Time
	End   time.Time
}

const maxOccurrences = 100

// ErrTooManyOccurrences means the series would exceed the materialization cap.
var ErrTooManyOccurrences = errors.New("too many occurrences (max 100)")

// ErrRecurrenceWindow means a recurring series has a missing or invalid end date.
var ErrRecurrenceWindow = errors.New("recurring meeting needs a valid end date (>= start)")

// Occurrences expands a recurring meeting into spans from start to until
// (inclusive by date). Once returns a single span and ignores until. Non-once
// requires a valid until (date >= start's date) and is capped at maxOccurrences.
func Occurrences(start, end time.Time, r Recurrence, until time.Time) ([]Span, error) {
	if r == Once {
		return []Span{{Start: start, End: end}}, nil
	}
	if until.IsZero() {
		return nil, ErrRecurrenceWindow
	}
	startDay := dateOnly(start)
	untilDay := dateOnly(until)
	if untilDay.Before(startDay) {
		return nil, ErrRecurrenceWindow
	}
	dur := end.Sub(start)
	var spans []Span
	for cur := start; !dateOnly(cur).After(untilDay); cur = next(cur, r) {
		if len(spans) >= maxOccurrences {
			return nil, ErrTooManyOccurrences
		}
		spans = append(spans, Span{Start: cur, End: cur.Add(dur)})
	}
	return spans, nil
}

func next(t time.Time, r Recurrence) time.Time {
	switch r {
	case Daily:
		return t.AddDate(0, 0, 1)
	case Weekly:
		return t.AddDate(0, 0, 7)
	case Biweekly:
		return t.AddDate(0, 0, 14)
	case Monthly:
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(0, 0, 1)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
```

- [ ] **Step 4: Run, verify pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/domain/meeting/ -v` → all PASS (new + existing naming/validate tests).

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/domain/meeting/ && git commit -m "feat(meetings): domain Occurrences series expander

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: migration + model + repo (series columns + transactional batch insert)

**Files:**
- Create: `backend/migrations/20260531130000_meeting_series.sql`
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go` (Meeting struct)
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Migration.** Create `backend/migrations/20260531130000_meeting_series.sql`:

```sql
-- +goose Up
ALTER TABLE meetings ADD COLUMN series_id UUID;
CREATE INDEX meetings_series_idx ON meetings (series_id);

-- +goose Down
DROP INDEX IF EXISTS meetings_series_idx;
ALTER TABLE meetings DROP COLUMN IF EXISTS series_id;
```

- [ ] **Step 2: Model fields.** In `backend/internal/infrastructure/persistence/postgres/models.go`, add two fields to the `Meeting` struct (after `Status`, before `Participants`):

```go
	SeriesID        *uuid.UUID           `json:"series_id,omitempty"`
	RecurrenceUntil *time.Time           `json:"recurrence_until,omitempty"`
```

(Confirm `time` is imported in models.go — it is, used by `StartsAt`.)

- [ ] **Step 3: Extend column lists + scanMeeting + a shared insert.** In `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`:

(a) Extend `meetingCols` and `meetingColsM` by appending the two columns:

```go
const meetingCols = `id, workspace_id, organizer_user_id, dept, type, host,
	starts_at, ends_at, recurrence, name, description, google_event_id, meet_link, status,
	series_id, recurrence_until`

// meetingColsM is meetingCols qualified with the `m` alias for joins.
// Keep its columns (and the scanMeeting scan order) in sync with meetingCols.
const meetingColsM = `m.id, m.workspace_id, m.organizer_user_id, m.dept, m.type, m.host,
	m.starts_at, m.ends_at, m.recurrence, m.name, m.description, m.google_event_id, m.meet_link, m.status,
	m.series_id, m.recurrence_until`
```

(b) Extend `scanMeeting` to scan the two new columns (append to the Scan call):

```go
func scanMeeting(row interface {
	Scan(dest ...any) error
}) (Meeting, error) {
	var m Meeting
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.OrganizerUserID, &m.Dept, &m.Type, &m.Host,
		&m.StartsAt, &m.EndsAt, &m.Recurrence, &m.Name, &m.Description, &m.GoogleEventID, &m.MeetLink, &m.Status,
		&m.SeriesID, &m.RecurrenceUntil)
	return m, err
}
```

(c) Add a shared insert statement + args helper (after `scanMeeting`), and rewrite `CreateMeeting` to use them:

```go
const insertMeetingSQL = `
	INSERT INTO meetings (workspace_id, organizer_user_id, dept, type, host,
		starts_at, ends_at, recurrence, name, description, google_event_id, meet_link,
		series_id, recurrence_until)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	RETURNING ` + meetingCols

func meetingInsertArgs(m Meeting) []any {
	return []any{m.WorkspaceID, m.OrganizerUserID, m.Dept, m.Type, m.Host,
		m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description, m.GoogleEventID, m.MeetLink,
		m.SeriesID, m.RecurrenceUntil}
}

func (s *Store) CreateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	return scanMeeting(s.pool.QueryRow(ctx, insertMeetingSQL, meetingInsertArgs(m)...))
}
```

(Replace the existing `CreateMeeting` body, which had the inline INSERT, with this.)

- [ ] **Step 4: Add the transactional batch insert.** In the same file, add (after `CreateMeeting`):

```go
// CreateMeetingSeries inserts all meetings + their participants in one
// transaction (all-or-nothing) and returns the inserted rows with IDs.
func (s *Store) CreateMeetingSeries(ctx context.Context, ms []Meeting, ps []MeetingParticipant) ([]Meeting, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	out := make([]Meeting, 0, len(ms))
	for _, m := range ms {
		created, err := scanMeeting(tx.QueryRow(ctx, insertMeetingSQL, meetingInsertArgs(m)...))
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			if _, err := tx.Exec(ctx, `
				INSERT INTO meeting_participants (meeting_id, employee_id, email)
				VALUES ($1, $2, $3) ON CONFLICT (meeting_id, email) DO NOTHING`, created.ID, p.EmployeeID, p.Email); err != nil {
				return nil, err
			}
		}
		created.Participants = ps
		out = append(out, created)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 5: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
Expected: clean. (No DB harness — build/vet is the gate. The two new columns are appended to every `scanMeeting`-based query consistently via the shared `meetingCols`.)

- [ ] **Step 6: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/migrations/20260531130000_meeting_series.sql backend/internal/infrastructure/persistence/postgres/ && git commit -m "feat(meetings): series_id column + transactional CreateMeetingSeries

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: application — series creation in CreateMeeting

**Files:**
- Modify: `backend/internal/application/meeting_service.go` (`CreateMeetingInput`, `CreateMeeting`, new `createSeriesEvents` helper)
- Test: `backend/internal/application/series_test.go`
- Modify: `backend/internal/delivery/http/handlers/meetings.go` (request field)

- [ ] **Step 1: Write the failing helper test.** Create `backend/internal/application/series_test.go`:

```go
package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
)

type fakeCal struct {
	created  int
	deleted  int
	failAt   int // 1-based span index to fail CreateEvent on; 0 = never
}

func (f *fakeCal) CreateEvent(_ context.Context, _ CalendarEvent) (CalendarResult, error) {
	f.created++
	if f.failAt != 0 && f.created == f.failAt {
		return CalendarResult{}, errors.New("boom")
	}
	return CalendarResult{EventID: "ev", MeetLink: "https://meet/x"}, nil
}
func (f *fakeCal) UpdateEvent(_ context.Context, _ string, _ CalendarEvent) error          { return nil }
func (f *fakeCal) UpdateAttendees(_ context.Context, _ string, _ []string) error           { return nil }
func (f *fakeCal) DeleteEvent(_ context.Context, _ string) error                           { f.deleted++; return nil }

func spans(n int) []meeting.Span {
	out := make([]meeting.Span, n)
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	for i := range out {
		s := base.AddDate(0, 0, 7*i)
		out[i] = meeting.Span{Start: s, End: s.Add(time.Hour)}
	}
	return out
}

func TestCreateSeriesEvents_OK(t *testing.T) {
	cal := &fakeCal{}
	names := []string{"a", "b", "c"}
	evs, err := createSeriesEvents(context.Background(), cal, names, "desc", []string{"x@y"}, spans(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 || cal.created != 3 || cal.deleted != 0 {
		t.Fatalf("evs=%d created=%d deleted=%d", len(evs), cal.created, cal.deleted)
	}
	if evs[0].Name != "a" || evs[0].EventID != "ev" {
		t.Fatalf("bad event row: %+v", evs[0])
	}
}

func TestCreateSeriesEvents_CompensatesOnFailure(t *testing.T) {
	cal := &fakeCal{failAt: 3} // 2 succeed, 3rd fails
	names := []string{"a", "b", "c"}
	_, err := createSeriesEvents(context.Background(), cal, names, "desc", nil, spans(3))
	if err == nil {
		t.Fatal("expected error")
	}
	if cal.created != 3 || cal.deleted != 2 {
		t.Fatalf("compensation: created=%d deleted=%d (want 3 created, 2 deleted)", cal.created, cal.deleted)
	}
}
```

- [ ] **Step 2: Run, verify fail.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestCreateSeriesEvents -v` → FAIL (undefined createSeriesEvents).

- [ ] **Step 3: Implement the helper.** In `backend/internal/application/meeting_service.go`, add (near `CreateMeeting`):

```go
// seriesEvent is a created Google event paired with its span and computed name.
type seriesEvent struct {
	Span     meeting.Span
	Name     string
	EventID  string
	MeetLink string
}

// createSeriesEvents creates one Google event per span (names[i] is the title for
// spans[i]). On any failure it best-effort deletes the events already created and
// returns the error, so no partial series leaks.
func createSeriesEvents(ctx context.Context, cal CalendarService, names []string, description string, emails []string, spans []meeting.Span) ([]seriesEvent, error) {
	var created []seriesEvent
	for i, sp := range spans {
		res, err := cal.CreateEvent(ctx, CalendarEvent{
			Title: names[i], Description: description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
		})
		if err != nil {
			for _, c := range created {
				_ = cal.DeleteEvent(ctx, c.EventID)
			}
			return nil, fmt.Errorf("calendar: %w", err)
		}
		created = append(created, seriesEvent{Span: sp, Name: names[i], EventID: res.EventID, MeetLink: res.MeetLink})
	}
	return created, nil
}
```

- [ ] **Step 4: Run, verify pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestCreateSeriesEvents -v` → both PASS.

- [ ] **Step 5: Add `RecurrenceUntil` to the input + the series branch in `CreateMeeting`.** In `backend/internal/application/meeting_service.go`:

(a) Add the field to `CreateMeetingInput` (after `Recurrence`):

```go
	RecurrenceUntil string // YYYY-MM-DD; required when Recurrence != once
```

(b) In `CreateMeeting`, after the existing `dom.Validate()` block (which validates the base fields) and before the current `name := ...` line, insert the until-parse + series branch. Replace everything from `name := meeting.GenerateName(...)` through the end of the function with:

```go
	var until time.Time
	if in.RecurrenceUntil != "" {
		until, err = time.ParseInLocation("2006-01-02", in.RecurrenceUntil, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad recurrence_until", ErrInvalidInput)
		}
	}
	spansList, err := meeting.Occurrences(startsAt, endsAt, rec, until)
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	var emails []string
	for _, p := range in.Participants {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := s.Calendar.For(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}

	// Single (non-recurring) meeting: existing path.
	if rec == meeting.Once {
		name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)
		cal, err := calSvc.CreateEvent(ctx, CalendarEvent{
			Title: name, Description: in.Description, Start: startsAt, End: endsAt, AttendeeEmails: emails,
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
		s.enqueueCreated(ctx, workspaceID, m.ID)
		return m, nil
	}

	// Recurring series: materialize occurrences (Google first w/ compensation, then one DB tx).
	names := make([]string, len(spansList))
	for i, sp := range spansList {
		names[i] = meeting.GenerateName(in.Dept, in.Type, in.Host, sp.Start, rec)
	}
	evs, err := createSeriesEvents(ctx, calSvc, names, in.Description, emails, spansList)
	if err != nil {
		return postgres.Meeting{}, err
	}
	seriesID := uuid.New()
	untilUTC := until.UTC()
	rows := make([]postgres.Meeting, len(evs))
	for i, e := range evs {
		rows[i] = postgres.Meeting{
			WorkspaceID: workspaceID, OrganizerUserID: &organizerID,
			Dept: in.Dept, Type: in.Type, Host: in.Host,
			StartsAt: e.Span.Start.UTC(), EndsAt: e.Span.End.UTC(),
			Recurrence: string(rec), Name: e.Name, Description: in.Description,
			GoogleEventID: e.EventID, MeetLink: e.MeetLink,
			SeriesID: &seriesID, RecurrenceUntil: &untilUTC,
		}
	}
	created, err := s.Store.CreateMeetingSeries(ctx, rows, in.Participants)
	if err != nil {
		for _, e := range evs {
			_ = calSvc.DeleteEvent(ctx, e.EventID) // best-effort: keep DB all-or-nothing
		}
		return postgres.Meeting{}, err
	}
	anchor := created[0]
	s.enqueueCreated(ctx, workspaceID, anchor.ID)
	return anchor, nil
```

(c) Add the small `enqueueCreated` helper (extracted from the existing best-effort enqueue block so both paths reuse it) — place it after `CreateMeeting`:

```go
// enqueueCreated best-effort enqueues the meeting-created notification (once).
func (s *Services) enqueueCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) {
	if s.Queue == nil {
		return
	}
	if err := s.Queue.EnqueueMeetingCreated(ctx, workspaceID, meetingID); err != nil && s.Log != nil {
		s.Log.Warn("enqueue meeting created",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}
```

(Remove the old inline `if s.Queue != nil { ... EnqueueMeetingCreated ... }` block that previously ended `CreateMeeting`, since the once path now calls `enqueueCreated` and returns.)

- [ ] **Step 6: Map the request field in the HTTP handler.** In `backend/internal/delivery/http/handlers/meetings.go`, add `RecurrenceUntil` to the `CreateMeeting` request struct and pass it through:

In the anonymous `body` struct, after `Recurrence`:
```go
		RecurrenceUntil string                        `json:"recurrence_until"`
```
In the `application.CreateMeetingInput{...}` literal, add:
```go
		RecurrenceUntil: body.RecurrenceUntil,
```

- [ ] **Step 7: Build + test.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/application/ -v`
Expected: build OK; `TestCreateSeriesEvents_*` + existing application tests PASS. (The full series orchestration — `CreateMeetingSeries` tx — is build-verified; the Google compensation is covered by `TestCreateSeriesEvents_CompensatesOnFailure`.)

- [ ] **Step 8: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/ backend/internal/delivery/http/handlers/meetings.go && git commit -m "feat(meetings): materialize recurring series on create

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: full verification + docs

**Files:**
- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Run the full suite.** From the repo root: `make test && make lint && make build`. (Fallback: `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && env -u GOROOT go build ./...`; then `cd backend && gofmt -l .` and `gofmt -w` any listed file, re-run lint.) If a real failure occurs, STOP and report BLOCKED.

- [ ] **Step 2: Document.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Employee schedule view (§4.6, done)" line, add:

```markdown
> **Recurring-series creation (done):** creating a meeting with `recurrence != once` + a required `recurrence_until` materializes the series into individual occurrence rows (linked by `series_id`), each its own Google event and reminders. Occurrences are expanded by the pure `meeting.Occurrences` (cap 100). Google events are created first (best-effort compensation on failure), then all rows + participants are inserted in one DB transaction (`CreateMeetingSeries`); the `meeting:created` DM fires once per series. Tradeoffs of materialization: each occurrence has its own Meet link, and series are bounded by `until` (no open-ended/re-materialization yet). Editing a series "this/whole" (§4.4.2) is the next increment.
```

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document recurring-series creation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** domain `Occurrences` + cap + validation (Task 1) · `series_id` migration + model + cols/scanMeeting + `CreateMeetingSeries` tx (Task 2) · `CreateMeetingInput.RecurrenceUntil` + HTTP field + once-path-preserved + series branch (Google-first compensation, DB tx, enqueue-once) (Task 3) · testing (Tasks 1,3) · docs (Task 4). Out-of-scope (§4.4.2 edit, re-materialization, single Meet link, series delete) recorded in spec + Task 4 note. All covered.
- **Type consistency:** `meeting.Span{Start,End}` + `Occurrences(start,end,r,until) ([]Span,error)` + `ErrTooManyOccurrences`/`ErrRecurrenceWindow` (Task 1) used in Task 3. `Meeting.SeriesID *uuid.UUID` + `RecurrenceUntil *time.Time` (Task 2) set in Task 3's rows and scanned by `scanMeeting`. `insertMeetingSQL`/`meetingInsertArgs`/`CreateMeetingSeries` (Task 2) called by Task 3. `seriesEvent{Span,Name,EventID,MeetLink}` + `createSeriesEvents(ctx,cal,names,description,emails,spans)` (Task 3) used by the series branch. `enqueueCreated` (Task 3) used by both paths.
- **No placeholders:** every code/command step is concrete. The once-path code in Task 3 Step 5 is shown in full (not "as before") because the surrounding function tail is rewritten.
```