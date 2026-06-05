# §4.4.2 series editing ("this / whole series") Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When editing a meeting that belongs to a series, let the organizer choose "this occurrence" (existing flow) or "whole series (this and following)" — applying field + time-of-day changes to all future occurrences.

**Architecture:** A pure `applySeriesUpdate` builds each occurrence's new row; `Services.UpdateSeries` loads future occurrences, applies it, persists in one transaction, patches Google best-effort, and notifies once. The `meetingedit` FSM gains a scope screen after picking a series meeting and a series-flavored menu (time-of-day only, no recurrence).

**Tech Stack:** Go, pgx (tx), go-telegram/bot, google/uuid, asynq.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-series-edit-design.md`

**Conventions:** Run Go from `backend/` with `env -u GOROOT`. Module `github.com/Jaryq-Lab/notify-bot`. Build check: `env -u GOROOT go build ./...`.

---

## Task 1: application — `applySeriesUpdate` + `SeriesUpdateInput`

**Files:**

- Create: `backend/internal/application/series_edit.go`
- Test: `backend/internal/application/series_edit_test.go`

Context: `postgres.Meeting` has `Dept/Type/Host/Description/Recurrence string`, `StartsAt/EndsAt time.Time`, `Name string`. `meeting.Input{...}.Validate()`, `meeting.GenerateName(dept,type,host string, date time.Time, r meeting.Recurrence) string`, `meeting.Recurrence`. `ErrInvalidInput` and the `orStr(*string, string) string` helper already exist in the `application` package (participants.go).

- [ ] **Step 1: Write the failing test.** Create `backend/internal/application/series_edit_test.go`:

```go
package application

import (
	"errors"
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func occ() postgres.Meeting {
	loc, _ := time.LoadLocation("Asia/Almaty")
	return postgres.Meeting{
		Dept: "Разработка", Type: "Планёрка", Host: "Иванов",
		StartsAt:   time.Date(2026, 6, 8, 14, 0, 0, 0, loc).UTC(),
		EndsAt:     time.Date(2026, 6, 8, 15, 0, 0, 0, loc).UTC(),
		Recurrence: "weekly", Description: "d", Name: "old",
	}
}

func TestApplySeriesUpdate_FieldOnly(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applySeriesUpdate(occ(), SeriesUpdateInput{Dept: strp("Маркетинг")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Dept != "Маркетинг" {
		t.Fatalf("dept=%q", out.Dept)
	}
	// name recomputed; weekly label preserved; date is the occurrence's own (08).
	if out.Name != "Маркетинг | Планёрка | Иванов | 2026-06-08 | Еженедельно" {
		t.Fatalf("name=%q", out.Name)
	}
	if !out.StartsAt.Equal(occ().StartsAt) {
		t.Fatal("times must be unchanged when no time override")
	}
}

func TestApplySeriesUpdate_TimeOfDayKeepsDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applySeriesUpdate(occ(), SeriesUpdateInput{Start: strp("10:00"), End: strp("11:00")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 6, 8, 10, 0, 0, 0, loc).UTC() // same date, new time
	if !out.StartsAt.Equal(wantStart) {
		t.Fatalf("start=%v want %v", out.StartsAt, wantStart)
	}
	if !out.EndsAt.Equal(time.Date(2026, 6, 8, 11, 0, 0, 0, loc).UTC()) {
		t.Fatalf("end=%v", out.EndsAt)
	}
}

func TestApplySeriesUpdate_BadTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := applySeriesUpdate(occ(), SeriesUpdateInput{Start: strp("15:00"), End: strp("14:00")}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}
```

(`strp` is already defined in `meeting_update_test.go` in this package.)

- [ ] **Step 2: Run, verify fail.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestApplySeriesUpdate -v` → FAIL (undefined applySeriesUpdate / SeriesUpdateInput).

- [ ] **Step 3: Implement.** Create `backend/internal/application/series_edit.go`:

```go
package application

import (
	"fmt"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// SeriesUpdateInput carries series-wide field overrides (nil = unchanged) plus an
// optional time-of-day change (Start/End HH:MM applied to each occurrence's own
// date). Date and recurrence pattern are not changed series-wide.
type SeriesUpdateInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Description *string
	Start       *string // HH:MM
	End         *string // HH:MM
}

// applySeriesUpdate applies field overrides + an optional time-of-day to one
// occurrence, keeping the occurrence's own date and recurrence. Pure; recomputes
// the name. Returns ErrInvalidInput on bad time.
func applySeriesUpdate(cur postgres.Meeting, in SeriesUpdateInput, loc *time.Location) (postgres.Meeting, error) {
	dept := orStr(in.Dept, cur.Dept)
	typ := orStr(in.Type, cur.Type)
	host := orStr(in.Host, cur.Host)
	desc := orStr(in.Description, cur.Description)

	startLocal := cur.StartsAt.In(loc)
	startsAt := cur.StartsAt
	endsAt := cur.EndsAt
	if in.Start != nil && in.End != nil {
		day := cur.StartsAt.In(loc).Format("2006-01-02")
		s, err := time.ParseInLocation("2006-01-02 15:04", day+" "+*in.Start, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", day+" "+*in.End, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	rec := meeting.Recurrence(cur.Recurrence)
	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}
```

- [ ] **Step 4: Run, verify pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestApplySeriesUpdate -v` → 3 PASS.

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/series_edit.go backend/internal/application/series_edit_test.go && git commit -m "feat(meetings): applySeriesUpdate pure helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: repo — `ListSeriesOccurrences` + transactional `UpdateMeetingsTx`

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Extract a shared update statement + add the queries.** In `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`:

(a) Add a shared update SQL + args helper (place near `insertMeetingSQL`), and rewrite `UpdateMeeting` to use them:

```go
const updateMeetingSQL = `
	UPDATE meetings SET dept=$3, type=$4, host=$5, starts_at=$6, ends_at=$7,
		recurrence=$8, name=$9, description=$10, updated_at=now()
	WHERE id=$1 AND workspace_id=$2 AND status='scheduled'`

// updateMeetingArgs returns the args for updateMeetingSQL; order MUST match its $1..$10.
func updateMeetingArgs(workspaceID, id uuid.UUID, m Meeting) []any {
	return []any{id, workspaceID, m.Dept, m.Type, m.Host, m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description}
}
```

Replace the existing `UpdateMeeting` body with:

```go
func (s *Store) UpdateMeeting(ctx context.Context, workspaceID, id uuid.UUID, m Meeting) error {
	ct, err := s.pool.Exec(ctx, updateMeetingSQL, updateMeetingArgs(workspaceID, id, m)...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrMeetingNotEditable
	}
	return nil
}
```

(b) Add the transactional batch update (after `UpdateMeeting`):

```go
// UpdateMeetingsTx updates all meetings in one transaction (all-or-nothing). Each
// must still be scheduled in the workspace, else ErrMeetingNotEditable rolls back.
func (s *Store) UpdateMeetingsTx(ctx context.Context, workspaceID uuid.UUID, ms []Meeting) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, m := range ms {
		ct, err := tx.Exec(ctx, updateMeetingSQL, updateMeetingArgs(workspaceID, m.ID, m)...)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrMeetingNotEditable
		}
	}
	return tx.Commit(ctx)
}
```

(c) Add the series occurrence query (after `ListScheduleForEmail` or near the other List\* methods):

```go
// ListSeriesOccurrences returns the scheduled occurrences of a series at or after
// fromStart, in the workspace, ordered by start.
func (s *Store) ListSeriesOccurrences(ctx context.Context, workspaceID, seriesID uuid.UUID, fromStart time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE series_id = $1 AND workspace_id = $2 AND starts_at >= $3 AND status = 'scheduled'
		ORDER BY starts_at`, seriesID, workspaceID, fromStart)
}
```

- [ ] **Step 2: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
      Expected: clean.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go && git commit -m "feat(meetings): ListSeriesOccurrences + transactional UpdateMeetingsTx

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: application — `Services.UpdateSeries`

**Files:**

- Modify: `backend/internal/application/series_edit.go`

Context: `Services{Store, Cipher, Queue, Calendar, Log}`. `ownerOrOrganizer(w postgres.Workspace, organizerUserID *uuid.UUID, userID uuid.UUID) bool` and `orDefault(v, def string) string` exist in the package. `s.Calendar.For(ctx, ws) (CalendarService, error)`, `CalendarService.UpdateEvent(ctx, eventID, CalendarEvent) error`, `s.Queue.EnqueueMeetingUpdated(ctx, ws, id) error`.

- [ ] **Step 1: Implement `UpdateSeries`.** Append to `backend/internal/application/series_edit.go` (add imports `context`, `github.com/google/uuid`, `go.uber.org/zap` to the file's import block):

```go
// UpdateSeries applies a series-wide edit to the picked occurrence and all later
// ones (organizer or owner only): validates per occurrence, persists atomically,
// patches Google best-effort, and enqueues one change notification. Returns the
// number of occurrences updated.
func (s *Services) UpdateSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
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
	occs, err := s.Store.ListSeriesOccurrences(ctx, workspaceID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	rows := make([]postgres.Meeting, 0, len(occs))
	for _, occ := range occs {
		upd, err := applySeriesUpdate(occ, in, loc)
		if err != nil {
			return 0, err
		}
		rows = append(rows, upd)
	}
	if err := s.Store.UpdateMeetingsTx(ctx, workspaceID, rows); err != nil {
		return 0, err
	}
	// Google best-effort: DB is the source of truth; a failed patch is reconciled
	// by re-editing (series patches are not reversible, so we don't compensate).
	if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("series update calendar provider", zap.String("workspace_id", workspaceID.String()), zap.Error(ferr))
		}
	} else {
		for _, m := range rows {
			if m.GoogleEventID == "" {
				continue
			}
			if err := calSvc.UpdateEvent(ctx, m.GoogleEventID, CalendarEvent{
				Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
			}); err != nil && s.Log != nil {
				s.Log.Warn("series update event", zap.String("event_id", m.GoogleEventID), zap.Error(err))
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

- [ ] **Step 2: Build + vet + test.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/ && env -u GOROOT go test ./internal/application/`
      Expected: build OK; `TestApplySeriesUpdate*` PASS. (The orchestration's DB/Google path is build-verified, per the `CreateMeeting`-series convention.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/series_edit.go && git commit -m "feat(meetings): Services.UpdateSeries (this-and-following)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: `meetingedit` — scope screen + series menu

**Files:**

- Modify: `backend/internal/platform/meetingedit/state.go`
- Modify: `backend/internal/platform/meetingedit/parse.go`
- Modify: `backend/internal/platform/meetingedit/service.go`
- Modify: `backend/internal/platform/meetingedit/parse_test.go`
- Modify: `backend/internal/platform/meetingedit/service_test.go`

- [ ] **Step 1: State fields.** In `state.go`, add to the `State` struct (after `PendingRemove`):

```go
	SeriesID string `json:"series_id,omitempty"` // set if the picked meeting is part of a series
	Scope    string `json:"scope,omitempty"`     // one | series
```

- [ ] **Step 2: `parseTimeRange` (TDD).** In `parse_test.go`, add:

```go
func TestParseTimeRange(t *testing.T) {
	for _, in := range []string{"14:00–15:00", "14:00-15:00"} {
		s, e, err := parseTimeRange(in)
		if err != nil || s != "14:00" || e != "15:00" {
			t.Fatalf("%q -> %q %q err %v", in, s, e, err)
		}
	}
	for _, bad := range []string{"", "14:00", "9:00-10:00", "15:00-14:00", "14:00-14:00"} {
		if _, _, err := parseTimeRange(bad); err == nil {
			t.Fatalf("%q: expected error", bad)
		}
	}
}
```

Run `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -run TestParseTimeRange -v` → FAIL. Then add to `parse.go`:

```go
// parseTimeRange parses "HH:MM–HH:MM" (en dash or hyphen) into start/end, requiring
// strict HH:MM and end > start.
func parseTimeRange(text string) (start, end string, err error) {
	rng := strings.NewReplacer("–", "-", "—", "-").Replace(strings.TrimSpace(text))
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("формат: ЧЧ:ММ–ЧЧ:ММ")
	}
	a := strings.TrimSpace(parts[0])
	b := strings.TrimSpace(parts[1])
	st, err := time.Parse("15:04", a)
	if err != nil || len(a) != 5 {
		return "", "", fmt.Errorf("неверное время начала (ЧЧ:ММ)")
	}
	en, err := time.Parse("15:04", b)
	if err != nil || len(b) != 5 {
		return "", "", fmt.Errorf("неверное время конца (ЧЧ:ММ)")
	}
	if !en.After(st) {
		return "", "", fmt.Errorf("конец должен быть позже начала")
	}
	return st.Format("15:04"), en.Format("15:04"), nil
}
```

(`parse.go` already imports `fmt`, `strings`, `time`.) Run the test → PASS.

- [ ] **Step 3: Scope-aware pick + scope screen + Backend method.** In `service.go`:

(a) Extend the `Backend` interface — add:

```go
	UpdateSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in application.SeriesUpdateInput) (int, error)
```

(b) Replace `pick`'s tail (the `st := State{...}; _ = s.sessions.Set(...); return menuReply(st, true)` part) so a series meeting shows the scope screen:

```go
	loc := loadLoc(found.TZ)
	st := State{
		Step:        stepMenu,
		MeetingID:   mid.String(),
		WorkspaceID: found.WorkspaceID.String(),
		UserID:      found.OrganizerUserID.String(),
		Cur:         snapshot(found.Meeting, loc),
		Overrides:   map[string]string{},
	}
	if found.SeriesID != nil {
		st.SeriesID = found.SeriesID.String()
		_ = s.sessions.Set(ctx, telegramID, st)
		return scopeReply()
	}
	st.Scope = "one"
	_ = s.sessions.Set(ctx, telegramID, st)
	return menuReply(st, true)
```

(c) Add the scope screen + the two scope callbacks. Add `scopeReply` helper and handlers:

```go
func scopeReply() Reply {
	return Reply{
		Text: "Что редактируем?",
		Keyboard: [][]Button{
			{{Text: "✏️ Эту встречу", Data: "medit:scope:one"}},
			{{Text: "🔁 Всю серию (эту и далее)", Data: "medit:scope:series"}},
		},
	}
}

func (s *Service) setScope(ctx context.Context, telegramID int64, scope string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	st.Scope = scope
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true)
}
```

(d) Route them in `OnCallback` (add cases before the final `return Reply{}, false`):

```go
	case data == "medit:scope:one":
		return s.setScope(ctx, telegramID, "one"), true
	case data == "medit:scope:series":
		return s.setScope(ctx, telegramID, "series"), true
```

- [ ] **Step 4: Scope-aware menu, field prompt, OnText, apply.** In `service.go`:

(a) Make the keyboard + summary scope-aware. Change `menuReply` and `menuKeyboard`/`menuText`:

```go
func menuReply(st State, edit bool) Reply {
	return Reply{Text: menuText(st), Keyboard: menuKeyboard(st.Scope), Edit: edit}
}

func menuKeyboard(scope string) [][]Button {
	if scope == "series" {
		return [][]Button{
			{{Text: "🕒 Время", Data: "medit:field:datetime"}},
			{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
			{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
			{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
		}
	}
	return [][]Button{
		{{Text: "📅 Дата/время", Data: "medit:field:datetime"}},
		{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
		{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
		{{Text: "🔁 Частота", Data: "medit:field:rec"}},
		{{Text: "👥 Участники", Data: "medit:parts"}},
		{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
	}
}
```

And in `menuText`, branch the header + the date/time line + recurrence line on `st.Scope`. Replace the body of `menuText` with:

```go
func menuText(st State) string {
	eff := func(k string) string {
		if v, ok := st.Overrides[k]; ok {
			return v
		}
		return st.Cur[k]
	}
	mark := func(k string) string {
		if _, ok := st.Overrides[k]; ok {
			return " ★"
		}
		return ""
	}
	var b strings.Builder
	if st.Scope == "series" {
		fmt.Fprintf(&b, "Редактирование всей серии с %s (★ — изменено):\n", st.Cur["date"])
		tmark := ""
		if _, ok := st.Overrides["start"]; ok {
			tmark = " ★"
		}
		fmt.Fprintf(&b, "• Время: %s–%s%s\n", eff("start"), eff("end"), tmark)
		fmt.Fprintf(&b, "• Отдел: %s%s\n", eff("dept"), mark("dept"))
		fmt.Fprintf(&b, "• Тип: %s%s\n", eff("type"), mark("type"))
		fmt.Fprintf(&b, "• Ведущий: %s%s\n", eff("host"), mark("host"))
		fmt.Fprintf(&b, "• Описание: %s%s\n", eff("description"), mark("description"))
		return b.String()
	}
	b.WriteString("Редактирование встречи (★ — изменено):\n")
	dmark := ""
	if _, ok := st.Overrides["date"]; ok {
		dmark = " ★"
	}
	fmt.Fprintf(&b, "• Дата/время: %s %s–%s%s\n", eff("date"), eff("start"), eff("end"), dmark)
	fmt.Fprintf(&b, "• Отдел: %s%s\n", eff("dept"), mark("dept"))
	fmt.Fprintf(&b, "• Тип: %s%s\n", eff("type"), mark("type"))
	fmt.Fprintf(&b, "• Ведущий: %s%s\n", eff("host"), mark("host"))
	fmt.Fprintf(&b, "• Описание: %s%s\n", eff("description"), mark("description"))
	fmt.Fprintf(&b, "• Частота: %s%s\n", recLabel(eff("recurrence")), mark("recurrence"))
	return b.String()
}
```

(b) Scope-aware datetime prompt in `field()`. Replace the `prompt, ok := fieldPrompts[f]` lookup with a special-case for the series time field:

```go
	prompt, ok := fieldPrompts[f]
	if !ok {
		return Reply{}
	}
	if f == "datetime" && st.Scope == "series" {
		prompt = "Введи новое время ЧЧ:ММ–ЧЧ:ММ (например: 15:00–16:00):"
	}
	st.Step = stepAwaiting
	st.AwaitingField = f
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: prompt}
```

(c) Scope-aware datetime in `OnText`. Leave the existing `if st.AwaitingField == "participant" { return s.searchParticipant(...) }` line (above) untouched. Replace ONLY the existing `if st.AwaitingField == "datetime" { ... } else { ... }` block with this scope-aware version (the trailing `else` for plain fields is preserved exactly):

```go
	if st.AwaitingField == "datetime" {
		if st.Scope == "series" {
			start, end, perr := parseTimeRange(text)
			if perr != nil {
				return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
			}
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		} else {
			d, start, end, perr := parseDateTime(text)
			if perr != nil {
				return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
			}
			st.Overrides["date"] = d
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		}
	} else {
		if text == "" {
			return Reply{Text: "Пусто. Введи значение:"}, true
		}
		st.Overrides[st.AwaitingField] = text
	}
```

(The `st.Step = stepMenu; st.AwaitingField = ""; sessions.Set; return menuReply(*st, false), true` tail after this block stays unchanged.)

(d) Scope-aware `apply`. In `apply`, after the `len(st.Overrides) == 0` guard and the `ws/uid/mid` parse, branch:

```go
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)

	if st.Scope == "series" {
		n, err := s.backend.UpdateSeries(ctx, ws, uid, mid, seriesInput(st.Overrides))
		if err != nil {
			switch {
			case errors.Is(err, application.ErrInvalidInput):
				return Reply{Text: "Неверные данные. Поправь поле и попробуй снова."}
			case errors.Is(err, application.ErrForbidden):
				_ = s.sessions.Del(ctx, telegramID)
				return Reply{Text: "Нет доступа к этой встрече."}
			default:
				return Reply{Text: "Не удалось обновить серию, попробуй позже."}
			}
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: fmt.Sprintf("Готово ✏️ — обновлено встреч серии: %d", n)}
	}

	m, err := s.backend.UpdateMeeting(ctx, ws, uid, mid, toInput(st.Overrides))
```

(the rest of the existing `UpdateMeeting` error switch + success path stays unchanged below this).

(e) Add `seriesInput` (near `toInput`):

```go
func seriesInput(ov map[string]string) application.SeriesUpdateInput {
	var in application.SeriesUpdateInput
	set := func(p **string, k string) {
		if v, ok := ov[k]; ok {
			vv := v
			*p = &vv
		}
	}
	set(&in.Dept, "dept")
	set(&in.Type, "type")
	set(&in.Host, "host")
	set(&in.Description, "description")
	set(&in.Start, "start")
	set(&in.End, "end")
	return in
}
```

- [ ] **Step 5: Tests — scope flow.** In `service_test.go`, extend `fakeBackend` with `UpdateSeries` (record call) and add a sample series meeting + flow tests. Add the field + method to `fakeBackend`:

```go
	// add to fakeBackend struct:
	//   updatedSeries int
	//   seriesIn      application.SeriesUpdateInput

func (f *fakeBackend) UpdateSeries(_ context.Context, _, _, _ uuid.UUID, in application.SeriesUpdateInput) (int, error) {
	f.seriesIn = in
	f.updatedSeries++
	return 3, nil
}
```

Add a series sample + tests:

```go
func seriesMeeting() postgres.MeetingWithTZ {
	m := sampleMeeting()
	sid := uuid.New()
	m.SeriesID = &sid
	m.Recurrence = "weekly"
	return m
}

func TestEditFlow_SeriesScopePrompt(t *testing.T) {
	ctx := context.Background()
	m := seriesMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(80)
	r, ok := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	if !ok || !strings.Contains(r.Text, "Что редактируем") {
		t.Fatalf("series pick should show scope prompt, got %+v", r)
	}
}

func TestEditFlow_SeriesEdit(t *testing.T) {
	ctx := context.Background()
	m := seriesMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(81)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	// choose whole series → menu must omit "Частота"
	r, _ := svc.OnCallback(ctx, tg, "medit:scope:series")
	for _, row := range r.Keyboard {
		for _, b := range row {
			if b.Data == "medit:field:rec" {
				t.Fatal("series menu must not offer recurrence")
			}
		}
	}
	// time-of-day edit
	svc.OnCallback(ctx, tg, "medit:field:datetime")
	if r, ok := svc.OnText(ctx, tg, "10:00-11:00"); !ok || !strings.Contains(r.Text, "★") {
		t.Fatalf("time input: %+v ok=%v", r, ok)
	}
	svc.OnCallback(ctx, tg, "medit:apply")
	if be.updatedSeries != 1 || be.seriesIn.Start == nil || *be.seriesIn.Start != "10:00" {
		t.Fatalf("UpdateSeries not called with time: %+v (called=%d)", be.seriesIn, be.updatedSeries)
	}
}

func TestEditFlow_NonSeriesNoPrompt(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting() // no SeriesID
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(82)
	r, _ := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	if !strings.Contains(r.Text, "Редактирование встречи") {
		t.Fatalf("non-series pick should go straight to field menu, got %+v", r)
	}
}
```

(`sampleMeeting` returns a `postgres.MeetingWithTZ` with no `SeriesID` — confirm; if its struct literal would need `SeriesID` left nil, that's the default.)

- [ ] **Step 6: Run tests + build.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v && env -u GOROOT go build ./...`
      Expected: all PASS (existing + parseTimeRange + 3 scope tests), build OK. (`*application.Services` now satisfies the extended `Backend` because `UpdateSeries` was added in Task 3.)

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingedit/ && git commit -m "feat(meetings): series edit scope (this/whole) in /edit FSM

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: full verification + docs

**Files:**

- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Run the full suite from the repo root.** `cd /Users/temirlan/Workspace/in-house/lead-cat && make test && make lint && make build`. `make lint` runs golangci-lint (incl. gofmt). If gofmt issues appear, `gofmt -w` the listed backend files and re-run `make lint`. If a real failure occurs, STOP and report BLOCKED with output.

- [ ] **Step 2: Document.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Recurring-series creation (done)" line, add:

```markdown
> **Recurring-series editing (§4.4.2, done):** when editing a meeting that belongs to a series, `/edit` asks "эту встречу / всю серию (эту и далее)". "Whole series" (`UpdateSeries`) applies dept/type/host/description and a time-of-day change (HH:MM kept on each occurrence's own date) to the picked occurrence and all later scheduled ones — one DB transaction, Google events patched best-effort, one `meeting:updated` DM. The recurrence pattern and occurrence dates are unchanged (re-materialization out of scope). Editing "this occurrence" is the existing single-meeting flow.
```

- [ ] **Step 3: Commit (do NOT add frontend/vite.config.ts).**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §4.4.2 series editing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** `applySeriesUpdate` field+time-of-day (Task 1) · `ListSeriesOccurrences` + `UpdateMeetingsTx` (Task 2) · `UpdateSeries` orchestration (ACL, future-from-picked, tx, Google best-effort, enqueue-once) (Task 3) · FSM scope screen + series menu + `parseTimeRange` + scope-aware field/OnText/apply (Task 4) · testing (Tasks 1,3,4) · docs (Task 5). Out-of-scope (recurrence-pattern change, series delete, past occurrences) recorded in spec + Task 5 note. All covered.
- **Type consistency:** `SeriesUpdateInput{Dept,Type,Host,Description,Start,End *string}` (Task 1) used by `applySeriesUpdate`/`UpdateSeries` (Tasks 1,3), `meetingedit.Backend.UpdateSeries` + `seriesInput` (Task 4). `UpdateMeetingsTx(ws, []Meeting)` + `ListSeriesOccurrences(ws, seriesID, fromStart)` (Task 2) called by `UpdateSeries` (Task 3). `parseTimeRange` (Task 4) used in OnText. `State.SeriesID/Scope` (Task 4) drive pick/menu/field/OnText/apply. `updateMeetingSQL`/`updateMeetingArgs` (Task 2) shared by `UpdateMeeting` + `UpdateMeetingsTx`.
- **No placeholders:** every code/command step is concrete. The OnText edit (Task 4 Step 4c) explicitly preserves the existing `participant` branch and only rewrites the `datetime` block.

```

```
