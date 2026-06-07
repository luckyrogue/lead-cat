# Slice B — Recurrence (series everywhere) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire recurring Google Meet meetings end-to-end through the TMA — create with end date + custom-weekday support, scope-aware edit/cancel (`this | whole`), full-series conflict warnings.

**Architecture:** Backend first (domain → persistence → application → HTTP → OpenAPI), then frontend types/mutations, wizard until-picker, grouped conflicts, dual-scope meeting-detail actions, docs/verify. New backend application commands `UpdateWholeSeries`, `CancelWholeSeries`, `MeetingSeriesConflicts` reuse existing per-occurrence primitives. Existing `UpdateSeries`/`CancelSeries` (from-forward) remain as internal helpers, not exposed via TMA HTTP.

**Tech Stack:** Go 1.x (Fiber, pgx, asynq, zap), Postgres 16, React 19 (Vite, TanStack Router/Query, axios), Telegram Mini App.

**Spec:** `docs/superpowers/specs/2026-06-07-slice-b-recurrence-design.md`.

**Branch:** `feat/meetings-recurrence-b` from current `main` (= `47ced23` plus the slice-B spec commit `537db79` that needs to ff into main first; see B-T0).

---

## File structure

| Path | Action | Responsibility |
|------|--------|----------------|
| `backend/internal/domain/meeting/meeting.go` | Modify | Add `Custom`, drop `Biweekly`, add `Input.RecurrenceDays`. |
| `backend/internal/domain/meeting/recurrence.go` | Modify | `Occurrences` accepts `days []int`; custom weekday filter. |
| `backend/internal/domain/meeting/recurrence_test.go` | Modify | TDD for Custom kind. |
| `backend/internal/domain/meeting/validate.go` | Modify | Custom requires non-empty `RecurrenceDays`. |
| `backend/internal/domain/meeting/validate_test.go` | Modify | Drop Biweekly cases; add Custom cases. |
| `backend/migrations/20260607120000_meeting_recurrence_days.sql` | Create | `ALTER TABLE meetings ADD COLUMN recurrence_days JSONB`. |
| `backend/internal/infrastructure/persistence/postgres/models.go` | Modify | `Meeting.RecurrenceDays []int`. |
| `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` | Modify | Scan/write `recurrence_days`; add `ListSeriesAllOccurrences`, `CancelAllSeriesOccurrences`. |
| `backend/internal/application/conflict.go` | Modify | `CreateMeetingInput.RecurrenceDays`, pass through to `Occurrences`. |
| `backend/internal/application/meeting_service.go` | Modify | Persist `RecurrenceDays` on series rows. |
| `backend/internal/application/series_edit.go` | Modify | Add `UpdateWholeSeries`, `CancelWholeSeries`. |
| `backend/internal/application/series_conflicts.go` | Create | `MeetingSeriesConflicts(ctx, emails, span, r, days, until)`. |
| `backend/internal/application/series_conflicts_test.go` | Create | Pure expand test. |
| `backend/internal/delivery/http/handlers/tma_write.go` | Modify | Recurrence params on create; `scope=` on patch/delete; series conflicts on `/tma/conflicts`. |
| `backend/openapi/openapi.json` | Modify | New params + occurrence-grouped conflicts schema. |
| `backend/docs/openapi.json` | Modify | Byte-identical mirror. |
| `frontend/src/shared/api/generated/schema.ts` | Regen | From updated openapi.json. |
| `frontend/src/features/meetings/api.ts` | Modify | `OccurrenceConflicts`, `recurrence_until`/`recurrence_days`/`scope` plumbing. |
| `frontend/src/features/meetings/queries.ts` | Modify | `useUpdateMeeting`/`useDeleteMeeting` grow `scope`; `useConflicts` returns groups. |
| `frontend/src/entities/meeting/types.ts` | Modify | `Meeting.seriesId?`. |
| `frontend/src/entities/meeting/lib/format.ts` | Modify | `detailToDraft`/`draftToMeeting` preserve `seriesId`. |
| `frontend/src/features/meeting-create/lib/use-create-wizard.ts` | Modify | Add `until`, smart defaults, drop `recurringBlocked`, support `lockedFields`. |
| `frontend/src/features/meeting-create/components/wizard-step-when.tsx` | Modify | Until picker for non-once. |
| `frontend/src/features/meeting-create/components/wizard-step-review.tsx` | Modify | Grouped-by-date conflicts; drop recurringSoon block. |
| `frontend/src/features/meeting-create/components/create-wizard.tsx` | Modify | Pass `lockedFields`, drop `recurringBlocked`. |
| `frontend/src/features/meeting-create/pages/create-page.tsx` | Modify | Pass `recurrence_until`/`recurrence_days`; read `scope` from search; pass to mutations. |
| `frontend/src/shared/tma/i18n.ts` | Modify | Add until + series-edit + grouped-conflicts keys (ru/kk/en). |
| `frontend/src/features/meetings/components/meeting-detail.tsx` | Modify | Compute `isSeries` from `seriesId`; pass `onEdit(scope)` / `onDelete(scope)`. |
| `frontend/src/features/meetings/components/meeting-detail-actions.tsx` | Modify | Dual-scope buttons or sheet. |
| `frontend/src/features/meetings/pages/meetings-list-page.tsx` | Modify | Wire `handleDelete(scope)`; `useDeleteMeeting` takes `{ id, scope }`. |
| `frontend/src/routes/_tma/meetings.create.$editId.tsx` (or `.create.$editId.tsx`) | Modify | Search-param schema includes `scope?: "this" \| "whole"`. |
| `docs/MEETINGS.md` | Modify | Status note + write-path table updates (recurring done). |
| `docs/API.md` | Modify | Document `scope`, recurrence body fields, occurrence-grouped conflicts shape. |

---

## Pre-flight gate (do this BEFORE any task)

The slice-B spec commit `537db79` sits on `feat/meetings-tma-write-paths-a`, one commit ahead of `main`. Confirm with the user that ff-merging the spec to `main` is allowed, then:

```bash
git push . feat/meetings-tma-write-paths-a:main
git push origin main
```

Verify: `git log --oneline -1 main` shows `537db79`. THEN create the slice-B branch in B-T0.

---

## Task B-T0: Create slice-B branch

**Files:**
- No code changes.

- [ ] **Step 1: Verify main is at the spec commit**

Run:
```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat log --oneline -1 main
```
Expected: `537db79 docs(spec): slice B — recurrence (series everywhere)`.

- [ ] **Step 2: Create and switch to the slice-B branch**

Run:
```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat checkout main
git -C /Users/temirlan/Workspace/in-house/lead-cat checkout -b feat/meetings-recurrence-b
git -C /Users/temirlan/Workspace/in-house/lead-cat status --short
```
Expected: `On branch feat/meetings-recurrence-b`, working tree status mirrors the existing uncommitted user-edits that have been carried across the session (the docs/frontend `M` files). DO NOT touch those.

- [ ] **Step 3: No commit for T0**

T0 is a branch-only step; nothing to commit.

---

## Task B-T1: Domain — add Custom, drop Biweekly, extend Occurrences (TDD)

**Files:**
- Modify: `backend/internal/domain/meeting/meeting.go`
- Modify: `backend/internal/domain/meeting/recurrence.go`
- Modify: `backend/internal/domain/meeting/recurrence_test.go`
- Modify: `backend/internal/domain/meeting/validate.go`
- Modify: `backend/internal/domain/meeting/validate_test.go`

- [ ] **Step 1: Grep for any Biweekly callers outside the domain**

Run:
```bash
grep -rn "Biweekly\|\"biweekly\"" backend/ --include="*.go"
```
Expected: only the three lines in `domain/meeting/meeting.go` and `recurrence.go`. (Earlier exploration confirmed this — re-verify.)

- [ ] **Step 2: Write failing tests for Custom recurrence**

Replace existing `recurrence_test.go` TestOccurrences cases with additions for Custom. Append to the test file (keep existing daily/weekly/biweekly tests — biweekly will be dropped next; we'll fix those failures in step 4):

```go
func TestOccurrences_Custom(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)  // Mon
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, loc)
	until := time.Date(2026, 6, 21, 0, 0, 0, 0, loc) // Sun (3 weeks)

	// Mon, Wed, Fri => weekdays 1, 3, 5
	got, err := Occurrences(start, end, Custom, []int{1, 3, 5}, until)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("expected 9 occurrences (3 wks × 3 days), got %d", len(got))
	}
	wantDates := []string{
		"2026-06-01", "2026-06-03", "2026-06-05",
		"2026-06-08", "2026-06-10", "2026-06-12",
		"2026-06-15", "2026-06-17", "2026-06-19",
	}
	for i, sp := range got {
		if sp.Start.Format("2006-01-02") != wantDates[i] {
			t.Errorf("occurrence %d: want %s, got %s", i, wantDates[i], sp.Start.Format("2006-01-02"))
		}
	}
}

func TestOccurrences_CustomEmptyDays(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, loc)
	until := time.Date(2026, 6, 7, 0, 0, 0, 0, loc)
	_, err := Occurrences(start, end, Custom, nil, until)
	if !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}
```

The test imports `errors` — ensure the import block lists it.

- [ ] **Step 3: Run tests to verify they fail**

Run:
```bash
cd backend && go test ./internal/domain/meeting/... 2>&1 | tail -20
```
Expected: compilation error (`Custom` undefined, `ErrRecurrenceDays` undefined, `Occurrences` signature wrong) — or runtime fails for existing tests once compilation passes. Either is OK; we're about to fix it.

- [ ] **Step 4: Update meeting.go — add Custom, drop Biweekly, add RecurrenceDays**

Replace the const block and `recurrenceLabels` map:

```go
const (
	Once    Recurrence = "once"
	Daily   Recurrence = "daily"
	Weekly  Recurrence = "weekly"
	Custom  Recurrence = "custom"
	Monthly Recurrence = "monthly"
)

var recurrenceLabels = map[Recurrence]string{
	Once:    "",
	Daily:   "Ежедневно",
	Weekly:  "Еженедельно",
	Custom:  "По выбранным дням",
	Monthly: "Ежемесячно",
}
```

Add `RecurrenceDays []int` to `Input`:

```go
type Input struct {
	Dept           string
	Type           string
	Host           string
	StartsAt       time.Time
	EndsAt         time.Time
	Recurrence     Recurrence
	RecurrenceDays []int // 1=Mon..7=Sun; required when Recurrence == Custom.
	Description    string
}
```

- [ ] **Step 5: Update recurrence.go — extend Occurrences and add ErrRecurrenceDays**

Replace the file body's `Occurrences` and `next` functions:

```go
// ErrRecurrenceDays means a custom-weekday series has no days set.
var ErrRecurrenceDays = errors.New("custom recurrence needs at least one weekday")

// Occurrences expands a recurring meeting into spans from start to until
// (inclusive by date). Once returns a single span and ignores until/days.
// Custom steps daily and emits only when weekday ∈ days (1=Mon..7=Sun).
// Non-once requires a valid until (date >= start's date); capped at maxOccurrences.
func Occurrences(start, end time.Time, r Recurrence, days []int, until time.Time) ([]Span, error) {
	if r == Once {
		return []Span{{Start: start, End: end}}, nil
	}
	if r == Custom && len(days) == 0 {
		return nil, ErrRecurrenceDays
	}
	if until.IsZero() {
		return nil, ErrRecurrenceWindow
	}
	startDay := dateOnly(start)
	untilDay := dateOnly(until)
	if untilDay.Before(startDay) {
		return nil, ErrRecurrenceWindow
	}
	dayMask := make(map[int]bool, len(days))
	for _, d := range days {
		dayMask[d] = true
	}
	dur := end.Sub(start)
	var spans []Span
	for cur := start; !dateOnly(cur).After(untilDay); cur = nextStep(cur, r) {
		if len(spans) >= maxOccurrences {
			return nil, ErrTooManyOccurrences
		}
		if r == Custom {
			if !dayMask[isoWeekday(cur)] {
				continue
			}
		}
		spans = append(spans, Span{Start: cur, End: cur.Add(dur)})
	}
	return spans, nil
}

// nextStep advances by recurrence's natural step (Custom walks day-by-day,
// the per-day filter in Occurrences keeps only matching weekdays).
func nextStep(t time.Time, r Recurrence) time.Time {
	switch r {
	case Daily, Custom:
		return t.AddDate(0, 0, 1)
	case Weekly:
		return t.AddDate(0, 0, 7)
	case Monthly:
		// AddDate normalizes overflow (e.g. Jan 31 + 1mo → Mar 3), so a series
		// anchored past a short month's end shifts forward — accepted behavior.
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(0, 0, 1)
}

// isoWeekday maps time.Weekday() to 1..7 (Mon..Sun).
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday()) // Sun=0..Sat=6
	if wd == 0 {
		return 7
	}
	return wd
}
```

Remove the old `next` function (replaced by `nextStep`). Verify the import block has `errors`.

- [ ] **Step 6: Update validate.go — Custom requires non-empty days**

Replace the `Validate` body's recurrence check:

```go
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
	if in.Recurrence == Custom && len(in.RecurrenceDays) == 0 {
		return ErrRecurrenceDays
	}
	for _, d := range in.RecurrenceDays {
		if d < 1 || d > 7 {
			return fmt.Errorf("recurrence_days values must be 1..7, got %d", d)
		}
	}
	return nil
}
```

- [ ] **Step 7: Update validate_test.go**

Replace any `Biweekly`-using test with a Custom equivalent. Update `TestValidate_BadRecurrence`:

```go
func TestValidate_BadRecurrence(t *testing.T) {
	in := validBaseInput()
	in.Recurrence = "yearly"
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for unknown recurrence")
	}
}

func TestValidate_CustomRequiresDays(t *testing.T) {
	in := validBaseInput()
	in.Recurrence = Custom
	in.RecurrenceDays = nil
	if !errors.Is(in.Validate(), ErrRecurrenceDays) {
		t.Fatal("expected ErrRecurrenceDays")
	}
}
```

(If `validBaseInput()` doesn't exist, locate the existing test helper that constructs a valid `Input` — it's defined near the top of `validate_test.go`.)

- [ ] **Step 8: Update all Occurrences callers to pass `days`**

Run:
```bash
grep -rn "meeting.Occurrences(" backend/ --include="*.go"
```
Expected: one caller at `backend/internal/application/meeting_service.go:92`.

Patch that line to pass `days`:
```go
spansList, err := meeting.Occurrences(startsAt, endsAt, rec, in.RecurrenceDays, until)
```

But `CreateMeetingInput` doesn't have `RecurrenceDays` yet — we'll wire it in B-T3 (application). For now, pass `nil` here and add a `// TODO(B-T3)` note (this is the ONE step where a transient TODO is OK — it gets removed within the same slice's commit chain):

Actually, do it cleanly: add `RecurrenceDays []int` to `CreateMeetingInput` in `conflict.go` now (B-T1 step) so the compile passes without TODOs.

```go
type CreateMeetingInput struct {
	// existing fields...
	Recurrence      string
	RecurrenceUntil string
	RecurrenceDays  []int  // 1..7; required when Recurrence == "custom".
	Description     string
	Participants    []postgres.MeetingParticipant
}
```

Then patch the `dom := meeting.Input{...}` block in `CreateMeeting` to include:
```go
RecurrenceDays: in.RecurrenceDays,
```

- [ ] **Step 9: Run all backend tests**

Run:
```bash
cd backend && go test ./... 2>&1 | tail -30
```
Expected: all pass. If any test references `Biweekly`, fix it (replace with Custom or remove if redundant).

- [ ] **Step 10: Run lint**

Run:
```bash
make lint 2>&1 | tail -10
```
Expected: `0 issues`.

- [ ] **Step 11: Commit B-T1**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  backend/internal/domain/meeting/meeting.go \
  backend/internal/domain/meeting/recurrence.go \
  backend/internal/domain/meeting/recurrence_test.go \
  backend/internal/domain/meeting/validate.go \
  backend/internal/domain/meeting/validate_test.go \
  backend/internal/application/conflict.go \
  backend/internal/application/meeting_service.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(meeting): add Custom recurrence with weekday days; drop Biweekly

- Recurrence enum: + Custom ("custom"), − Biweekly (unused outside domain).
- Input gains RecurrenceDays []int (1=Mon..7=Sun); validate enforces non-empty
  when Recurrence == Custom and range 1..7 for each entry.
- Occurrences signature gains days; Custom steps daily and emits only matching
  weekdays; reuses existing maxOccurrences cap.
- ErrRecurrenceDays sentinel for "custom needs ≥1 day".
- Application CreateMeetingInput + CreateMeeting wire RecurrenceDays through.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T2: Migration + persistence — recurrence_days JSONB column

**Files:**
- Create: `backend/migrations/20260607120000_meeting_recurrence_days.sql`
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go`
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Write the migration**

Create `backend/migrations/20260607120000_meeting_recurrence_days.sql`:

```sql
-- +goose Up
ALTER TABLE meetings ADD COLUMN recurrence_days JSONB;

-- +goose Down
ALTER TABLE meetings DROP COLUMN IF EXISTS recurrence_days;
```

Nullable; safe online (no rewrite). Existing rows: NULL = non-custom.

- [ ] **Step 2: Run the migration locally**

Run:
```bash
make migrate
```
Expected: migration applied. (If `make migrate` is unavailable in this shell, document it and let the user run it — the integration tests below will surface column-missing errors.)

- [ ] **Step 3: Add RecurrenceDays to the Meeting struct**

In `models.go`, add the field after `RecurrenceUntil`:

```go
type Meeting struct {
	// existing fields...
	SeriesID        *uuid.UUID           `json:"series_id,omitempty"`
	RecurrenceUntil *time.Time           `json:"recurrence_until,omitempty"`
	RecurrenceDays  []int                `json:"recurrence_days,omitempty"`
	Participants    []MeetingParticipant `json:"participants"`
}
```

- [ ] **Step 4: Update meetingCols and scan/insert**

In `meeting_repo.go`, extend the columns list and scan function. Two parallel changes:

```go
const meetingCols = `id, workspace_id, organizer_user_id, dept, type, host,
	starts_at, ends_at, recurrence, name, description, google_event_id, meet_link, status,
	series_id, recurrence_until, recurrence_days`

const meetingColsM = `m.id, m.workspace_id, m.organizer_user_id, m.dept, m.type, m.host,
	m.starts_at, m.ends_at, m.recurrence, m.name, m.description, m.google_event_id, m.meet_link, m.status,
	m.series_id, m.recurrence_until, m.recurrence_days`
```

Update `scanMeeting`:

```go
func scanMeeting(row interface {
	Scan(dest ...any) error
}) (Meeting, error) {
	var m Meeting
	var daysRaw []byte
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.OrganizerUserID, &m.Dept, &m.Type, &m.Host,
		&m.StartsAt, &m.EndsAt, &m.Recurrence, &m.Name, &m.Description, &m.GoogleEventID, &m.MeetLink, &m.Status,
		&m.SeriesID, &m.RecurrenceUntil, &daysRaw)
	if err == nil && len(daysRaw) > 0 {
		err = json.Unmarshal(daysRaw, &m.RecurrenceDays)
	}
	return m, err
}
```

Add `encoding/json` to the import block if missing.

Update `insertMeetingSQL`:

```go
const insertMeetingSQL = `
	INSERT INTO meetings (workspace_id, organizer_user_id, dept, type, host,
		starts_at, ends_at, recurrence, name, description, google_event_id, meet_link,
		series_id, recurrence_until, recurrence_days)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	RETURNING ` + meetingCols
```

Update `meetingInsertArgs`:

```go
func meetingInsertArgs(m Meeting) []any {
	var daysJSON any
	if len(m.RecurrenceDays) > 0 {
		b, _ := json.Marshal(m.RecurrenceDays)
		daysJSON = b
	}
	return []any{m.WorkspaceID, m.OrganizerUserID, m.Dept, m.Type, m.Host,
		m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description, m.GoogleEventID, m.MeetLink,
		m.SeriesID, m.RecurrenceUntil, daysJSON}
}
```

Update any other `Scan(...)` calls that use the raw column list (search for `&mt.RecurrenceUntil,` — there's at least one in `ListMeetingsByOrganizerTelegram`). Each needs the same `daysRaw []byte` + unmarshal pattern.

- [ ] **Step 5: Add ListSeriesAllOccurrences and CancelAllSeriesOccurrences**

After the existing `ListSeriesOccurrences` and `CancelSeriesOccurrences`, add:

```go
// ListSeriesAllOccurrences returns ALL scheduled occurrences of a series in the
// workspace, regardless of start time, ordered by start. Used for whole-series
// edit (slice B). Past occurrences are included.
func (s *Store) ListSeriesAllOccurrences(ctx context.Context, workspaceID, seriesID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE series_id = $1 AND workspace_id = $2 AND status = 'scheduled'
		ORDER BY starts_at`, seriesID, workspaceID)
}

// CancelAllSeriesOccurrences cancels (status='cancelled') ALL scheduled
// occurrences of a series in the workspace, in one atomic statement.
// Returns the count.
func (s *Store) CancelAllSeriesOccurrences(ctx context.Context, workspaceID, seriesID uuid.UUID) (int, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE series_id = $1 AND workspace_id = $2 AND status = 'scheduled'`,
		seriesID, workspaceID)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}
```

- [ ] **Step 6: Run backend tests**

Run:
```bash
cd backend && go test ./... 2>&1 | tail -20
```
Expected: all pass. The postgres-package tests will need the column to exist — if `make migrate` wasn't possible above, ask the user to run it before tests pass.

- [ ] **Step 7: Run lint**

Run:
```bash
make lint 2>&1 | tail -5
```
Expected: `0 issues`.

- [ ] **Step 8: Commit B-T2**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  backend/migrations/20260607120000_meeting_recurrence_days.sql \
  backend/internal/infrastructure/persistence/postgres/models.go \
  backend/internal/infrastructure/persistence/postgres/meeting_repo.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(meetings): persist recurrence_days + add whole-series queries

- Migration: ADD COLUMN recurrence_days JSONB (nullable).
- Meeting.RecurrenceDays []int; scan/insert via json.Marshal/Unmarshal.
- New Store methods: ListSeriesAllOccurrences, CancelAllSeriesOccurrences
  for slice B's scope=whole edit/cancel paths.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T3: Application — UpdateWholeSeries + CancelWholeSeries

**Files:**
- Modify: `backend/internal/application/series_edit.go`

- [ ] **Step 1: Read existing UpdateSeries body**

Run:
```bash
sed -n '60,135p' backend/internal/application/series_edit.go
```
Confirm the existing pattern (auth → load picked → workspace → owner/organizer check → load occurrences → apply → tx update → google patch → enqueue).

- [ ] **Step 2: Add a comment marking UpdateSeries/CancelSeries as internal**

In `series_edit.go`, prefix the existing `UpdateSeries` doc comment with:

```go
// UpdateSeries applies a series-wide edit to the picked occurrence and all later
// ones. Internal: not currently exposed via TMA HTTP (slice B uses scope=whole
// via UpdateWholeSeries). Slice E admin scope may revive this for a
// "from-forward" admin operation.
```

Same prefix on `CancelSeries`. No behavior change.

- [ ] **Step 3: Add UpdateWholeSeries**

Append to `series_edit.go`:

```go
// UpdateWholeSeries applies a series-wide edit to EVERY occurrence in the series
// (including past ones), keyed by series_id. Auth: organizer or workspace owner.
// Returns the count of occurrences updated.
func (s *Services) UpdateWholeSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return 0, fmt.Errorf("bad timezone: %w", err)
	}
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, workspaceID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	rows := make([]postgres.Meeting, 0, len(occs))
	for _, oc := range occs {
		upd, err := applySeriesUpdate(oc, in, loc)
		if err != nil {
			return 0, err
		}
		rows = append(rows, upd)
	}
	if err := s.Store.UpdateMeetingsTx(ctx, workspaceID, rows); err != nil {
		return 0, err
	}
	if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("whole-series update calendar provider", zap.String("workspace_id", workspaceID.String()), zap.Error(ferr))
		}
	} else {
		for _, m := range rows {
			if m.GoogleEventID == "" {
				continue
			}
			if err := calSvc.UpdateEvent(ctx, m.GoogleEventID, CalendarEvent{
				Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
			}); err != nil && s.Log != nil {
				s.Log.Warn("whole-series update event", zap.String("event_id", m.GoogleEventID), zap.Error(err))
			}
		}
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, workspaceID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting updated",
				zap.String("workspace_id", workspaceID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}
```

- [ ] **Step 4: Add CancelWholeSeries**

Append to `series_edit.go`:

```go
// CancelWholeSeries cancels EVERY occurrence in the series (including past ones),
// keyed by series_id. Auth: organizer or workspace owner. Returns the count.
func (s *Services) CancelWholeSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, workspaceID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	n, err := s.Store.CancelAllSeriesOccurrences(ctx, workspaceID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("whole-series cancel calendar provider", zap.String("workspace_id", workspaceID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		s.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	s.enqueueCancelled(ctx, workspaceID, meetingID)
	return n, nil
}
```

- [ ] **Step 5: Build, test, lint**

Run:
```bash
cd backend && go build ./... && go test ./... 2>&1 | tail -20
make lint 2>&1 | tail -5
```
Expected: build clean, all tests pass, 0 lint issues.

- [ ] **Step 6: Commit B-T3**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add backend/internal/application/series_edit.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(application): UpdateWholeSeries + CancelWholeSeries

Adapt UpdateSeries/CancelSeries to operate on the entire series
(keyed by series_id) instead of from-picked-forward. Reuses
applySeriesUpdate, the new ListSeriesAllOccurrences /
CancelAllSeriesOccurrences store methods, same auth, calendar,
notification semantics.

Mark UpdateSeries/CancelSeries as internal (not TMA-exposed) —
they remain available for slice E admin "from-forward" needs.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T4: Application — MeetingSeriesConflicts (TDD)

**Files:**
- Create: `backend/internal/application/series_conflicts.go`
- Create: `backend/internal/application/series_conflicts_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/application/series_conflicts_test.go`:

```go
package application

import (
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
)

func TestExpandSeriesSpans_Once(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, loc).UTC()
	end := time.Date(2026, 6, 1, 11, 0, 0, 0, loc).UTC()
	spans, err := expandSeriesSpans(start, end, meeting.Once, nil, time.Time{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("once should expand to 1, got %d", len(spans))
	}
}

func TestExpandSeriesSpans_CustomThreeWeeks(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, loc).UTC() // Mon
	end := time.Date(2026, 6, 1, 11, 0, 0, 0, loc).UTC()
	until := time.Date(2026, 6, 21, 0, 0, 0, 0, loc)
	// Mon, Wed, Fri × 3 weeks = 9
	spans, err := expandSeriesSpans(start, end, meeting.Custom, []int{1, 3, 5}, until)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(spans) != 9 {
		t.Fatalf("expected 9 spans, got %d", len(spans))
	}
}
```

Run:
```bash
cd backend && go test ./internal/application/... -run TestExpandSeries 2>&1 | tail -10
```
Expected: compile error (`expandSeriesSpans` undefined).

- [ ] **Step 2: Create series_conflicts.go**

Create `backend/internal/application/series_conflicts.go`:

```go
package application

import (
	"context"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/google/uuid"
)

// OccurrenceConflicts is the per-occurrence result of a series conflict check.
type OccurrenceConflicts struct {
	Span      meeting.Span
	Conflicts []Conflict
}

// expandSeriesSpans is the pure expansion helper.
func expandSeriesSpans(start, end time.Time, r meeting.Recurrence, days []int, until time.Time) ([]meeting.Span, error) {
	return meeting.Occurrences(start, end, r, days, until)
}

// MeetingSeriesConflicts expands a hypothetical series and runs the existing
// per-occurrence conflict check against each. Only occurrences with ≥1 conflict
// are returned. Spans are returned in chronological order.
func (s *Services) MeetingSeriesConflicts(ctx context.Context, emails []string, start, end time.Time, r meeting.Recurrence, days []int, until time.Time) ([]OccurrenceConflicts, error) {
	spans, err := expandSeriesSpans(start, end, r, days, until)
	if err != nil {
		return nil, err
	}
	out := make([]OccurrenceConflicts, 0, len(spans))
	for _, sp := range spans {
		cs, err := s.MeetingConflicts(ctx, emails, sp.Start, sp.End, uuid.Nil)
		if err != nil {
			return nil, err
		}
		if len(cs) == 0 {
			continue
		}
		out = append(out, OccurrenceConflicts{Span: sp, Conflicts: cs})
	}
	return out, nil
}
```

- [ ] **Step 3: Run the test — expect pass**

Run:
```bash
cd backend && go test ./internal/application/... -run TestExpandSeries 2>&1 | tail -10
```
Expected: both pass.

- [ ] **Step 4: Full backend test + lint**

Run:
```bash
cd backend && go test ./... 2>&1 | tail -10
make lint 2>&1 | tail -5
```
Expected: all pass, 0 issues.

- [ ] **Step 5: Commit B-T4**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  backend/internal/application/series_conflicts.go \
  backend/internal/application/series_conflicts_test.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(application): MeetingSeriesConflicts (expand → per-occurrence check)

Pure expansion helper expandSeriesSpans wraps meeting.Occurrences;
MeetingSeriesConflicts iterates spans and runs the existing per-occurrence
MeetingConflicts. Returns OccurrenceConflicts grouped by span (empty
results omitted) for the wizard review step.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T5: HTTP — remove recurring block on create; accept until/days

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_write.go`

- [ ] **Step 1: Read current tmaCreateRequest + TMACreateMeeting**

Run:
```bash
grep -n "tmaCreateRequest\|TMACreateMeeting\|meetings_recurring_unsupported" backend/internal/delivery/http/handlers/tma_write.go | head -20
```

- [ ] **Step 2: Extend tmaCreateRequest**

Find the `tmaCreateRequest` struct (probably JSON-tagged camel_case from frontend). Add the two new optional fields:

```go
type tmaCreateRequest struct {
	// existing fields...
	Recurrence      string  `json:"recurrence"`
	RecurrenceUntil *string `json:"recurrence_until,omitempty"`
	RecurrenceDays  *[]int  `json:"recurrence_days,omitempty"`
	// ...
}
```

In `toCreateMeetingInput` (the request→application input mapper), pass them through:

```go
in := application.CreateMeetingInput{
	// existing fields...
	Recurrence: req.Recurrence,
	// ...
}
if req.RecurrenceUntil != nil {
	in.RecurrenceUntil = *req.RecurrenceUntil
}
if req.RecurrenceDays != nil {
	in.RecurrenceDays = *req.RecurrenceDays
}
```

- [ ] **Step 3: Remove the meetings_recurring_unsupported guard**

Locate the early `if rec != meeting.Once { return 400 meetings_recurring_unsupported }` check inside `TMACreateMeeting`. Delete it entirely — the domain `Validate()` now enforces correct recurrence semantics; bad input becomes 400 `validation_failed` via the existing ErrInvalidInput path.

- [ ] **Step 4: Verify build + tests**

Run:
```bash
cd backend && go build ./... && go test ./... 2>&1 | tail -10
```
Expected: all pass.

- [ ] **Step 5: Commit B-T5**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add backend/internal/delivery/http/handlers/tma_write.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): POST /api/tma/meetings accepts recurrence_until + recurrence_days

- tmaCreateRequest gains optional recurrence_until, recurrence_days.
- Drop the slice-A meetings_recurring_unsupported early-return; the
  domain Validate() now handles all recurrence rules and bad inputs
  surface as 400 validation_failed.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T6: HTTP — scope=this|whole on PATCH + DELETE

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_write.go`

- [ ] **Step 1: Locate current TMAUpdateMeeting + TMADeleteMeeting**

Run:
```bash
grep -n "TMAUpdateMeeting\|TMADeleteMeeting\|UpdateMeeting(\|CancelMeeting(" backend/internal/delivery/http/handlers/tma_write.go | head -10
```

- [ ] **Step 2: Add a scope parser helper**

Append to `tma_write.go`:

```go
// parseScope reads ?scope=this|whole from a fiber.Ctx; default "this".
// Returns ("", error) for any other value.
func parseScope(c *fiber.Ctx) (string, error) {
	switch s := c.Query("scope", "this"); s {
	case "this", "whole":
		return s, nil
	default:
		return "", fmt.Errorf("invalid scope %q", s)
	}
}
```

If `fmt` isn't imported in this file yet, add it.

- [ ] **Step 3: Wire scope into TMAUpdateMeeting**

In `TMAUpdateMeeting`, after auth + uuid parse:

```go
scope, err := parseScope(c)
if err != nil {
	return c.Status(400).JSON(fiber.Map{"code": "validation_failed", "message": err.Error()})
}
```

After the editable-workspace check, replace the existing direct `UpdateMeeting` call with a branch:

```go
if scope == "this" {
	updated, err := a.svc.UpdateMeeting(ctx, workspaceID, userID, meetingID, mapToUpdateInput(req))
	// existing error mapping (forbidden → 403, invalid → 400, etc.)
	// existing 200 response
} else {
	// scope == "whole"
	n, err := a.svc.UpdateWholeSeries(ctx, workspaceID, userID, meetingID, mapToSeriesUpdateInput(req))
	if err != nil {
		// reuse same error-to-status mapping; if "%w: not a series" — 400 validation_failed.
	}
	a.log.Info("tma_meeting_whole_series_updated",
		zap.String("meeting_id", meetingID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("count", n),
	)
	// 200 response: refetch the picked occurrence and return it (so the frontend
	// gets the updated row); or just return { "updated": n } and frontend invalidates.
	// Decision: return { "meeting": <picked refetch> } to match scope=this shape.
	return c.JSON(fiber.Map{"meeting": <DTO of refetched>})
}
```

Where `mapToSeriesUpdateInput(req)` is a small helper that converts the existing tmaUpdateRequest (which has `dept/type/host/date/start/end/desc` *strings) to `application.SeriesUpdateInput` (which does NOT accept `Date`; the date is locked for whole-series). If `req.Date != nil`, return 400 `validation_failed` with message "date cannot be changed for whole-series edit".

```go
func mapToSeriesUpdateInput(req tmaUpdateRequest) application.SeriesUpdateInput {
	return application.SeriesUpdateInput{
		Dept:        req.Dept,
		Type:        req.Type,
		Host:        req.Host,
		Description: req.Desc,
		Start:       req.Start,
		End:         req.End,
	}
}
```

- [ ] **Step 4: Wire scope into TMADeleteMeeting**

Same parser pattern:

```go
scope, err := parseScope(c)
if err != nil {
	return c.Status(400).JSON(fiber.Map{"code": "validation_failed", "message": err.Error()})
}
// ...
if scope == "this" {
	if err := a.svc.CancelMeeting(ctx, workspaceID, userID, meetingID); err != nil {
		// existing mapping
	}
} else {
	if _, err := a.svc.CancelWholeSeries(ctx, workspaceID, userID, meetingID); err != nil {
		// same mapping; "not a series" → 400 validation_failed
	}
}
return c.SendStatus(fiber.StatusNoContent)
```

- [ ] **Step 5: Verify build + tests + lint**

Run:
```bash
cd backend && go build ./... && go test ./... 2>&1 | tail -10
make lint 2>&1 | tail -5
```
Expected: all pass, 0 lint.

- [ ] **Step 6: Commit B-T6**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add backend/internal/delivery/http/handlers/tma_write.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): scope=this|whole on PATCH + DELETE /api/tma/meetings/:id

- New parseScope helper; default "this" when query param missing.
- PATCH this → existing UpdateMeeting; PATCH whole → UpdateWholeSeries.
  Whole-series rejects body.date (not changeable series-wide).
- DELETE this → existing CancelMeeting; DELETE whole → CancelWholeSeries.
- "not a series" surfaces as 400 validation_failed.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T7: HTTP — conflicts accepts recurrence params; returns occurrence-grouped shape

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_write.go`

- [ ] **Step 1: Read current tmaConflictRequest + TMAConflicts**

Run:
```bash
grep -n "tmaConflictRequest\|TMAConflicts\|toConflictDTO" backend/internal/delivery/http/handlers/tma_write.go | head -10
```

- [ ] **Step 2: Extend tmaConflictRequest**

Add three optional fields:

```go
type tmaConflictRequest struct {
	// existing fields (Participants, Date, Start, End, ExcludeID)
	Recurrence      *string `json:"recurrence,omitempty"`
	RecurrenceUntil *string `json:"recurrence_until,omitempty"`
	RecurrenceDays  *[]int  `json:"recurrence_days,omitempty"`
}
```

- [ ] **Step 3: Build the occurrence-grouped DTO**

Add to `tma_write.go`:

```go
type tmaOccurrenceConflictsDTO struct {
	Date      string             `json:"date"`
	Start     string             `json:"start"`
	End       string             `json:"end"`
	Conflicts []tmaConflictDTO   `json:"conflicts"`
}

// toOccurrenceConflicts maps an application.OccurrenceConflicts + locale into a wire DTO.
func toOccurrenceConflicts(oc application.OccurrenceConflicts, loc *time.Location) tmaOccurrenceConflictsDTO {
	startLocal := oc.Span.Start.In(loc)
	endLocal := oc.Span.End.In(loc)
	cs := make([]tmaConflictDTO, 0, len(oc.Conflicts))
	for _, c := range oc.Conflicts {
		cs = append(cs, toConflictDTO(c, loc))
	}
	return tmaOccurrenceConflictsDTO{
		Date:      startLocal.Format("2006-01-02"),
		Start:     startLocal.Format("15:04"),
		End:       endLocal.Format("15:04"),
		Conflicts: cs,
	}
}
```

- [ ] **Step 4: Update TMAConflicts to branch on recurrence**

Replace the body's call to `MeetingConflicts` with a branch:

```go
loc, _ := time.LoadLocation("Asia/Almaty")
startLocal, err := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
if err != nil { /* 400 */ }
endLocal, err := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
if err != nil { /* 400 */ }

var occurrences []tmaOccurrenceConflictsDTO

rec := ""
if req.Recurrence != nil { rec = *req.Recurrence }

if rec == "" || rec == string(meeting.Once) {
	cs, err := a.svc.MeetingConflicts(ctx, req.Participants, startLocal.UTC(), endLocal.UTC(), uuid.Nil)
	if err != nil { /* 500 */ }
	cdtos := make([]tmaConflictDTO, 0, len(cs))
	for _, c := range cs {
		cdtos = append(cdtos, toConflictDTO(c, loc))
	}
	occurrences = []tmaOccurrenceConflictsDTO{{
		Date:      req.Date,
		Start:     req.Start,
		End:       req.End,
		Conflicts: cdtos,
	}}
} else {
	var until time.Time
	if req.RecurrenceUntil != nil && *req.RecurrenceUntil != "" {
		until, err = time.ParseInLocation("2006-01-02", *req.RecurrenceUntil, loc)
		if err != nil { /* 400 */ }
	}
	var days []int
	if req.RecurrenceDays != nil { days = *req.RecurrenceDays }
	ocs, err := a.svc.MeetingSeriesConflicts(ctx, req.Participants, startLocal.UTC(), endLocal.UTC(), meeting.Recurrence(rec), days, until)
	if err != nil { /* 400 if domain error, 500 otherwise */ }
	occurrences = make([]tmaOccurrenceConflictsDTO, 0, len(ocs))
	for _, oc := range ocs {
		occurrences = append(occurrences, toOccurrenceConflicts(oc, loc))
	}
}
return c.JSON(fiber.Map{"occurrences": occurrences})
```

- [ ] **Step 5: Verify build + tests + lint**

Run:
```bash
cd backend && go build ./... && go test ./... 2>&1 | tail -10
make lint 2>&1 | tail -5
```
Expected: all pass, 0 lint.

- [ ] **Step 6: Commit B-T7**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add backend/internal/delivery/http/handlers/tma_write.go

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): POST /api/tma/conflicts accepts recurrence; returns occurrence-grouped

- Request gains optional recurrence, recurrence_until, recurrence_days.
- Response always returns { "occurrences": [{date,start,end,conflicts:[]}] }:
  - once (or absent) → one-element array with the single check.
  - series → expanded via MeetingSeriesConflicts, only non-empty occurrences.
- Conflicts rendered in Almaty TZ (HH:MM) via existing toConflictDTO.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T8: OpenAPI + frontend schema regen

**Files:**
- Modify: `backend/openapi/openapi.json`
- Modify: `backend/docs/openapi.json` (parallel byte-identical mirror)
- Regen: `frontend/src/shared/api/generated/schema.ts`

- [ ] **Step 1: Update openapi.json — POST /api/tma/meetings request**

In the request schema for `tmaCreateMeeting` (the existing component), add:

```json
"recurrence_until": { "type": "string", "format": "date" },
"recurrence_days": {
  "type": "array",
  "items": { "type": "integer", "minimum": 1, "maximum": 7 }
}
```

Both optional (no `required` list change).

- [ ] **Step 2: PATCH and DELETE — add scope query param**

For both `paths["/api/tma/meetings/{id}"].patch` and `.delete`, add to their `parameters`:

```json
{
  "name": "scope",
  "in": "query",
  "required": false,
  "schema": { "type": "string", "enum": ["this", "whole"], "default": "this" }
}
```

- [ ] **Step 3: POST /api/tma/conflicts — request fields + new response**

Add to `TmaConflictsRequest` schema:

```json
"recurrence": { "type": "string", "enum": ["once","daily","weekly","custom","monthly"] },
"recurrence_until": { "type": "string", "format": "date" },
"recurrence_days": {
  "type": "array",
  "items": { "type": "integer", "minimum": 1, "maximum": 7 }
}
```

Replace `TmaConflictsResponse` body schema with:

```json
{
  "type": "object",
  "required": ["occurrences"],
  "properties": {
    "occurrences": {
      "type": "array",
      "items": { "$ref": "#/components/schemas/TmaOccurrenceConflicts" }
    }
  }
}
```

Add new component schema `TmaOccurrenceConflicts`:

```json
"TmaOccurrenceConflicts": {
  "type": "object",
  "required": ["date","start","end","conflicts"],
  "properties": {
    "date": { "type": "string", "format": "date" },
    "start": { "type": "string", "example": "10:00" },
    "end": { "type": "string", "example": "11:00" },
    "conflicts": {
      "type": "array",
      "items": { "$ref": "#/components/schemas/TmaConflict" }
    }
  }
}
```

Keep the existing `TmaConflict` schema unchanged.

- [ ] **Step 4: Mirror to backend/docs/openapi.json**

Run:
```bash
cp /Users/temirlan/Workspace/in-house/lead-cat/backend/openapi/openapi.json \
   /Users/temirlan/Workspace/in-house/lead-cat/backend/docs/openapi.json
diff /Users/temirlan/Workspace/in-house/lead-cat/backend/openapi/openapi.json \
     /Users/temirlan/Workspace/in-house/lead-cat/backend/docs/openapi.json
```
Expected: no diff output (byte-identical).

- [ ] **Step 5: Regen frontend schema**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && \
  pnpm dlx openapi-typescript ../backend/openapi/openapi.json -o src/shared/api/generated/schema.ts
```
Expected: schema regenerated. Inspect the diff briefly to confirm the new fields/schemas are present:

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat diff --stat frontend/src/shared/api/generated/schema.ts
```

- [ ] **Step 6: Backend build + frontend typecheck**

Run:
```bash
cd backend && go build ./...
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck
```
Expected: clean. (Typecheck may flag usage sites where the old conflict response shape was assumed — fix those in B-T9, not here.)

If typecheck fails ONLY at the conflicts response consumer sites, that's expected — those are wired in B-T9. Proceed to commit OpenAPI separately so the regen lands in its own commit.

- [ ] **Step 7: Commit B-T8**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  backend/openapi/openapi.json \
  backend/docs/openapi.json \
  frontend/src/shared/api/generated/schema.ts

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(api): document slice B recurrence in OpenAPI + frontend regen

- POST /api/tma/meetings: optional recurrence_until + recurrence_days.
- PATCH/DELETE /api/tma/meetings/{id}: scope=this|whole query param.
- POST /api/tma/conflicts: optional recurrence + recurrence_until +
  recurrence_days; response replaced with occurrence-grouped shape.
- New TmaOccurrenceConflicts component schema.
- Regen frontend types from updated openapi.json.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T9: Frontend types + mutations — scope, until, days, OccurrenceConflicts

**Files:**
- Modify: `frontend/src/features/meetings/api.ts`
- Modify: `frontend/src/features/meetings/queries.ts`
- Modify: `frontend/src/entities/meeting/types.ts`
- Modify: `frontend/src/entities/meeting/lib/format.ts`

- [ ] **Step 1: Add seriesId to Meeting type and DTO mapping**

In `entities/meeting/types.ts`, add to the `Meeting` type:

```ts
export type Meeting = {
  // existing fields
  rec: string
  recDays?: number[]
  seriesId?: string
  // ...
}
```

In `features/meetings/api.ts`, add `series_id?: string` to the private `MeetingDTO` type and map it in `toMeeting`:

```ts
type MeetingDTO = {
  // existing
  series_id?: string
  // ...
}

function toMeeting(d: MeetingDTO): Meeting {
  return {
    // existing
    rec: d.rec,
    seriesId: d.series_id,
    // ...
  }
}
```

In `entities/meeting/lib/format.ts`, the `draftToMeeting` and `detailToDraft` already handle most fields — no change needed for seriesId (it's a read-only field on Meeting; drafts don't carry it).

- [ ] **Step 2: Extend MeetingInput, ConflictsParams + add OccurrenceConflicts**

In `features/meetings/api.ts`:

```ts
export type MeetingInput = {
  // existing fields
  recurrence: string
  desc: string
  participants: string[]
  recurrence_until?: string
  recurrence_days?: number[]
}

export type OccurrenceConflicts = {
  date: string
  start: string
  end: string
  conflicts: Conflict[]
}

export type ConflictsParams = {
  participants: string[]
  date: string
  start: string
  end: string
  excludeId?: string
  recurrence?: string
  recurrenceUntil?: string
  recurrenceDays?: number[]
}
```

- [ ] **Step 3: Update fetchers**

In `features/meetings/api.ts`:

```ts
export async function updateMeeting(
  id: string,
  patch: MeetingPatch,
  opts?: { scope?: "this" | "whole" }
): Promise<Meeting> {
  const scope = opts?.scope ?? "this"
  const data = await apiFetch<{ meeting: MeetingDTO }>(`/tma/meetings/${id}`, {
    method: "PATCH",
    body: patch,
    params: { scope },
  })
  return toMeeting(data.meeting)
}

export async function deleteMeeting(
  id: string,
  opts?: { scope?: "this" | "whole" }
): Promise<void> {
  const scope = opts?.scope ?? "this"
  await apiFetch<void>(`/tma/meetings/${id}`, {
    method: "DELETE",
    params: { scope },
  })
}

export async function fetchConflicts(
  params: ConflictsParams
): Promise<OccurrenceConflicts[]> {
  const data = await apiFetch<{ occurrences: OccurrenceConflicts[] }>(
    "/tma/conflicts",
    {
      method: "POST",
      body: {
        participants: params.participants,
        date: params.date,
        start: params.start,
        end: params.end,
        exclude_id: params.excludeId ?? "",
        recurrence: params.recurrence,
        recurrence_until: params.recurrenceUntil,
        recurrence_days: params.recurrenceDays,
      },
    }
  )
  return data.occurrences
}
```

The `createMeeting` fetcher needs no signature change — `MeetingInput` already grew the new optional fields, and they'll be present in the JSON body when set.

- [ ] **Step 4: Update mutation hooks**

In `features/meetings/queries.ts`:

```ts
export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: {
      id: string
      patch: MeetingPatch
      scope?: "this" | "whole"
    }) => updateMeeting(args.id, args.patch, { scope: args.scope }),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; scope?: "this" | "whole" } | string) => {
      if (typeof args === "string") return deleteMeeting(args)
      return deleteMeeting(args.id, { scope: args.scope })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}
```

Keeping `useDeleteMeeting` callable with either a bare id (slice-A call sites) OR the new args object — preserves backwards-compatibility for `handleDelete(detail.id)` in `meetings-list-page.tsx` while letting B-T12 pass `{ id, scope }`.

`useConflicts` return type changes from `Conflict[]` to `OccurrenceConflicts[]` — no other hook changes, but the consumer in `use-create-wizard.ts` will need updating (B-T10).

- [ ] **Step 5: Update existing call sites that break**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck 2>&1 | tail -40
```

Likely failure points:
- `use-create-wizard.ts` reads `conflictsMut.data` as `Conflict[]` — will become `OccurrenceConflicts[]`. Defer the actual rewrite to B-T10; for now, narrow the read to `conflictsMut.data?.[0]?.conflicts ?? []` so typecheck passes and existing single-occurrence behavior is preserved.
- `meetings-list-page.tsx` calls `handleDelete(id)` → still works via the string overload above.
- `create-page.tsx` calls `useUpdateMeeting().mutateAsync({ id, patch })` — still works (scope is optional).

Apply the narrow fix in `use-create-wizard.ts`:

```ts
const conflictPeople = useMemo(() => {
  const list = conflictsMut.data?.[0]?.conflicts ?? []
  // ... existing name-formatting code unchanged
}, [conflictsMut.data])
```

- [ ] **Step 6: Frontend typecheck, format, build**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck && pnpm format && pnpm build
```
Expected: all clean.

- [ ] **Step 7: Commit B-T9**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  frontend/src/features/meetings/api.ts \
  frontend/src/features/meetings/queries.ts \
  frontend/src/entities/meeting/types.ts \
  frontend/src/features/meeting-create/lib/use-create-wizard.ts

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): frontend types + mutations grow scope/until/days

- Meeting.seriesId? (from new series_id DTO field).
- MeetingInput accepts optional recurrence_until + recurrence_days.
- updateMeeting/deleteMeeting accept { scope?: "this" | "whole" };
  default "this" preserves slice-A behavior.
- useUpdateMeeting/useDeleteMeeting forward scope; useDeleteMeeting
  remains bare-id callable for existing call sites.
- fetchConflicts returns OccurrenceConflicts[] (always one-element
  for once, expanded for series). Wizard temporarily reads .[0].
- ConflictsParams gains recurrence + recurrenceUntil + recurrenceDays.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T10: Wizard — until picker, smart defaults, recurringBlocked drop, lockedFields

**Files:**
- Modify: `frontend/src/features/meeting-create/lib/use-create-wizard.ts`
- Modify: `frontend/src/features/meeting-create/components/wizard-step-when.tsx`
- Modify: `frontend/src/features/meeting-create/components/create-wizard.tsx`
- Modify: `frontend/src/shared/tma/i18n.ts`

- [ ] **Step 1: Add until + smartDefaults to MeetingDraft**

In `entities/meeting/types.ts` (already touched by B-T9), add to `MeetingDraft`:

```ts
export type MeetingDraft = {
  // existing fields
  rec: string
  recDays: number[]
  until: string  // YYYY-MM-DD; empty when rec === "once"
  participants: Employee[]
  desc: string
}
```

Update `format.ts` `detailToDraft` to compute `until: m.recurrenceUntil ?? ""` (if Meeting carries it; if not, leave empty — `Meeting` already has `recurrenceUntil` via the API → no, that's only in `postgres.Meeting`. Frontend `Meeting` doesn't yet. ADD it to `Meeting` and DTO mapping):

```ts
// types.ts
export type Meeting = {
  // ...
  rec: string
  recDays?: number[]
  recurrenceUntil?: string
  seriesId?: string
}
```

```ts
// api.ts MeetingDTO + toMeeting
type MeetingDTO = {
  // ...
  recurrence_until?: string
}
function toMeeting(d: MeetingDTO): Meeting {
  return {
    // ...
    recurrenceUntil: d.recurrence_until,
  }
}
```

`detailToDraft`:

```ts
return {
  // ...
  rec: m.rec,
  recDays: m.recDays ?? [],
  until: m.recurrenceUntil ?? "",
  // ...
}
```

- [ ] **Step 2: Update use-create-wizard.ts — smart default + until in canNext**

Inside `useCreateWizard`:

```ts
const [draft, setDraft] = useState<MeetingDraft>(() => ({
  dept: "",
  type: "",
  host: ME.name,
  date: "",
  start: "10:00",
  dur: 30,
  rec: "once",
  recDays: [],
  until: "",
  participants: [],
  desc: "",
  ...initial,
}))
```

Add a helper to compute the smart default:

```ts
function defaultUntil(date: string, rec: string): string {
  if (!date || rec === "once") return ""
  const [y, m, d] = date.split("-").map(Number)
  const dt = new Date(y, m - 1, d)
  switch (rec) {
    case "daily":
      dt.setDate(dt.getDate() + 30)
      break
    case "weekly":
    case "custom":
      dt.setDate(dt.getDate() + 7 * 12)
      break
    case "monthly":
      dt.setMonth(dt.getMonth() + 12)
      break
    default:
      return ""
  }
  return `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, "0")}-${String(dt.getDate()).padStart(2, "0")}`
}
```

Replace the `set` function with one that auto-fills `until` when `rec` changes:

```ts
const set = <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) =>
  setDraft((d) => {
    const nd = { ...d, [k]: v }
    if (k === "rec" || (k === "date" && d.rec !== "once" && !d.until)) {
      nd.until = defaultUntil(nd.date, nd.rec)
    }
    return nd
  })
```

Remove the existing `recurringBlocked` line (`const recurringBlocked = draft.rec !== "once"`).

Extend `canNext`:

```ts
const canNext = (() => {
  const step_ = WIZARD_STEPS[step]
  if (step_ === "what") return Boolean(draft.dept && draft.type)
  if (step_ === "when") {
    if (!draft.date || !draft.start) return false
    if (draft.rec !== "once") {
      if (!draft.until) return false
      if (draft.until < draft.date) return false
      if (draft.rec === "custom" && draft.recDays.length === 0) return false
    }
    return true
  }
  if (step_ === "who") return Boolean(draft.host)
  if (step_ === "review") return true
  return false
})()
```

Update the `useEffect` that fires `useConflicts` to include recurrence params:

```ts
useEffect(() => {
  if (WIZARD_STEPS[step] !== "review") return
  if (!draft.date || !draft.start || !endTime) return
  if (!draft.participants.length) return
  const emails = draft.participants.map((p) => p.email)
  conflictsMut.mutate({
    participants: emails,
    date: draft.date,
    start: draft.start,
    end: endTime,
    recurrence: draft.rec !== "once" ? draft.rec : undefined,
    recurrenceUntil: draft.rec !== "once" ? draft.until : undefined,
    recurrenceDays: draft.rec === "custom" ? draft.recDays : undefined,
  })
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [step, draft.date, draft.start, endTime, draft.participants, draft.rec, draft.until, draft.recDays])
```

Update `conflictPeople` to flatten across occurrences (drop the temporary `.[0]` shim from B-T9):

```ts
const conflictPeople = useMemo(() => {
  const list = conflictsMut.data ?? []
  const names = new Set<string>()
  list.forEach((oc) => {
    oc.conflicts.forEach((c) => {
      const parts = c.name.split(" ")
      names.add(parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : ""))
    })
  })
  return [...names]
}, [conflictsMut.data])
```

Expose the raw occurrences for the review step too:

```ts
const conflictOccurrences = conflictsMut.data ?? []
```

Hook return adds `conflictOccurrences` and removes `recurringBlocked`. Also accept a new optional input `lockedFields`:

```ts
export function useCreateWizard({
  initial,
  onComplete,
  lockedFields,
}: {
  initial?: Partial<MeetingDraft>
  onComplete: (m: MeetingDraft & { end: string }) => void
  lockedFields?: { date?: boolean; rec?: boolean; until?: boolean; participants?: boolean }
}) {
  // ...
  return {
    // existing return
    conflictOccurrences,
    lockedFields: lockedFields ?? {},
  }
}
```

- [ ] **Step 3: Add i18n keys**

In `frontend/src/shared/tma/i18n.ts`, add to each of `ru/kk/en` blocks:

```ts
// ru
untilLabel: "Дата окончания",
untilPlaceholder: "Когда серия заканчивается",
untilRequired: "Укажите дату окончания серии",
untilBeforeStart: "Дата окончания должна быть не раньше первой встречи",
seriesConflicts: "Конфликты по датам",
seriesConflictsMore: "+ ещё {count}",
editThis: "Эту встречу",
editSeries: "Всю серию",
delThis: "Эту встречу",
delSeries: "Всю серию",
seriesEditLockedNote: "Дата и периодичность серии не редактируются",
```

```ts
// kk (Kazakh translations)
untilLabel: "Аяқталу күні",
untilPlaceholder: "Сериал қашан аяқталады",
untilRequired: "Сериалдың аяқталу күнін көрсетіңіз",
untilBeforeStart: "Аяқталу күні бірінші кездесуден кейін болуы керек",
seriesConflicts: "Күндер бойынша қайшылықтар",
seriesConflictsMore: "+ тағы {count}",
editThis: "Осы кездесуді",
editSeries: "Бүкіл серияны",
delThis: "Осы кездесуді",
delSeries: "Бүкіл серияны",
seriesEditLockedNote: "Сериалдың күні мен жиілігі өзгертілмейді",
```

```ts
// en
untilLabel: "End date",
untilPlaceholder: "When the series ends",
untilRequired: "Pick the end date of the series",
untilBeforeStart: "End date must be on or after the first meeting",
seriesConflicts: "Conflicts by date",
seriesConflictsMore: "+ {count} more",
editThis: "This meeting",
editSeries: "Whole series",
delThis: "This meeting",
delSeries: "Whole series",
seriesEditLockedNote: "Series date and recurrence are not editable",
```

- [ ] **Step 4: Render the until picker in wizard-step-when.tsx**

After the existing rec selector + recDays chips, add:

```tsx
{draft.rec !== "once" && (
  <div style={{ marginTop: 14 }}>
    <label style={{
      display: "block",
      fontSize: 12,
      fontWeight: 700,
      color: p.muted,
      marginBottom: 6,
    }}>
      {t("untilLabel")}
    </label>
    <input
      type="date"
      value={draft.until}
      min={draft.date || undefined}
      onChange={(e) => set("until", e.target.value)}
      placeholder={t("untilPlaceholder")}
      disabled={lockedFields?.until}
      style={{
        width: "100%",
        padding: "10px 12px",
        borderRadius: 12,
        border: `1px solid ${p.border}`,
        background: p.tgBar,
        color: p.text,
        fontSize: 15,
      }}
    />
    {!draft.until && (
      <div style={{ color: p.danger, fontSize: 12, marginTop: 6 }}>
        {t("untilRequired")}
      </div>
    )}
    {draft.until && draft.until < draft.date && (
      <div style={{ color: p.danger, fontSize: 12, marginTop: 6 }}>
        {t("untilBeforeStart")}
      </div>
    )}
  </div>
)}
```

The component needs `lockedFields` in its props type — add it; pass-through from `create-wizard.tsx`.

If `scope === "whole"` (whole-series edit), add a banner at the top of step-when:

```tsx
{lockedFields?.rec && (
  <div style={{
    background: p.accentSoft,
    borderRadius: 12,
    padding: "10px 12px",
    marginBottom: 14,
    fontSize: 13,
    color: p.text,
  }}>
    {t("seriesEditLockedNote")}
  </div>
)}
```

- [ ] **Step 5: Drop recurringBlocked from create-wizard.tsx**

In `create-wizard.tsx`, the destructuring drops `recurringBlocked` (no longer returned). Confirm button disabled rule:

```tsx
<CatBtn
  variant="primary"
  full
  disabled={!canNext}
  onClick={() => go(1)}
>
```

Pass `lockedFields` through (caller might supply it):

```tsx
export function CreateWizard({
  initial,
  onComplete,
  lockedFields,
}: {
  initial?: Partial<MeetingDraft>
  onComplete: (m: MeetingDraft & { end: string }) => void
  lockedFields?: { date?: boolean; rec?: boolean; until?: boolean; participants?: boolean }
}) {
  // ...
  const wizard = useCreateWizard({ initial, onComplete, lockedFields })
```

Pass `lockedFields={wizard.lockedFields}` to the `<WizardStepWhen>` component.

- [ ] **Step 6: Frontend typecheck + format + build**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck && pnpm format && pnpm build
```
Expected: all clean.

- [ ] **Step 7: Commit B-T10**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  frontend/src/features/meeting-create/lib/use-create-wizard.ts \
  frontend/src/features/meeting-create/components/wizard-step-when.tsx \
  frontend/src/features/meeting-create/components/create-wizard.tsx \
  frontend/src/shared/tma/i18n.ts \
  frontend/src/entities/meeting/types.ts \
  frontend/src/entities/meeting/lib/format.ts \
  frontend/src/features/meetings/api.ts

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): wizard until-picker, smart defaults, recurring guard removed

- MeetingDraft.until added; wizard auto-fills smart default per rec
  (daily +30d, weekly/custom +12w, monthly +12m). Manual edit allowed.
- wizard-step-when renders required date input when rec != "once" with
  inline errors for empty / before-start dates.
- canNext gates review step on until validity and (for custom) ≥1 day.
- useConflicts now sends recurrence params; conflict-people flattens
  across all expanded occurrences.
- Drop the slice-A recurringBlocked flag from the hook + confirm button.
- lockedFields prop threads down for whole-series edit (banner + disabled
  inputs); consumed in B-T12.
- i18n keys (ru/kk/en) for end-date + series-edit labels.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T11: Wizard review — grouped-by-date conflicts list

**Files:**
- Modify: `frontend/src/features/meeting-create/components/wizard-step-review.tsx`
- Modify: `frontend/src/features/meeting-create/components/create-wizard.tsx`

- [ ] **Step 1: Update wizard-step-review props**

In `wizard-step-review.tsx`, replace the `conflictPeople: string[]` prop with `conflictOccurrences: OccurrenceConflicts[]` (import the type):

```tsx
import type { OccurrenceConflicts } from "@/features/meetings/api"
// ...
export function WizardStepReview({
  draft,
  endTime,
  finalMeeting,
  conflictOccurrences,
}: {
  draft: MeetingDraft
  endTime: string
  finalMeeting: MeetingDraft & { end: string; organizer: string }
  conflictOccurrences: OccurrenceConflicts[]
}) {
```

Remove the existing `conflictPeople`-driven warning. Replace with two paths:

```tsx
{conflictOccurrences.length > 0 && draft.rec === "once" && (
  // Existing single-occurrence warning, reading conflictOccurrences[0].conflicts
  // for the names.
)}
{conflictOccurrences.length > 0 && draft.rec !== "once" && (
  <div style={{ /* same dangerSoft styling */ }}>
    <div style={{ /* header */ }}>⚠️ {t("seriesConflicts")}</div>
    <div style={{ fontSize: 13, color: p.text, opacity: 0.85, lineHeight: 1.5 }}>
      {conflictOccurrences.slice(0, 5).map((oc) => {
        const peopleNames = Array.from(new Set(oc.conflicts.map((c) => {
          const parts = c.name.split(" ")
          return parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : "")
        })))
        return (
          <div key={oc.date} style={{ marginTop: 4 }}>
            <strong>{oc.date}</strong> {oc.start}–{oc.end}: {peopleNames.join(", ")}
          </div>
        )
      })}
      {conflictOccurrences.length > 5 && (
        <div style={{ marginTop: 6, opacity: 0.7 }}>
          {t("seriesConflictsMore").replace("{count}", String(conflictOccurrences.length - 5))}
        </div>
      )}
    </div>
  </div>
)}
```

- [ ] **Step 2: Drop the recurringSoon block**

Remove any `{recurringBlocked && (...)}` or `t("recurringSoon")` JSX from `wizard-step-review.tsx`. (Slice A's guardrail is gone.)

Keep the `recurringSoon` key in `i18n.ts` itself — it's still mapped by `write-error.ts` defensively for one slice (see spec). Remove in slice H.

- [ ] **Step 3: Update create-wizard.tsx to pass conflictOccurrences**

```tsx
<WizardStepReview
  draft={draft}
  endTime={endTime}
  finalMeeting={finalMeeting}
  conflictOccurrences={wizard.conflictOccurrences}
/>
```

Drop the `conflictPeople={conflictPeople}` prop spread (the destructured `conflictPeople` from `wizard` was only used here and in the confirm-button label below — update the label too):

```tsx
<CatBtn
  variant="primary"
  full
  disabled={!canNext}
  onClick={() => go(1)}
>
  {step === WIZARD_STEPS.length - 1
    ? wizard.conflictOccurrences.length
      ? `🐾 ${t("proceed")}`
      : `🐾 ${t("confirmCreate")}`
    : t("next")}
</CatBtn>
```

`conflictPeople` can stay in the hook return for now (unused but harmless), or be removed — REMOVE it to keep things clean. Adjust `use-create-wizard.ts` return: drop `conflictPeople`, keep `conflictOccurrences`.

- [ ] **Step 4: Typecheck, format, build**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck && pnpm format && pnpm build
```
Expected: clean.

- [ ] **Step 5: Commit B-T11**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  frontend/src/features/meeting-create/components/wizard-step-review.tsx \
  frontend/src/features/meeting-create/components/create-wizard.tsx \
  frontend/src/features/meeting-create/lib/use-create-wizard.ts

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): review step shows series conflicts grouped by date

- Replace single conflictPeople list with conflictOccurrences[] from
  the hook; renders up to 5 dates, "+ N more" overflow.
- Once path uses occurrences[0].conflicts (single-element).
- Confirm-button label reads from wizard.conflictOccurrences.length.
- Drop conflictPeople from the hook return (now unused).

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task B-T12: Meeting detail — dual-scope edit/cancel + create-page wiring

**Files:**
- Modify: `frontend/src/features/meetings/components/meeting-detail.tsx`
- Modify: `frontend/src/features/meetings/components/meeting-detail-actions.tsx`
- Modify: `frontend/src/features/meetings/pages/meetings-list-page.tsx`
- Modify: `frontend/src/features/meeting-create/pages/create-page.tsx`
- Modify: route file for `/meetings/create/$editId` (search-param schema)

- [ ] **Step 1: Locate the editId route file**

Run:
```bash
ls frontend/src/routes/_tma/ | grep -i create
```
Expected: a file like `meetings.create.$editId.tsx` (TanStack Router file route). Read it; note where its `validateSearch` (or equivalent) lives.

- [ ] **Step 2: Add scope to the route's search-param schema**

Modify the search schema to accept `scope?: "this" | "whole"`:

```ts
export const Route = createFileRoute("/_tma/meetings/create/$editId")({
  validateSearch: (search) => {
    const scope = search.scope === "whole" ? "whole" : "this"
    return { scope: scope as "this" | "whole" }
  },
  component: CreateMeetingPage,
})
```

- [ ] **Step 3: Update meeting-detail-actions.tsx — dual-scope buttons via bottom sheet**

Replace the props signature:

```tsx
export function MeetingDetailActions({
  isSeries,
  onEdit,
  onDelete,
}: {
  isSeries: boolean
  onEdit: (scope: "this" | "whole") => void
  onDelete: (scope: "this" | "whole") => void
}) {
  const p = useTmaApp()
  const t = p.t
  const [editSheet, setEditSheet] = useState(false)
  const [delSheet, setDelSheet] = useState(false)

  return (
    <>
      <div style={{ display: "flex", gap: 10, marginTop: 18 }}>
        <CatBtn
          variant="outline"
          full
          icon={<CatIcon name="pencil" size={18} color={p.text} sw={2} />}
          onClick={() => (isSeries ? setEditSheet(true) : onEdit("this"))}
        >
          {t("edit")}
        </CatBtn>
        <CatBtn
          variant="danger"
          icon={<CatIcon name="trash" size={18} color={p.danger} sw={2} />}
          onClick={() => (isSeries ? setDelSheet(true) : onDelete("this"))}
          style={{ flex: "0 0 auto" }}
        >
          {t("del")}
        </CatBtn>
      </div>
      <Sheet open={editSheet} onClose={() => setEditSheet(false)} maxH="40%">
        <div style={{ display: "flex", flexDirection: "column", gap: 10, padding: 16 }}>
          <CatBtn variant="outline" full onClick={() => { setEditSheet(false); onEdit("this") }}>
            {t("editThis")}
          </CatBtn>
          <CatBtn variant="primary" full onClick={() => { setEditSheet(false); onEdit("whole") }}>
            {t("editSeries")}
          </CatBtn>
        </div>
      </Sheet>
      <Sheet open={delSheet} onClose={() => setDelSheet(false)} maxH="40%">
        <div style={{ display: "flex", flexDirection: "column", gap: 10, padding: 16 }}>
          <CatBtn variant="outline" full onClick={() => { setDelSheet(false); onDelete("this") }}>
            {t("delThis")}
          </CatBtn>
          <CatBtn variant="danger" full onClick={() => { setDelSheet(false); onDelete("whole") }}>
            {t("delSeries")}
          </CatBtn>
        </div>
      </Sheet>
    </>
  )
}
```

Imports: `useState` from `react`, `Sheet` from `@/components/tma-shell`.

- [ ] **Step 4: Update meeting-detail.tsx — pass isSeries**

In `meeting-detail.tsx`:

```tsx
const isSeries = Boolean(m.seriesId)
// ...
{canManage && !past && (
  <MeetingDetailActions
    isSeries={isSeries}
    onEdit={(scope) => onEdit(scope)}
    onDelete={(scope) => onDelete(scope)}
  />
)}
```

Update the props type:

```tsx
export function MeetingDetail({
  m,
  onEdit,
  onDelete,
}: {
  m: Meeting
  onEdit: (scope: "this" | "whole") => void
  onDelete: (scope: "this" | "whole") => void
}) {
```

- [ ] **Step 5: Update meetings-list-page.tsx — wire scope through**

```tsx
<MeetingDetail
  m={detail}
  onEdit={(scope) => {
    void navigate({
      to: "/meetings/create/$editId",
      params: { editId: detail.id },
      search: { scope },
    })
  }}
  onDelete={(scope) => void handleDelete(detail.id, scope)}
/>
```

`handleDelete`:

```tsx
const handleDelete = async (id: string, scope: "this" | "whole" = "this") => {
  try {
    await deleteMut.mutateAsync({ id, scope })
    closeDetail()
    toastSuccess(t("deleted"))
  } catch (err) {
    toastError(err, t(writeErrorKey(err)))
  }
}
```

- [ ] **Step 6: Update create-page.tsx — read scope, pass lockedFields, scope mutation**

```tsx
const search = useSearch({ strict: false }) as { scope?: "this" | "whole" }
const scope: "this" | "whole" = search.scope === "whole" ? "whole" : "this"
const lockedFields = scope === "whole"
  ? { date: true, rec: true, until: true, participants: true }
  : undefined

// ...

const completeCreate = async (m: MeetingDraft & { end: string }) => {
  const participants = m.participants.map((x) => x.email)
  try {
    if (editId) {
      const patch: MeetingPatch = {
        dept: m.dept,
        type: m.type,
        host: m.host,
        date: scope === "whole" ? undefined : m.date,  // backend rejects date for whole
        start: m.start,
        end: m.end,
        desc: m.desc,
      }
      await updateMut.mutateAsync({ id: editId, patch, scope })
      toastSuccess(p.t("updated"))
      void navigate({ to: "/meetings", search: { scope: "upcoming" } })
    } else {
      const input: MeetingInput = {
        dept: m.dept, type: m.type, host: m.host,
        date: m.date, start: m.start, end: m.end,
        recurrence: m.rec, desc: m.desc, participants,
        recurrence_until: m.rec !== "once" ? m.until : undefined,
        recurrence_days: m.rec === "custom" ? m.recDays : undefined,
      }
      const created = await createMut.mutateAsync(input)
      void navigate({
        to: "/meetings",
        search: { scope: "upcoming", success: created.id },
      })
    }
  } catch (err) {
    toastError(err, p.t(writeErrorKey(err)))
  }
}

return (
  <Overlay open onClose={goBack} onBack={goBack} title={p.t("create")}>
    <CreateWizard initial={initial} onComplete={completeCreate} lockedFields={lockedFields} />
  </Overlay>
)
```

`MeetingPatch.date` is currently `string` — make it optional in `api.ts`:

```ts
export type MeetingPatch = Partial<{
  dept: string
  type: string
  host: string
  date: string
  start: string
  end: string
  desc: string
}>
```

It already is `Partial<…>`, so `date: undefined` is fine — `apiFetch` will omit it.

- [ ] **Step 7: Typecheck, format, build**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck && pnpm format && pnpm build
```
Expected: clean.

- [ ] **Step 8: Commit B-T12**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add \
  frontend/src/features/meetings/components/meeting-detail.tsx \
  frontend/src/features/meetings/components/meeting-detail-actions.tsx \
  frontend/src/features/meetings/pages/meetings-list-page.tsx \
  frontend/src/features/meeting-create/pages/create-page.tsx \
  frontend/src/routes/_tma/

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
feat(tma): series-aware edit/cancel on meeting detail; scope param in route

- meeting-detail-actions renders dual-scope bottom-sheet when isSeries
  ("this meeting" / "whole series" for both edit + delete); single-button
  flow preserved for non-series meetings.
- meeting-detail forwards isSeries + onEdit(scope)/onDelete(scope).
- meetings-list handleDelete accepts scope; useDeleteMeeting called with
  { id, scope }.
- /meetings/create/$editId route accepts ?scope=this|whole; create-page
  reads it, locks fields for whole-series edit, suppresses body.date,
  and forwards scope to useUpdateMeeting.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

(Adjust the `git add frontend/src/routes/_tma/` to the actual file path you modified in Step 1.)

---

## Task B-T13: Docs refresh + final verification

**Files:**
- Modify: `docs/MEETINGS.md`
- Modify: `docs/API.md`

- [ ] **Step 1: Update MEETINGS.md status note**

Replace the slice-A status line:

> **Status:** TMA auth, all read paths, all non-recurring write paths, and recurring series are live. Create, edit, delete, conflict warnings, and scope-aware (this/whole) series edits all flow through `/api/tma/*`. Frontend still uses mock fixtures in a few places; see `frontend/README.md` for layout.

Update the **Write paths** table:

```markdown
| Route                          | Status                                          |
| ------------------------------ | ----------------------------------------------- |
| `POST /api/tma/meetings`       | Done (incl. recurring with recurrence_until / recurrence_days) |
| `PATCH /api/tma/meetings/:id`  | Done (organizer-only; scope=this\|whole)        |
| `DELETE /api/tma/meetings/:id` | Done (organizer-only; scope=this\|whole)        |
| `POST /api/tma/conflicts`      | Done (occurrence-grouped response)              |
```

Delete the slice-A note `Recurring series ... ships in slice B.`

- [ ] **Step 2: Update API.md**

In the **TMA — present** table, change:

- `POST /api/tma/meetings` purpose → `Create a meeting (recurring supported via recurrence_until + recurrence_days)`.
- `PATCH /api/tma/meetings/:id` purpose → `Edit a meeting (organizer-only, 403). Query: scope=this|whole (default this).`
- `DELETE /api/tma/meetings/:id` purpose → `Cancel a meeting (organizer-only, 403). Query: scope=this|whole (default this).`
- `POST /api/tma/conflicts` purpose → `Conflict-warning check (single or expanded series). Response: occurrences[]`.

Append a short note below the table:

> Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

- [ ] **Step 3: Backend full verification**

Run from repo root:
```bash
make test 2>&1 | tail -15
make lint 2>&1 | tail -5
make build 2>&1 | tail -5
```
Expected: all green, `0 issues`.

- [ ] **Step 4: Frontend full verification**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat/frontend && pnpm typecheck && pnpm format && pnpm build && pnpm test -- --run 2>&1 | tail -10
```
Expected: all clean.

- [ ] **Step 5: Commit B-T13**

```bash
git -C /Users/temirlan/Workspace/in-house/lead-cat add docs/MEETINGS.md docs/API.md

git -C /Users/temirlan/Workspace/in-house/lead-cat commit -m "$(cat <<'EOF'
docs(meetings,api): recurring series live (slice B)

- MEETINGS.md: status note + write-paths table reflect non-recurring
  AND recurring shipped end-to-end via /api/tma/*.
- API.md: scope=this|whole on PATCH/DELETE; recurrence params on POST
  /tma/meetings + POST /tma/conflicts; occurrence-grouped conflicts.
- Note recurrence kinds (once/daily/weekly/custom/monthly) with
  recurrence_days / recurrence_until contract.

Verification: make test/lint/build green; pnpm typecheck/format/build
+ vitest clean (incl. write-error tests, recurrence_test, series_conflicts_test).

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Acceptance

After all 13 tasks land on `feat/meetings-recurrence-b`:

1. **Create a weekly meeting**: wizard picks rec=weekly, end-date defaults to +12w, edits to +6w; backend materializes 6 occurrences; one `MeetingCreated` enqueued; Google has 6 events.
2. **Create a custom Mon/Wed/Fri meeting × 4 weeks**: 12 occurrences; conflicts step shows real conflicts grouped by date.
3. **Edit "this only"**: only the picked occurrence row updates; one event patched in Google; `MeetingUpdated` enqueued once.
4. **Edit "whole series"**: wizard opens with date/recurrence/until disabled; all series rows update on confirm; all events patched; `MeetingUpdated` once.
5. **Cancel "this only"** and **cancel "whole series"**: scope routing works; correct number of rows cancelled; events deleted; `MeetingCancelled` once.
6. **403 path**: non-organizer non-admin user gets 403 on PATCH/DELETE for both scopes.
7. **make test / make lint / make build** green; **pnpm -C frontend typecheck/format/build/test** green.

---

## Self-review checklist (run after writing this plan — done by the plan author)

✅ Each spec section is covered by at least one task — verified mapping:

| Spec section | Tasks |
|---|---|
| Domain enum changes | B-T1 |
| Domain Occurrences extension | B-T1 |
| Migration + persistence | B-T2 |
| UpdateWholeSeries / CancelWholeSeries | B-T3 |
| MeetingSeriesConflicts | B-T4 |
| HTTP create recurrence body | B-T5 |
| HTTP scope on PATCH/DELETE | B-T6 |
| HTTP conflicts series response | B-T7 |
| OpenAPI + regen | B-T8 |
| Frontend types/mutations grow | B-T9 |
| Wizard until + smart defaults | B-T10 |
| Wizard review grouped conflicts | B-T11 |
| Meeting detail dual-scope | B-T12 |
| Docs + verify | B-T13 |

✅ No placeholders ("TBD", "implement later") — verified.
✅ Type/signature consistency — `OccurrenceConflicts` (frontend) ↔ `tmaOccurrenceConflictsDTO` (backend) ↔ `TmaOccurrenceConflicts` (OpenAPI). `Custom` constant used consistently. `scope: "this" | "whole"` consistent.
✅ Scope: one branch, ~13 tasks, ~1.5 weeks — within single-plan size.
