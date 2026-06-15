# Recurring Series Implementation Plan (Phase 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let organizers change a recurring series' end date after creation (extend → append new occurrences + Google events; trim → cancel/delete future ones), and group a series' occurrences in both meeting lists. (Per-occurrence "skip" already works via cancel `scope=this`.)

**Architecture:** A pure planner (`planSeriesReshape`) decides which spans to create and which occurrences to cancel given the current occurrences, the regenerated cadence, and the new end date. The application `ChangeSeriesEnd` orchestrates: load + authorize, regenerate the cadence via the existing `meeting.Occurrences`, run the planner, create Google events + rows for the appended tail (with rollback) or delete events + cancel rows for the trimmed tail, then set `recurrence_until` on all scheduled rows. New `PATCH .../series-end` endpoints on web + mini-app. The mini-app meeting DTO gains `series_id` + `recurrence_until`. Frontends: a "series ends" control in each edit dialog (series only), and a pure group-by-series helper driving collapsible series rows in both lists.

**Tech Stack:** Go (domain `meeting.Occurrences`, `Repository` port, postgres, Fiber handlers), OpenAPI → `@leadcat/api-client`, React Router admin + mini-app, TanStack Query.

**Critical rule (reshape semantics):** when **extending**, only append occurrences whose start is **after the latest existing scheduled occurrence** — never recreate occurrences the user previously skipped (cancelled). When **trimming**, only cancel scheduled occurrences whose start is after the new end date.

**Prerequisite:** Branch `feat/mini-app-meeting-parity`, on top of Phase 2 (HEAD `4a7a9b3`). Run Go from `apps/backend` with `env -u GOROOT go ...`. Stage explicit paths only; never `git add -A`; never `.gitignore`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Ignore IDE diagnostics (stale LSP) — trust `go`/`pnpm` output. Frontend style: no semicolons, double quotes, 2-space; format touched files with each app's `config/prettier.config.mjs` (never the app-wide `pnpm format`, never config-less `npx prettier`).

---

## File Structure

**Backend**
- Modify `internal/application/repository.go` — add `SetSeriesRecurrenceUntil` to the port.
- Modify `internal/application/repository_unimpl_test.go` — stub it.
- Modify `internal/infrastructure/persistence/postgres/meeting_repo.go` — implement it.
- Create `internal/application/series_reshape.go` — pure `planSeriesReshape` + `SeriesReshape` type.
- Create `internal/application/series_reshape_test.go` — planner table tests.
- Modify `internal/application/series_edit.go` — add `ChangeSeriesEnd`.
- Create `internal/application/series_end_test.go` — orchestration test with fakes.
- Modify `internal/delivery/http/handlers/web_meetings.go` — `WebChangeSeriesEnd`.
- Modify `internal/delivery/http/handlers/miniapp_write.go` — `MiniAppChangeSeriesEnd`.
- Modify `internal/delivery/http/app.go` — two routes.
- Modify `internal/delivery/http/handlers/miniapp_read.go` (+ `internal/application/query/*`) — add `series_id`/`recurrence_until` to the mini-app DTO.
- Modify `apps/backend/openapi/openapi.json` — two endpoints + MiniAppMeeting fields.

**Frontend**
- Admin: `entities/meeting/api.ts`, `entities/meeting/mutations.ts`, `features/meetings/components/meeting-edit-dialog.tsx`, a new `features/meetings/lib/group-series.ts` + test-free pure helper, `features/meetings/components/meetings-table.tsx`.
- Mini-app: `entities/meeting/api.ts`, `entities/meeting/mutations.ts`, `features/meetings/components/meeting-edit-dialog.tsx`, `features/meetings/lib/group-series.ts`, `features/meetings/pages/meetings-list-page.tsx`.

---

## Task 1: Backend — pure reshape planner + repo `SetSeriesRecurrenceUntil`

**Files:**
- Create: `apps/backend/internal/application/series_reshape.go`
- Test: `apps/backend/internal/application/series_reshape_test.go`
- Modify: `apps/backend/internal/application/repository.go`, `apps/backend/internal/application/repository_unimpl_test.go`, `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Write the failing planner test** — `apps/backend/internal/application/series_reshape_test.go`:

```go
package application

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

func occ(id uuid.UUID, start time.Time) model.Meeting {
	return model.Meeting{ID: id, StartsAt: start, EndsAt: start.Add(time.Hour)}
}

func TestPlanSeriesReshape_Extend(t *testing.T) {
	loc := time.UTC
	a, b := uuid.New(), uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 2, 9, 0, 0, 0, loc)
	occs := []model.Meeting{occ(a, d1), occ(b, d2)}
	// candidate cadence regenerated to a later until: 4 days
	candidate := []meeting.Span{
		{Start: d1, End: d1.Add(time.Hour)},
		{Start: d2, End: d2.Add(time.Hour)},
		{Start: d2.AddDate(0, 0, 1), End: d2.AddDate(0, 0, 1).Add(time.Hour)},
		{Start: d2.AddDate(0, 0, 2), End: d2.AddDate(0, 0, 2).Add(time.Hour)},
	}
	newUntil := time.Date(2026, 6, 4, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, candidate, newUntil, loc)
	if len(r.Create) != 2 {
		t.Fatalf("create = %d, want 2", len(r.Create))
	}
	if len(r.CancelIDs) != 0 {
		t.Fatalf("cancel = %d, want 0", len(r.CancelIDs))
	}
}

func TestPlanSeriesReshape_ExtendSkipsGaps(t *testing.T) {
	loc := time.UTC
	a := uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	// only one existing occurrence (the middle ones were skipped/cancelled, so not in occs)
	occs := []model.Meeting{occ(a, d1)}
	candidate := []meeting.Span{
		{Start: d1, End: d1.Add(time.Hour)},
		{Start: d1.AddDate(0, 0, 1), End: d1.AddDate(0, 0, 1).Add(time.Hour)},
		{Start: d1.AddDate(0, 0, 2), End: d1.AddDate(0, 0, 2).Add(time.Hour)},
	}
	newUntil := time.Date(2026, 6, 3, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, candidate, newUntil, loc)
	// latest existing is d1; only spans strictly after d1 are appended (2), gap not resurrected
	if len(r.Create) != 2 || len(r.CancelIDs) != 0 {
		t.Fatalf("create=%d cancel=%d, want 2/0", len(r.Create), len(r.CancelIDs))
	}
}

func TestPlanSeriesReshape_Trim(t *testing.T) {
	loc := time.UTC
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 2, 9, 0, 0, 0, loc)
	d3 := time.Date(2026, 6, 3, 9, 0, 0, 0, loc)
	occs := []model.Meeting{occ(a, d1), occ(b, d2), occ(c, d3)}
	candidate := []meeting.Span{{Start: d1, End: d1.Add(time.Hour)}}
	newUntil := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, candidate, newUntil, loc)
	if len(r.Create) != 0 {
		t.Fatalf("create = %d, want 0", len(r.Create))
	}
	if len(r.CancelIDs) != 2 {
		t.Fatalf("cancel = %d, want 2 (d2,d3)", len(r.CancelIDs))
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (`undefined: planSeriesReshape`):
`cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestPlanSeriesReshape`

- [ ] **Step 3: Implement the planner** — `apps/backend/internal/application/series_reshape.go`:

```go
package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

// SeriesReshape is the plan for changing a series' end date: spans to append and
// occurrence IDs to cancel.
type SeriesReshape struct {
	Create    []meeting.Span
	CancelIDs []uuid.UUID
}

// planSeriesReshape decides the delta for moving a series' end to newUntil.
// occs are the current scheduled occurrences (any order); candidate is the full
// on-cadence span list regenerated up to newUntil. Extend appends only spans
// after the latest existing occurrence (never resurrecting skipped gaps); trim
// cancels scheduled occurrences that start after the newUntil day.
func planSeriesReshape(occs []model.Meeting, candidate []meeting.Span, newUntil time.Time, loc *time.Location) SeriesReshape {
	var latest time.Time
	for _, o := range occs {
		if o.StartsAt.After(latest) {
			latest = o.StartsAt
		}
	}
	cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	var out SeriesReshape
	for _, sp := range candidate {
		if sp.Start.After(latest) {
			out.Create = append(out.Create, sp)
		}
	}
	for _, o := range occs {
		if !o.StartsAt.Before(cutoff) {
			out.CancelIDs = append(out.CancelIDs, o.ID)
		}
	}
	return out
}
```

- [ ] **Step 4: Run, expect PASS:** `cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestPlanSeriesReshape`

- [ ] **Step 5: Add `SetSeriesRecurrenceUntil` to the port + stub + postgres impl.**

In `internal/application/repository.go`, after `CancelAllSeriesOccurrences(...)`, add:
```go
	SetSeriesRecurrenceUntil(ctx context.Context, organizationID, seriesID uuid.UUID, until time.Time) error
```
In `internal/application/repository_unimpl_test.go`, add the matching stub on `unimplementedRepo` (match the file's existing style — `return nil`/`panic`):
```go
func (unimplementedRepo) SetSeriesRecurrenceUntil(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return errUnimplemented
}
```
In `internal/infrastructure/persistence/postgres/meeting_repo.go`, add:
```go
// SetSeriesRecurrenceUntil updates recurrence_until on all scheduled occurrences of a series.
func (s *Store) SetSeriesRecurrenceUntil(ctx context.Context, organizationID, seriesID uuid.UUID, until time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET recurrence_until = $3, updated_at = now()
		WHERE series_id = $1 AND organization_id = $2 AND status = 'scheduled'`,
		seriesID, organizationID, until)
	return err
}
```
(Confirm the stub's error sentinel name by reading `repository_unimpl_test.go`; if stubs `panic("unimplemented")`, do that instead.)

- [ ] **Step 6: Build + planner test green:**
`cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/application/ -run TestPlanSeriesReshape`

- [ ] **Step 7: Commit**
```bash
git add apps/backend/internal/application/series_reshape.go \
  apps/backend/internal/application/series_reshape_test.go \
  apps/backend/internal/application/repository.go \
  apps/backend/internal/application/repository_unimpl_test.go \
  apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go
git commit -m "feat(meetings): series reshape planner + SetSeriesRecurrenceUntil repo method

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Backend — application `ChangeSeriesEnd`

**Files:**
- Modify: `apps/backend/internal/application/series_edit.go`
- Test: `apps/backend/internal/application/series_end_test.go`

- [ ] **Step 1: Write the failing test** — `apps/backend/internal/application/series_end_test.go`. Use a fake repo embedding `unimplementedRepo` and a fake calendar. Assert: not-a-series → `ErrInvalidInput`; extend creates rows + sets until.

```go
package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type reshapeRepo struct {
	unimplementedRepo
	org      model.Organization
	picked   model.Meeting
	occs     []model.Meeting
	created  []model.Meeting
	setUntil time.Time
	setCalls int
}

func (r *reshapeRepo) GetMeeting(context.Context, uuid.UUID, uuid.UUID) (model.Meeting, error) {
	return r.picked, nil
}
func (r *reshapeRepo) GetOrganization(context.Context, uuid.UUID) (model.Organization, error) {
	return r.org, nil
}
func (r *reshapeRepo) ListSeriesAllOccurrences(context.Context, uuid.UUID, uuid.UUID) ([]model.Meeting, error) {
	return r.occs, nil
}
func (r *reshapeRepo) ListParticipants(context.Context, uuid.UUID) ([]model.MeetingParticipant, error) {
	return nil, nil
}
func (r *reshapeRepo) CreateMeetingSeries(_ context.Context, ms []model.Meeting, _ []model.MeetingParticipant) ([]model.Meeting, error) {
	r.created = append(r.created, ms...)
	return ms, nil
}
func (r *reshapeRepo) SetSeriesRecurrenceUntil(_ context.Context, _ uuid.UUID, _ uuid.UUID, until time.Time) error {
	r.setUntil = until
	r.setCalls++
	return nil
}

func TestChangeSeriesEnd_NotASeries(t *testing.T) {
	owner := uuid.New()
	repo := &reshapeRepo{
		org:    model.Organization{OwnerUserID: &owner, TZ: "UTC"},
		picked: model.Meeting{ID: uuid.New(), Recurrence: "once"},
	}
	s := &Services{Store: repo, Calendar: stubProvider{}}
	if _, _, err := s.ChangeSeriesEnd(context.Background(), uuid.New(), owner, repo.picked.ID, "2026-07-01"); err == nil {
		t.Fatal("expected error for non-series")
	}
}

func TestChangeSeriesEnd_Extend(t *testing.T) {
	owner := uuid.New()
	seriesID := uuid.New()
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	until := start.AddDate(0, 0, 1)
	anchor := model.Meeting{
		ID: uuid.New(), OrganizerUserID: &owner, SeriesID: &seriesID,
		Dept: "Eng", Type: "Sync", Host: "h", Recurrence: "daily",
		StartsAt: start, EndsAt: start.Add(time.Hour), RecurrenceUntil: &until,
	}
	repo := &reshapeRepo{
		org:    model.Organization{OwnerUserID: &owner, TZ: "UTC"},
		picked: anchor, occs: []model.Meeting{anchor},
	}
	s := &Services{Store: repo, Calendar: stubProvider{}}
	added, removed, err := s.ChangeSeriesEnd(context.Background(), uuid.New(), owner, anchor.ID, "2026-06-03")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if added != 2 || removed != 0 {
		t.Fatalf("added=%d removed=%d, want 2/0", added, removed)
	}
	if repo.setCalls == 0 {
		t.Fatal("expected SetSeriesRecurrenceUntil to be called")
	}
}
```

Add a tiny calendar stub in this test file (reuse if one already exists in the package — search for an existing `stubProvider`/calendar fake first and use that instead):

```go
type stubProvider struct{}

func (stubProvider) For(context.Context, uuid.UUID) (CalendarService, error) {
	return stubCal{}, nil
}

type stubCal struct{}

func (stubCal) CreateEvent(context.Context, CalendarEvent) (CalendarResult, error) {
	return CalendarResult{EventID: "evt", MeetLink: "https://meet"}, nil
}
func (stubCal) UpdateEvent(context.Context, string, CalendarEvent) error        { return nil }
func (stubCal) UpdateAttendees(context.Context, string, []string) error         { return nil }
func (stubCal) DeleteEvent(context.Context, string) error                       { return nil }
```

- [ ] **Step 2: Run, expect FAIL** (`s.ChangeSeriesEnd undefined`). If `stubProvider`/`stubCal` collide with existing test helpers, rename yours.
`cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestChangeSeriesEnd`

- [ ] **Step 3: Implement `ChangeSeriesEnd`** in `internal/application/series_edit.go` (append). Reuse `ownerOrOrganizer`, `orDefault`, `s.deleteEventsBestEffort`, `s.enqueue... ` already present in the package.

```go
// ChangeSeriesEnd moves a series' recurrence_until. Extending appends new
// occurrences (with Google events) after the latest existing one; trimming
// cancels and deletes occurrences after the new end. Returns counts added/removed.
func (s *Services) ChangeSeriesEnd(ctx context.Context, organizationID, userID, meetingID uuid.UUID, untilStr string) (int, int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return 0, 0, fmt.Errorf("bad timezone: %w", err)
	}
	newUntil, err := time.ParseInLocation("2006-01-02", untilStr, loc)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: bad until date", ErrInvalidInput)
	}
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, 0, err
	}
	if len(occs) == 0 {
		return 0, 0, nil
	}
	anchor := occs[0] // ordered by starts_at ascending
	rec := meeting.Recurrence(anchor.Recurrence)
	if rec == meeting.Once {
		return 0, 0, fmt.Errorf("%w: not a recurring series", ErrInvalidInput)
	}
	anchorStart := anchor.StartsAt.In(loc)
	anchorEnd := anchor.EndsAt.In(loc)
	candidate, err := meeting.Occurrences(anchorStart, anchorEnd, rec, anchor.RecurrenceDays, newUntil)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	plan := planSeriesReshape(occs, candidate, newUntil, loc)

	added := 0
	if len(plan.Create) > 0 {
		calSvc, ferr := s.Calendar.For(ctx, organizationID)
		if ferr != nil {
			return 0, 0, ferr
		}
		parts, perr := s.Store.ListParticipants(ctx, anchor.ID)
		if perr != nil {
			return 0, 0, perr
		}
		var emails []string
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
		var createdIDs []string
		rows := make([]model.Meeting, 0, len(plan.Create))
		for _, sp := range plan.Create {
			name := meeting.GenerateName(anchor.Dept, anchor.Type, anchor.Host, sp.Start, rec)
			res, cerr := calSvc.CreateEvent(ctx, CalendarEvent{
				Title: name, Description: anchor.Description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
			})
			if cerr != nil {
				s.deleteEventsBestEffort(ctx, calSvc, createdIDs)
				return 0, 0, fmt.Errorf("calendar: %w", cerr)
			}
			createdIDs = append(createdIDs, res.EventID)
			until := newUntil
			rows = append(rows, model.Meeting{
				OrganizationID: organizationID, OrganizerUserID: anchor.OrganizerUserID,
				Dept: anchor.Dept, Type: anchor.Type, Host: anchor.Host,
				StartsAt: sp.Start.UTC(), EndsAt: sp.End.UTC(),
				Recurrence: string(rec), RecurrenceDays: anchor.RecurrenceDays,
				Name: name, Description: anchor.Description,
				GoogleEventID: res.EventID, MeetLink: res.MeetLink,
				SeriesID: picked.SeriesID, RecurrenceUntil: &until,
			})
		}
		if _, serr := s.Store.CreateMeetingSeries(ctx, rows, parts); serr != nil {
			s.deleteEventsBestEffort(ctx, calSvc, createdIDs)
			return 0, 0, serr
		}
		added = len(rows)
	}

	removed := 0
	if len(plan.CancelIDs) > 0 {
		cancelSet := make(map[uuid.UUID]bool, len(plan.CancelIDs))
		for _, id := range plan.CancelIDs {
			cancelSet[id] = true
		}
		cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
		if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr == nil {
			var ids []string
			for _, o := range occs {
				if cancelSet[o.ID] && o.GoogleEventID != "" {
					ids = append(ids, o.GoogleEventID)
				}
			}
			s.deleteEventsBestEffort(ctx, calSvc, ids)
		}
		n, cerr := s.Store.CancelSeriesOccurrences(ctx, organizationID, *picked.SeriesID, cutoff)
		if cerr != nil {
			return added, 0, cerr
		}
		removed = n
	}

	if err := s.Store.SetSeriesRecurrenceUntil(ctx, organizationID, *picked.SeriesID, newUntil); err != nil {
		return added, removed, err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue_series_end_changed",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return added, removed, nil
}
```

Confirm the file already imports `fmt`, `time`, `zap`, `uuid`, the `meeting` domain pkg, and `model` (the sibling series funcs use them). Add any missing.

- [ ] **Step 4: Run, expect PASS:** `cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestChangeSeriesEnd`
- [ ] **Step 5: Full build + application tests:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/application/...`
- [ ] **Step 6: Commit**
```bash
git add apps/backend/internal/application/series_edit.go apps/backend/internal/application/series_end_test.go
git commit -m "feat(meetings): ChangeSeriesEnd — extend/trim a series end date

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Backend — series-end HTTP handlers + routes

**Files:** `internal/delivery/http/handlers/web_meetings.go`, `internal/delivery/http/handlers/miniapp_write.go`, `internal/delivery/http/app.go`

- [ ] **Step 1: Web handler.** In `web_meetings.go` add (the file already has `webUser`, `mapMeetingWriteError`, `a.Log`, `a.App`):

```go
type seriesEndRequest struct {
	Until string `json:"until"`
}

func (a *API) WebChangeSeriesEnd(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	meetingID, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_meeting_id")
	}
	var req seriesEndRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	added, removed, err := a.App.ChangeSeriesEnd(c.UserContext(), orgID, user.ID, meetingID, req.Until)
	if err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.Log.Info("web_series_end_changed", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()), zap.Int("added", added), zap.Int("removed", removed))
	return c.JSON(fiber.Map{"meeting": m, "added": added, "removed": removed})
}
```

- [ ] **Step 2: Mini-app handler.** In `miniapp_write.go` add (reuse `botUser`, `a.editableOrganization`, `a.App.EnsureMiniAppOrganizer`, `a.toMeetingDTO`, and the existing error-switch style):

```go
func (a *API) MiniAppChangeSeriesEnd(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	var req struct {
		Until string `json:"until"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	organizationID, found, err := a.editableOrganization(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if _, _, err := a.App.ChangeSeriesEnd(c.Context(), organizationID, organizerID, meetingID, req.Until); err != nil {
		switch {
		case errors.Is(err, application.ErrForbidden):
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		case errors.Is(err, application.ErrInvalidInput):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
	}
	m, err := a.App.GetMeeting(c.Context(), organizationID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}
```

- [ ] **Step 3: Routes.** In `app.go`, add after the existing meetings routes:
  - Web (after `scoped.Delete("/meetings/:mid/participants", ...)`):
    ```go
    scoped.Patch("/meetings/:mid/series-end", api.WebChangeSeriesEnd)
    ```
  - Mini-app (after `miniapp.Delete("/meetings/:id/participants", ...)`):
    ```go
    miniapp.Patch("/meetings/:id/series-end", api.MiniAppChangeSeriesEnd)
    ```

- [ ] **Step 4: Build + test + lint:**
`cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./... && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/delivery/... ./internal/application/...`
Expected: clean; 0 issues on touched packages.

- [ ] **Step 5: Commit**
```bash
git add apps/backend/internal/delivery/http/handlers/web_meetings.go \
  apps/backend/internal/delivery/http/handlers/miniapp_write.go \
  apps/backend/internal/delivery/http/app.go
git commit -m "feat(meetings): series-end PATCH endpoints (web + mini-app)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Backend — expose `series_id` + `recurrence_until` on the mini-app DTO

**Files:** `internal/application/query/meeting_read.go` (the `MiniAppMeeting` struct + `MeetingDTO`/`miniappMeetingFromQuery`), `internal/delivery/http/handlers/miniapp_read.go`

- [ ] **Step 1: Read** `internal/application/query/meeting_read.go` and `internal/delivery/http/handlers/miniapp_read.go` to find the `MiniAppMeeting` DTO struct (fields like `ID,Type,Dept,Host,Date,Start,End,Rec,Organizer,Participants,Desc,MeetLink,Status`) and where it is populated from a `model.Meeting`.

- [ ] **Step 2: Add two fields** to the `MiniAppMeeting` DTO struct:
```go
	SeriesID        string `json:"series_id"`
	RecurrenceUntil string `json:"recurrence_until"`
```
- [ ] **Step 3: Populate them** wherever the DTO is built from a `model.Meeting` (e.g. in `MeetingDTO`): set `SeriesID` to the meeting's `SeriesID.String()` when non-nil else `""`; set `RecurrenceUntil` to the local date `m.RecurrenceUntil.In(loc).Format("2006-01-02")` when non-nil else `""`. Match the existing date-formatting style used for `Date` in that function (it already has the loc). If the DTO is also built in a second place (e.g. `miniappMeetingFromQuery` from a SQL read model), thread the same two fields through that read model too (add to the query struct + SELECT if applicable) — read the code and keep both build paths consistent.

- [ ] **Step 4: Build + test:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`

- [ ] **Step 5: Commit**
```bash
git add apps/backend/internal/application/query/meeting_read.go apps/backend/internal/delivery/http/handlers/miniapp_read.go
git commit -m "feat(meetings): expose series_id + recurrence_until on mini-app meeting DTO

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Backend — OpenAPI + regenerate api-client

**Files:** `apps/backend/openapi/openapi.json`, `packages/api-client/src/generated/schema.ts`

- [ ] **Step 1:** In `openapi.json`, add a `PATCH` operation under a NEW path `"/api/orgs/{id}/meetings/{mid}/series-end"` (compact style, like the participants path added earlier): params `id`,`mid` (uuid path), requestBody `{ "until": string format date }` required, 200 response `{ meeting: Meeting, added: integer, removed: integer }`, plus 400/401/403/404 referencing `ApiErrorResponse`. Use `operationId: orgMeetingSeriesEnd`, tag `meetings`, `security: [{ cookieAuth: [] }]`.
- [ ] **Step 2:** Add the two new fields to the `MiniAppMeeting` schema's `properties` and `required`: `series_id` (string) and `recurrence_until` (string). (Keep existing fields; match the file's compact style.)
- [ ] **Step 3:** Validate + regenerate:
`cd /Users/temirlan/Workspace/in-house/lead-cat && python3 -c "import json; json.load(open('apps/backend/openapi/openapi.json')); print('valid')" && pnpm openapi:generate`
- [ ] **Step 4:** Confirm small additive diff on openapi.json (`git diff --stat apps/backend/openapi/openapi.json`) — not a whole-file reformat (do NOT run prettier on it).
- [ ] **Step 5: Commit**
```bash
git add apps/backend/openapi/openapi.json packages/api-client/src/generated/schema.ts
git commit -m "feat(meetings): openapi series-end endpoint + mini-app DTO fields; regen client

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Admin — series-end control in the edit dialog

**Files:** `apps/admin/app/entities/meeting/api.ts`, `apps/admin/app/entities/meeting/mutations.ts`, `apps/admin/app/features/meetings/components/meeting-edit-dialog.tsx`

- [ ] **Step 1: API.** In `entities/meeting/api.ts` add:
```typescript
export async function changeSeriesEnd(
  orgId: string,
  meetingId: string,
  until: string
): Promise<Meeting> {
  const { data } = await api.patch<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}/series-end`,
    { until }
  )
  return data.meeting
}
```
- [ ] **Step 2: Mutation.** In `entities/meeting/mutations.ts` add a `useChangeSeriesEnd(orgId)` mutation calling `changeSeriesEnd(orgId, meetingId, until)`, invalidating `meetingKeys.list(orgId)` on success (match the existing mutations' structure in that file).
- [ ] **Step 3: UI.** Read `features/meetings/components/meeting-edit-dialog.tsx`. When the meeting `isSeries(meeting)`, render an extra section (below the form / scope controls): a date `Input` prefilled from `meeting.recurrence_until` (slice the date part) plus an "Update end date" button that calls the mutation with the chosen date and toasts success/error. Keep it visually separate from the per-occurrence field edits. If the dialog receives the meeting + orgId already (it does for participants editing), reuse those props.
- [ ] **Step 4: Gates + format.**
`cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`
then `cd /Users/temirlan/Workspace/in-house/lead-cat && npx prettier --write --config apps/admin/config/prettier.config.mjs <the three files>` and re-run `pnpm --filter admin typecheck`.
- [ ] **Step 5: Commit**
```bash
git add apps/admin/app/entities/meeting/api.ts apps/admin/app/entities/meeting/mutations.ts apps/admin/app/features/meetings/components/meeting-edit-dialog.tsx
git commit -m "feat(admin): edit a recurring series' end date

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Mini-app — series-end control in the edit dialog

**Files:** `apps/mini-app/app/entities/meeting/api.ts`, `apps/mini-app/app/entities/meeting/mutations.ts`, `apps/mini-app/app/features/meetings/components/meeting-edit-dialog.tsx`

- [ ] **Step 1: API.** In mini-app `entities/meeting/api.ts` add (match the `apiFetch` style):
```typescript
export async function changeSeriesEnd(id: string, until: string): Promise<Meeting> {
  const res = await apiFetch<{ meeting: Meeting }>(
    `/api/miniapp/meetings/${id}/series-end`,
    { method: "PATCH", body: { until } }
  )
  return res.meeting
}
```
- [ ] **Step 2: Mutation.** Add `useChangeSeriesEnd()` to mini-app `entities/meeting/mutations.ts` calling `changeSeriesEnd(id, until)`, invalidating `meetingKeys.all` on success (match the file's existing mutations).
- [ ] **Step 3: UI.** Read `features/meetings/components/meeting-edit-dialog.tsx`. When `isSeriesMeeting(meeting)`, add a "Series ends" date input (prefilled from `meeting.recurrence_until`, now present on the DTO) + an "Update end date" button calling the mutation; toast on success/error. Lightweight (no new deps), matching the dialog's existing controls.
- [ ] **Step 4: Gates + format.** `cd apps/mini-app && pnpm typecheck && pnpm lint && pnpm build`, then prettier-write the three files with `apps/mini-app/config/prettier.config.mjs`, re-typecheck.
- [ ] **Step 5: Commit**
```bash
git add apps/mini-app/app/entities/meeting/api.ts apps/mini-app/app/entities/meeting/mutations.ts apps/mini-app/app/features/meetings/components/meeting-edit-dialog.tsx
git commit -m "feat(mini-app): edit a recurring series' end date

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Admin — group a series' occurrences in the meetings table

**Files:** Create `apps/admin/app/features/meetings/lib/group-series.ts` (+ `group-series.test.ts` if a frontend test runner exists; otherwise rely on typecheck), modify `apps/admin/app/features/meetings/components/meetings-table.tsx`

- [ ] **Step 1: Pure grouping helper** — `apps/admin/app/features/meetings/lib/group-series.ts`:
```typescript
import type { Meeting } from "~/entities/meeting/types"

export type MeetingGroup =
  | { kind: "single"; meeting: Meeting }
  | { kind: "series"; seriesId: string; meetings: Meeting[] }

export function groupBySeries(meetings: Meeting[]): MeetingGroup[] {
  const order: string[] = []
  const bySeries = new Map<string, Meeting[]>()
  const singles: MeetingGroup[] = []
  for (const m of meetings) {
    const sid = m.series_id ?? ""
    if (!sid) {
      singles.push({ kind: "single", meeting: m })
      continue
    }
    if (!bySeries.has(sid)) {
      bySeries.set(sid, [])
      order.push(sid)
    }
    bySeries.get(sid)!.push(m)
  }
  const grouped: MeetingGroup[] = order.map((sid) => ({
    kind: "series",
    seriesId: sid,
    meetings: bySeries.get(sid)!,
  }))
  return [...grouped, ...singles]
}
```
- [ ] **Step 2: Table grouping UI.** Read `meetings-table.tsx`. Render groups from `groupBySeries(meetings)`: a `series` group shows a header row (meeting name/dept + a "🔁 N occurrences" count using `meetings.length` + the next/earliest upcoming occurrence's date) with a collapse/expand toggle (local `useState` set of expanded seriesIds); expanded → render its occurrence rows (the existing row markup, each still with Edit/Cancel where Cancel scope=this is the per-occurrence skip). `single` groups render exactly as today. Keep the existing columns. Do NOT change the row actions' behavior.
- [ ] **Step 3: Gates + format.** `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`; prettier-write the touched files with the admin config; re-typecheck.
- [ ] **Step 4: Commit**
```bash
git add apps/admin/app/features/meetings/lib/group-series.ts apps/admin/app/features/meetings/components/meetings-table.tsx
git commit -m "feat(admin): group recurring series occurrences in the meetings table

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Mini-app — group a series in the meetings list

**Files:** Create `apps/mini-app/app/features/meetings/lib/group-series.ts`, modify `apps/mini-app/app/features/meetings/pages/meetings-list-page.tsx`

- [ ] **Step 1: Pure grouping helper** — `apps/mini-app/app/features/meetings/lib/group-series.ts` (mirror of admin's, but using the mini-app `Meeting` type and its `series_id` field now present on the DTO):
```typescript
import type { Meeting } from "~/entities/meeting/types"

export type MeetingGroup =
  | { kind: "single"; meeting: Meeting }
  | { kind: "series"; seriesId: string; meetings: Meeting[] }

export function groupBySeries(meetings: Meeting[]): MeetingGroup[] {
  const order: string[] = []
  const bySeries = new Map<string, Meeting[]>()
  const singles: MeetingGroup[] = []
  for (const m of meetings) {
    const sid = m.series_id ?? ""
    if (!sid) {
      singles.push({ kind: "single", meeting: m })
      continue
    }
    if (!bySeries.has(sid)) {
      bySeries.set(sid, [])
      order.push(sid)
    }
    bySeries.get(sid)!.push(m)
  }
  return [
    ...order.map((sid): MeetingGroup => ({ kind: "series", seriesId: sid, meetings: bySeries.get(sid)! })),
    ...singles,
  ]
}
```
- [ ] **Step 2: List grouping UI.** Read `meetings-list-page.tsx`. For each tab's meetings, render `groupBySeries(...)`: a `series` group shows one collapsible card (title + "🔁 N" + next occurrence) that expands to the individual `MeetingCard`s; `single` groups render the existing `MeetingCard`. Keep navigation/detail behavior unchanged. Local `useState` for expanded series.
- [ ] **Step 3: Gates + format.** `cd apps/mini-app && pnpm typecheck && pnpm lint && pnpm build`; prettier-write touched files with the mini-app config; re-typecheck.
- [ ] **Step 4: Commit**
```bash
git add apps/mini-app/app/features/meetings/lib/group-series.ts apps/mini-app/app/features/meetings/pages/meetings-list-page.tsx
git commit -m "feat(mini-app): group recurring series occurrences in the meetings list

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 10: Final verification

- [ ] **Step 1: Backend gate:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
- [ ] **Step 2: Admin gate:** `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`
- [ ] **Step 3: Mini-app gate:** `cd apps/mini-app && pnpm typecheck && pnpm lint && pnpm build`
- [ ] **Step 4: Coverage check:** grep that `WebChangeSeriesEnd`, `MiniAppChangeSeriesEnd`, `series-end` routes, `ChangeSeriesEnd`, `planSeriesReshape`, `SetSeriesRecurrenceUntil`, and `groupBySeries` (both apps) all exist.
- [ ] **Step 5: Clean tree:** `git status --short` shows no unexpected files.

---

## Notes & decisions

- **Reshape template = earliest scheduled occurrence** (anchor): supplies recurrence/days/time-of-day/duration/dept/type/host/desc/organizer/participants. Regeneration via `meeting.Occurrences(anchorStart, anchorEnd, …, newUntil)` reproduces the on-cadence sequence; only spans **after the latest existing occurrence** are appended (skipped gaps are never resurrected). Trim cancels scheduled occurrences starting after the new end day.
- **Extend creates Google events with rollback** (delete created events if the row insert fails), mirroring `CreateMeeting`'s series path. Trim deletes events best-effort.
- **`recurrence_until` is set on all scheduled rows** after either operation, keeping series metadata consistent.
- **Mini-app DTO gains `series_id`/`recurrence_until`** — required for both grouping and prefilling the end-date control (admin's `WebMeeting` already has them).
- **Grouping is display-only**; per-occurrence Edit + Cancel(scope=this = skip) are unchanged. Series order preserved by first-seen; singles after series.
- **Date handling stays in the org timezone** for v1 (consistent with the rest of the meetings code); revisit when per-user timezone (Phase 4) lands.
</content>
