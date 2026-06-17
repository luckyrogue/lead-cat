# WS2b — Application CQRS Handler Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the meeting CQRS handlers (`command.Meetings` Create/Update/Cancel + pure helpers, and `query.Meetings`) with fast, DB-free unit tests using in-memory fakes for the `Store`/`CalendarProvider`/`JobQueue` ports.

**Architecture:** White-box test files in `package command` define hand-written, call-recording fakes implementing the three ports; tests assert observable effects (calendar calls, persisted values, enqueued jobs, returned errors). Query tests live in `package query` with a small fake for its own ports. No DB, no testcontainers.

**Tech Stack:** Go 1.26, stdlib `testing`, `errors.Is`, zap (NewNop), uuid.

**Standing constraints (every task):**
- Work on `main`; no branches. Commit per task; human pushes on request. Stage only the explicit paths listed; never `git add -A`; `git status` before staging. Run Go with `env -u GOROOT`. Commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- Do NOT touch `.github/workflows/_build.yml` or any non-test/production file. These are test-only additions.
- Ignore stale IDE diagnostics; trust `go test` + `golangci-lint`.

---

## Reference (verified signatures)

- `command.Meetings{Store, Calendar, Queue, Log}`.
- `Store` (interface): `GetOrganization(ctx,id)(model.Organization,error)`, `GetMeeting(ctx,orgID,id)(model.Meeting,error)`, `CreateMeeting(ctx,model.Meeting)(model.Meeting,error)`, `CreateMeetingSeries(ctx,[]model.Meeting,[]model.MeetingParticipant)([]model.Meeting,error)`, `UpdateMeeting(ctx,orgID,id,model.Meeting)error`, `CancelMeeting(ctx,orgID,id)error`, `AddParticipants(ctx,meetingID,[]model.MeetingParticipant)error`.
- `CalendarProvider`: `For(ctx,orgID)(docalendar.Service,error)`.
- `docalendar.Service` (4 methods): `CreateEvent(ctx,CalendarEvent)(CalendarResult,error)`, `UpdateEvent(ctx,eventID,CalendarEvent)error`, `UpdateAttendees(ctx,eventID,[]string)error`, `DeleteEvent(ctx,eventID)error`. `CalendarEvent{Title,Description,Start,End,AttendeeEmails}`; `CalendarResult{EventID,MeetLink}`.
- `JobQueue`: `EnqueueMeetingCreated/Updated/Cancelled(ctx,orgID,meetingID)error`.
- `CreateInput{Dept,Type,Host,Date,Start,End,Recurrence,RecurrenceUntil,RecurrenceDays,Description,Participants,Timezone}`; `UpdateInput{Dept,Type,Host,Date,Start,End,Recurrence,Description *string; Timezone string}`.
- Errors: `ErrInvalidInput`, `ErrForbidden`.
- Behavior: `CreateMeeting` → GetOrganization → parse times in tz → `meeting.Input.Validate` → `Occurrences` → `Calendar.For` → once: `CreateEvent`+`Store.CreateMeeting`; recurring: series events + `Store.CreateMeetingSeries` → enqueue created. `UpdateMeeting` → GetMeeting → GetOrganization → `ownerOrOrganizer` (else `ErrForbidden`) → `ApplyMeetingUpdate` → **if `GoogleEventID != ""`** `UpdateEvent` → `Store.UpdateMeeting` → enqueue updated. `CancelMeeting` → permission → **if status != "scheduled" return nil** → if `GoogleEventID != ""` best-effort `DeleteEvent` → `Store.CancelMeeting` → enqueue cancelled. `ownerOrOrganizer(org, organizerUserID, userID)`: true if org.OwnerUserID==userID OR organizerUserID==userID.
- `query.Meetings{App meetingListApp}`; `meetingListApp.EmployeeSchedule(ctx,email,from,to)([]model.Meeting,error)`; `Schedule` delegates. `MeetingDTO(ctx, store meetingStore, m, loc)` where `meetingStore` = `GetUserByID(ctx,id)(model.User,error)` + `ListParticipants(ctx,meetingID)([]model.MeetingParticipant,error)`; formats Date `2006-01-02`, Start/End `15:04` in `loc`, resolves Organizer email, lists participant emails.

---

## File Structure

| File | Responsibility | Task |
|---|---|---|
| `apps/backend/internal/application/command/fakes_test.go` | In-memory fakes + interface assertions | 1 |
| `apps/backend/internal/application/command/meetings_create_test.go` | CreateMeeting (once/recurring/invalid/calendar-fail) | 1 |
| `apps/backend/internal/application/command/meetings_update_cancel_test.go` | Update + Cancel (incl. permission) | 2 |
| `apps/backend/internal/application/command/meetings_helpers_test.go` | ApplyMeetingUpdate, ownerOrOrganizer | 3 |
| `apps/backend/internal/application/query/meetings_test.go` | Schedule, MeetingDTO | 4 |

---

### Task 1: Fakes + CreateMeeting tests

**Files:**
- Create: `apps/backend/internal/application/command/fakes_test.go`
- Create: `apps/backend/internal/application/command/meetings_create_test.go`

- [ ] **Step 1: Write the fakes**

Create `apps/backend/internal/application/command/fakes_test.go`:

```go
package command

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

// Compile-time proof the fakes satisfy the real ports.
var (
	_ Store              = (*fakeStore)(nil)
	_ CalendarProvider   = (*fakeCalProvider)(nil)
	_ JobQueue           = (*fakeQueue)(nil)
	_ docalendar.Service = (*fakeCalService)(nil)
)

type fakeStore struct {
	org       model.Organization
	orgErr    error
	meetings  map[uuid.UUID]model.Meeting
	created   []model.Meeting
	series    [][]model.Meeting
	updated   []model.Meeting
	cancelled []uuid.UUID
}

func newFakeStore() *fakeStore { return &fakeStore{meetings: map[uuid.UUID]model.Meeting{}} }

func (f *fakeStore) GetOrganization(_ context.Context, _ uuid.UUID) (model.Organization, error) {
	if f.orgErr != nil {
		return model.Organization{}, f.orgErr
	}
	return f.org, nil
}

func (f *fakeStore) GetMeeting(_ context.Context, _, id uuid.UUID) (model.Meeting, error) {
	m, ok := f.meetings[id]
	if !ok {
		return model.Meeting{}, errors.New("not found")
	}
	return m, nil
}

func (f *fakeStore) CreateMeeting(_ context.Context, m model.Meeting) (model.Meeting, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	f.meetings[m.ID] = m
	f.created = append(f.created, m)
	return m, nil
}

func (f *fakeStore) CreateMeetingSeries(_ context.Context, ms []model.Meeting, ps []model.MeetingParticipant) ([]model.Meeting, error) {
	out := make([]model.Meeting, 0, len(ms))
	for _, m := range ms {
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		m.Participants = ps
		f.meetings[m.ID] = m
		out = append(out, m)
	}
	f.series = append(f.series, out)
	return out, nil
}

func (f *fakeStore) UpdateMeeting(_ context.Context, _, id uuid.UUID, m model.Meeting) error {
	f.meetings[id] = m
	f.updated = append(f.updated, m)
	return nil
}

func (f *fakeStore) CancelMeeting(_ context.Context, _, id uuid.UUID) error {
	if m, ok := f.meetings[id]; ok {
		m.Status = "cancelled"
		f.meetings[id] = m
	}
	f.cancelled = append(f.cancelled, id)
	return nil
}

func (f *fakeStore) AddParticipants(_ context.Context, _ uuid.UUID, _ []model.MeetingParticipant) error {
	return nil
}

type fakeCalProvider struct {
	svc    *fakeCalService
	forErr error
}

func (p *fakeCalProvider) For(_ context.Context, _ uuid.UUID) (docalendar.Service, error) {
	if p.forErr != nil {
		return nil, p.forErr
	}
	return p.svc, nil
}

type fakeCalService struct {
	failCreate bool
	created    []docalendar.CalendarEvent
	updated    []string
	deleted    []string
}

func (s *fakeCalService) CreateEvent(_ context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	if s.failCreate {
		return docalendar.CalendarResult{}, errors.New("boom")
	}
	s.created = append(s.created, e)
	id := "evt-" + uuid.NewString()[:8]
	return docalendar.CalendarResult{EventID: id, MeetLink: "https://meet.google.com/" + id}, nil
}

func (s *fakeCalService) UpdateEvent(_ context.Context, eventID string, _ docalendar.CalendarEvent) error {
	s.updated = append(s.updated, eventID)
	return nil
}

func (s *fakeCalService) UpdateAttendees(_ context.Context, _ string, _ []string) error { return nil }

func (s *fakeCalService) DeleteEvent(_ context.Context, eventID string) error {
	s.deleted = append(s.deleted, eventID)
	return nil
}

type fakeQueue struct {
	createdEnq   []uuid.UUID
	updatedEnq   []uuid.UUID
	cancelledEnq []uuid.UUID
}

func (q *fakeQueue) EnqueueMeetingCreated(_ context.Context, _, meetingID uuid.UUID) error {
	q.createdEnq = append(q.createdEnq, meetingID)
	return nil
}

func (q *fakeQueue) EnqueueMeetingUpdated(_ context.Context, _, meetingID uuid.UUID) error {
	q.updatedEnq = append(q.updatedEnq, meetingID)
	return nil
}

func (q *fakeQueue) EnqueueMeetingCancelled(_ context.Context, _, meetingID uuid.UUID) error {
	q.cancelledEnq = append(q.cancelledEnq, meetingID)
	return nil
}
```

- [ ] **Step 2: Write CreateMeeting tests**

Create `apps/backend/internal/application/command/meetings_create_test.go`:

```go
package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func newMeetingsCmd(fs *fakeStore, fcp *fakeCalProvider, fq *fakeQueue) *Meetings {
	return &Meetings{Store: fs, Calendar: fcp, Queue: fq, Log: zap.NewNop()}
}

func ownerOrg() (model.Organization, uuid.UUID) {
	owner := uuid.New()
	return model.Organization{TZ: "Asia/Almaty", OwnerUserID: &owner}, owner
}

func TestCreateMeeting_Once_HappyPath(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)

	m, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "once",
		Participants: []model.MeetingParticipant{{Email: "a@x.io"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(fcs.created) != 1 {
		t.Fatalf("want 1 calendar event, got %d", len(fcs.created))
	}
	if len(fs.created) != 1 || fs.created[0].GoogleEventID == "" || fs.created[0].MeetLink == "" {
		t.Fatalf("meeting not persisted with calendar ids: %+v", fs.created)
	}
	if len(fq.createdEnq) != 1 {
		t.Fatalf("want 1 enqueue, got %d", len(fq.createdEnq))
	}
	if m.StartsAt.Location() != time.UTC {
		t.Fatalf("starts_at should be stored UTC, got %v", m.StartsAt.Location())
	}
}

func TestCreateMeeting_Recurring_Series(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, &fakeQueue{})

	_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30",
		Recurrence: "daily", RecurrenceUntil: "2026-06-03",
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if len(fcs.created) != 3 {
		t.Fatalf("want 3 calendar events (Jun 1-3), got %d", len(fcs.created))
	}
	if len(fs.series) != 1 || len(fs.series[0]) != 3 {
		t.Fatalf("want CreateMeetingSeries with 3 meetings, got %+v", fs.series)
	}
}

func TestCreateMeeting_InvalidInput(t *testing.T) {
	cases := map[string]CreateInput{
		"bad-start": {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "99:99", End: "10:30", Recurrence: "once"},
		"bad-recurrence": {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "fortnightly"},
		"bad-until": {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "daily", RecurrenceUntil: "nope"},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			fs := newFakeStore()
			org, owner := ownerOrg()
			fs.org = org
			fcs := &fakeCalService{}
			fq := &fakeQueue{}
			c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)
			_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, in)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if len(fs.created) != 0 || len(fcs.created) != 0 || len(fq.createdEnq) != 0 {
				t.Fatalf("nothing should be created/enqueued on invalid input")
			}
		})
	}
}

func TestCreateMeeting_CalendarFailure(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{failCreate: true}
	fq := &fakeQueue{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)

	_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "once",
	})
	if err == nil || !strings.Contains(err.Error(), "calendar") {
		t.Fatalf("want calendar-wrapped error, got %v", err)
	}
	if len(fs.created) != 0 || len(fq.createdEnq) != 0 {
		t.Fatalf("nothing should persist/enqueue when calendar fails")
	}
}
```

- [ ] **Step 3: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/command/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/application/command/...` — `0 issues.`

If a test FAILS, investigate whether it reveals a real behavior mismatch (e.g. the recurring path enqueues differently, or `Once` parsing differs) and report it — do NOT weaken assertions blindly. The recurring-enqueue count is intentionally NOT asserted (only series creation + calendar events) to avoid coupling to an unverified enqueue-per-occurrence detail.

- [ ] **Step 4: Commit**

```bash
git add apps/backend/internal/application/command/fakes_test.go \
  apps/backend/internal/application/command/meetings_create_test.go
git commit -m "$(cat <<'EOF'
test(command): in-memory port fakes + CreateMeeting (once/recurring/invalid/calendar-fail)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Update + Cancel tests

**Files:**
- Create: `apps/backend/internal/application/command/meetings_update_cancel_test.go`

Uses the fakes from Task 1 (same package). Helper `strPtr` defined here.

- [ ] **Step 1: Write Update + Cancel tests**

Create `apps/backend/internal/application/command/meetings_update_cancel_test.go`:

```go
package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func strPtr(s string) *string { return &s }

// seedStored puts a scheduled meeting (with a calendar event id) in the fake
// store, organized by organizerID, and sets the org owner.
func seedStored(fs *fakeStore, organizerID uuid.UUID) (uuid.UUID, model.Organization) {
	owner := uuid.New()
	org := model.Organization{TZ: "Asia/Almaty", OwnerUserID: &owner}
	fs.org = org
	id := uuid.New()
	org2 := organizerID
	fs.meetings[id] = model.Meeting{
		ID: id, Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt: time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence: "once", Status: "scheduled", GoogleEventID: "evt-seed",
		OrganizerUserID: &org2,
	}
	return id, org
}

func TestUpdateMeeting_HappyPath_ByOrganizer(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	out, err := c.UpdateMeeting(context.Background(), uuid.New(), organizer, id, UpdateInput{Dept: strPtr("Sales")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if out.Dept != "Sales" {
		t.Fatalf("want Dept=Sales, got %q", out.Dept)
	}
	if len(fcs.updated) != 1 || fcs.updated[0] != "evt-seed" {
		t.Fatalf("calendar UpdateEvent not called for seeded event: %+v", fcs.updated)
	}
	if len(fs.updated) != 1 {
		t.Fatalf("Store.UpdateMeeting not called: %d", len(fs.updated))
	}
	if len(fq.updatedEnq) != 1 {
		t.Fatalf("updated job not enqueued: %d", len(fq.updatedEnq))
	}
}

func TestUpdateMeeting_PermissionDenied(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	stranger := uuid.New()
	_, err := c.UpdateMeeting(context.Background(), uuid.New(), stranger, id, UpdateInput{Dept: strPtr("Sales")})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(fcs.updated) != 0 || len(fs.updated) != 0 || len(fq.updatedEnq) != 0 {
		t.Fatalf("no mutation should occur on permission denial")
	}
}

func TestCancelMeeting_HappyPath_ByOrganizer(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	if err := c.CancelMeeting(context.Background(), uuid.New(), organizer, id); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(fcs.deleted) != 1 || fcs.deleted[0] != "evt-seed" {
		t.Fatalf("calendar DeleteEvent not called: %+v", fcs.deleted)
	}
	if len(fs.cancelled) != 1 {
		t.Fatalf("Store.CancelMeeting not called: %d", len(fs.cancelled))
	}
	if len(fq.cancelledEnq) != 1 {
		t.Fatalf("cancelled job not enqueued: %d", len(fq.cancelledEnq))
	}
}

func TestCancelMeeting_PermissionDenied(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	if err := c.CancelMeeting(context.Background(), uuid.New(), uuid.New(), id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(fs.cancelled) != 0 || len(fq.cancelledEnq) != 0 {
		t.Fatalf("no mutation should occur on permission denial")
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/command/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/application/command/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/application/command/meetings_update_cancel_test.go
git commit -m "$(cat <<'EOF'
test(command): UpdateMeeting/CancelMeeting incl. permission enforcement

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Pure-helper tests

**Files:**
- Create: `apps/backend/internal/application/command/meetings_helpers_test.go`

- [ ] **Step 1: Write helper tests**

Create `apps/backend/internal/application/command/meetings_helpers_test.go`:

```go
package command

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func curMeeting() model.Meeting {
	return model.Meeting{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt: time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence: "once",
	}
}

func TestApplyMeetingUpdate_PartialField(t *testing.T) {
	loc := time.UTC
	out, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{Dept: strPtr("Sales")}, loc)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if out.Dept != "Sales" || out.Type != "Sync" || out.Host != "Mia" {
		t.Fatalf("partial update wrong: %+v", out)
	}
	if out.Name == "" || out.Name == curMeeting().Name {
		t.Fatalf("name should be recomputed, got %q", out.Name)
	}
}

func TestApplyMeetingUpdate_DateMustComeTogether(t *testing.T) {
	_, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{Date: strPtr("2026-06-02")}, time.UTC)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput when only Date supplied, got %v", err)
	}
}

func TestApplyMeetingUpdate_AllDateApplied(t *testing.T) {
	out, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{
		Date: strPtr("2026-06-02"), Start: strPtr("09:00"), End: strPtr("09:45"),
	}, time.UTC)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !out.StartsAt.Equal(time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("start not applied: %v", out.StartsAt)
	}
	if !out.EndsAt.Equal(time.Date(2026, 6, 2, 9, 45, 0, 0, time.UTC)) {
		t.Fatalf("end not applied: %v", out.EndsAt)
	}
}

func TestApplyMeetingUpdate_EndBeforeStart(t *testing.T) {
	_, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{
		Date: strPtr("2026-06-02"), Start: strPtr("10:00"), End: strPtr("09:00"),
	}, time.UTC)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for end<=start, got %v", err)
	}
}

func TestOwnerOrOrganizer(t *testing.T) {
	owner := uuid.New()
	organizer := uuid.New()
	stranger := uuid.New()
	org := model.Organization{OwnerUserID: &owner}

	if !ownerOrOrganizer(org, &organizer, owner) {
		t.Fatal("owner should be allowed")
	}
	if !ownerOrOrganizer(org, &organizer, organizer) {
		t.Fatal("organizer should be allowed")
	}
	if ownerOrOrganizer(org, &organizer, stranger) {
		t.Fatal("stranger should be denied")
	}
	if ownerOrOrganizer(model.Organization{}, nil, stranger) {
		t.Fatal("nil owner + nil organizer should deny")
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/command/ -run 'ApplyMeetingUpdate|OwnerOrOrganizer' -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/application/command/...` — `0 issues.`

If `TestApplyMeetingUpdate_PartialField`'s "name recomputed" assertion fails because `curMeeting().Name` is empty (so recomputed != "" but the inequality check is trivially true), that is fine — the assertion still confirms a non-empty recomputed name. Do not loosen it.

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/application/command/meetings_helpers_test.go
git commit -m "$(cat <<'EOF'
test(command): ApplyMeetingUpdate + ownerOrOrganizer

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Query tests

**Files:**
- Create: `apps/backend/internal/application/query/meetings_test.go`

White-box `package query` so the fake can be asserted against the unexported `meetingStore`/`meetingListApp` interfaces.

- [ ] **Step 1: Write query tests**

Create `apps/backend/internal/application/query/meetings_test.go`:

```go
package query

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var (
	_ meetingListApp = (*fakeListApp)(nil)
	_ meetingStore   = (*fakeReadStore)(nil)
)

type fakeListApp struct {
	meetings []model.Meeting
}

func (f *fakeListApp) EmployeeSchedule(_ context.Context, _ string, _, _ time.Time) ([]model.Meeting, error) {
	return f.meetings, nil
}

func TestSchedule_Delegates(t *testing.T) {
	want := []model.Meeting{{Dept: "Eng"}, {Dept: "Sales"}}
	m := NewMeetings(&fakeListApp{meetings: want})
	got, err := m.Schedule(context.Background(), "a@x.io", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if len(got) != 2 || got[0].Dept != "Eng" || got[1].Dept != "Sales" {
		t.Fatalf("delegation wrong: %+v", got)
	}
}

type fakeReadStore struct {
	user  model.User
	parts []model.MeetingParticipant
}

func (f *fakeReadStore) GetUserByID(_ context.Context, _ uuid.UUID) (model.User, error) {
	return f.user, nil
}

func (f *fakeReadStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]model.MeetingParticipant, error) {
	return f.parts, nil
}

func TestMeetingDTO_FormatsInLocation(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Almaty") // UTC+5
	if err != nil {
		t.Fatalf("load loc: %v", err)
	}
	organizer := uuid.New()
	store := &fakeReadStore{
		user:  model.User{Email: "org@x.io"},
		parts: []model.MeetingParticipant{{Email: "a@x.io"}, {Email: ""}, {Email: "b@x.io"}},
	}
	m := model.Meeting{
		ID: uuid.New(), Type: "Sync", Dept: "Eng", Host: "Mia",
		StartsAt: time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC), // 10:00 Almaty
		EndsAt:   time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence: "once", Status: "scheduled",
		OrganizerUserID: &organizer,
	}
	dto := MeetingDTO(context.Background(), store, m, loc)
	if dto.Date != "2026-06-01" || dto.Start != "10:00" || dto.End != "10:30" {
		t.Fatalf("local formatting wrong: %+v", dto)
	}
	if dto.Organizer != "org@x.io" {
		t.Fatalf("organizer email not resolved: %q", dto.Organizer)
	}
	if len(dto.Participants) != 2 || dto.Participants[0] != "a@x.io" || dto.Participants[1] != "b@x.io" {
		t.Fatalf("participants wrong (empty email should be dropped): %+v", dto.Participants)
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/query/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/application/query/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/application/query/meetings_test.go
git commit -m "$(cat <<'EOF'
test(query): Schedule delegation + MeetingDTO timezone/organizer/participants

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Full verification

**Files:** none.

- [ ] **Step 1: Application suite + module suite**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/...` — all PASS.
Run: `cd apps/backend && env -u GOROOT go test ./...` — module-wide green (incl. WS2a postgres tests, Docker permitting).

- [ ] **Step 2: Lint gate**

Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...` — `0 issues.`

- [ ] **Step 3: Clean tree**

Run: `git status --short` — only the human's pre-existing items, if any; no stray files.

- [ ] **Step 4: Post-push CI observation (informational)**

After the human pushes, confirm the CI run is green (`gh run watch`). These tests are DB-free, so no testcontainers dependency in this batch.

---

## Notes on execution order
Task 1 creates the fakes (in `package command`) that Tasks 2 and 3 reuse, so it must run first. Task 4 (`package query`) is independent of Tasks 1–3 but trivially small. Task 5 verifies the whole.
