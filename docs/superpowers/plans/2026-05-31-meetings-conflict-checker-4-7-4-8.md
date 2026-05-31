# §4.7–4.8 Conflict Warning + Free-Time Checker — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a time-conflict warning on meeting edit (§4.7) and a common-free-time checker (§4.8) to the meetings feature, exposed via both the Telegram bot and REST.

**Architecture:** A pure interval core in `domain/meeting` (`Overlaps`, `FreeSlots`); one new global-by-email repo query (`ListMeetingsOverlapping`); application read methods `MeetingConflicts` (low-level: emails+span), `MeetingUpdateConflicts` (resolves a pending edit's effective time + participants + organizer, then delegates), and `FreeSlots`. Surfaces: the `/edit` FSM warning, a new `/checker` FSM, and two REST endpoints. Internal DB is the busyness source; queries are global-by-email with a hardcoded `Asia/Almaty` location for the free-slot windows, mirroring the shipped §4.6 `/schedule`.

**Tech Stack:** Go, Fiber, go-telegram/bot, pgx/Postgres, Redis-backed FSM sessions, zap.

**Spec:** `docs/superpowers/specs/2026-05-31-meetings-conflict-checker-4-7-4-8-design.md`

## Codebase facts (verified — rely on these, but confirm before editing)

- **Module path:** `github.com/Jaryq-Lab/notify-bot` (NOT `lead-cat/backend`). Every import is `github.com/Jaryq-Lab/notify-bot/internal/...`.
- **`Services` struct** (`internal/application/services.go`): fields `Store *postgres.Store`, `Calendar CalendarProvider`, `Queue Queuer`, `Log *zap.Logger`. Methods use receiver `s` and `s.Store`.
- **`queryMeetings`** (`postgres/meeting_repo.go:148`) scans meeting rows only — it does **NOT** hydrate `Meeting.Participants`. To get participant emails for a meeting, call `s.Store.ListParticipants(ctx, meetingID)`. Organizer email: `s.Store.GetUserByID(ctx, *m.OrganizerUserID)` → `PlatformUser.Email`.
- **`ListScheduleForEmail`** (`meeting_repo.go:241`) is the join shape to copy (participant OR organizer, global by email, `status='scheduled'`).
- **`orDefault(a, b string) string`** exists at `meeting_service.go:200` (workspace TZ fallback to `Asia/Almaty`). Reuse it.
- **Models** (`postgres/models.go`): `Meeting{ID *uuid.UUID, OrganizerUserID *uuid.UUID, StartsAt, EndsAt time.Time, Name string, ...}`; `MeetingParticipant{Email string, ...}`; `Employee{FullName, Email string, ...}`; `PlatformUser{Email string, ...}`.
- **Bot FSM convention** (see `internal/platform/scheduleview/` and `meetingedit/`): each package defines its own `State`, `Button{Text, Data string}`, `Reply{Text string, Keyboard [][]Button, Edit bool}` (all in `state.go`), a `sessions` interface `Get/Set/Del` (where `Set` takes `State` by value), `New(backend, sess)`, `NewRedisSessions(rdb)` (in `redis_sessions.go`), and `Start` returns `Reply` while `OnText`/`OnCallback` return `(Reply, bool)` (bool = handled). There is **no** `keyboard.go` — keyboards are inline funcs in `service.go`.
- **Dispatcher** (`internal/infrastructure/telegram/multitenant.go`): `botBackend` interface embeds `meetingedit.Backend` + `scheduleview.Backend`; each FSM gets a field on `MultiHandler`, a `sendXxxReply` + `toXxxMarkup` helper, a command `case` in `Handle`, an `OnText` call in the no-command private branch, and a prefix `case` in `handleCallback`. `/schedule` guards with `GetBotUserByTelegramID`.
- **REST** (`internal/delivery/http/handlers/`): handler receiver is `*API` with field **`App`** (`a.App`); workspace id via `c.Locals("workspace_id").(uuid.UUID)`; body via `c.BodyParser`; errors via `fiber.NewError`. Routes in `internal/delivery/http/app.go` under the `ws` group (lines ~132–136).

## Conventions

- Run all checks from the **repo root**: `make test && make lint && make build`. Run Go directly as `env -u GOROOT go ...` from `backend/`. **`make lint` includes a gofmt check** — always run it before committing; per-task `go test`/`go vet` will not catch gofmt drift. If lint flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/...`.
- Backend test convention: **pure logic is unit-tested; I/O paths (repo, REST handlers, bot wiring) are build-verified only** (no DB harness in the postgres package).
- Do **not** touch `frontend/vite.config.ts` (long-standing local-only edit).
- All times stored/compared in UTC; user-facing rendering uses `Asia/Almaty`.

## File structure (created/modified)

- Create `backend/internal/domain/meeting/conflict.go` + `conflict_test.go` — `Overlaps`, `FreeSlots` (pure).
- Modify `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` — `ListMeetingsOverlapping`.
- Create `backend/internal/application/conflict.go` — `Conflict`, `FreeSlot`, constants, `MeetingConflicts`, `MeetingUpdateConflicts`, `FreeSlots`, `personName`.
- Modify `backend/internal/platform/meetingedit/service.go` + `state.go` — §4.7 warning (extract `doApply`, add `applyForce`, conflict check).
- Create `backend/internal/platform/checker/{state,service,parse,redis_sessions}.go` + `parse_test.go` — `/checker` FSM.
- Modify `backend/internal/infrastructure/telegram/multitenant.go` — wire `/checker`.
- Create `backend/internal/delivery/http/handlers/meeting_availability.go` + modify `app.go` — REST.
- Modify `docs/MEETINGS.md`, `PLAN.md` — status.

---

## Task 1: Pure interval core (`Overlaps`, `FreeSlots`)

**Files:**
- Create: `backend/internal/domain/meeting/conflict.go`
- Test: `backend/internal/domain/meeting/conflict_test.go`

Pure logic — full TDD. `Span{Start, End time.Time}` already exists in `recurrence.go` (same package).

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/domain/meeting/conflict_test.go`:

```go
package meeting

import (
	"testing"
	"time"
)

func tm(h, m int) time.Time { return time.Date(2026, 6, 1, h, m, 0, 0, time.UTC) }

func TestOverlaps(t *testing.T) {
	cases := []struct {
		name           string
		aS, aE, bS, bE time.Time
		want           bool
	}{
		{"disjoint before", tm(9, 0), tm(10, 0), tm(10, 0), tm(11, 0), false}, // touching edge = no overlap
		{"disjoint after", tm(11, 0), tm(12, 0), tm(9, 0), tm(10, 0), false},
		{"partial", tm(9, 0), tm(10, 0), tm(9, 30), tm(11, 0), true},
		{"full contain", tm(9, 0), tm(12, 0), tm(10, 0), tm(11, 0), true},
		{"identical", tm(9, 0), tm(10, 0), tm(9, 0), tm(10, 0), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Overlaps(c.aS, c.aE, c.bS, c.bE); got != c.want {
				t.Fatalf("Overlaps=%v want %v", got, c.want)
			}
		})
	}
}

func spansEqual(a, b []Span) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Start.Equal(b[i].Start) || !a[i].End.Equal(b[i].End) {
			return false
		}
	}
	return true
}

func TestFreeSlots(t *testing.T) {
	win, winEnd := tm(9, 0), tm(18, 0)
	min := 30 * time.Minute

	t.Run("no busy = whole window", func(t *testing.T) {
		if got := FreeSlots(nil, win, winEnd, min); !spansEqual(got, []Span{{win, winEnd}}) {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("busy spanning window = none", func(t *testing.T) {
		if got := FreeSlots([]Span{{tm(8, 0), tm(19, 0)}}, win, winEnd, min); len(got) != 0 {
			t.Fatalf("got %v want empty", got)
		}
	})
	t.Run("gaps around one meeting", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(11, 0), tm(12, 30)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(11, 0)}, {tm(12, 30), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
	t.Run("merge overlapping busy", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(11, 0), tm(12, 0)}, {tm(11, 30), tm(13, 0)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(11, 0)}, {tm(13, 0), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
	t.Run("min-duration filter drops short gaps", func(t *testing.T) {
		// leaves 9:00-9:15 and 17:45-18:00, both < 30m
		if got := FreeSlots([]Span{{tm(9, 15), tm(17, 45)}}, win, winEnd, min); len(got) != 0 {
			t.Fatalf("got %v want empty", got)
		}
	})
	t.Run("busy outside window clipped", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(6, 0), tm(9, 30)}, {tm(18, 30), tm(20, 0)}}, win, winEnd, min)
		if !spansEqual(got, []Span{{tm(9, 30), tm(18, 0)}}) {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("unsorted busy input", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(15, 0), tm(16, 0)}, {tm(10, 0), tm(11, 0)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(10, 0)}, {tm(11, 0), tm(15, 0)}, {tm(16, 0), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run 'TestOverlaps|TestFreeSlots' -v`
Expected: FAIL — `undefined: Overlaps` / `undefined: FreeSlots`.

- [ ] **Step 3: Implement the core**

Create `backend/internal/domain/meeting/conflict.go`:

```go
package meeting

import (
	"sort"
	"time"
)

// Overlaps reports whether spans [aStart,aEnd) and [bStart,bEnd) intersect.
// Touching edges (aEnd == bStart) do NOT overlap. §4.7.1
func Overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// FreeSlots returns the gaps in [winStart,winEnd) not covered by busy, keeping only
// gaps with duration >= minDur. busy spans are clipped to the window and merged;
// input need not be sorted. Result is chronological. §4.8.3
func FreeSlots(busy []Span, winStart, winEnd time.Time, minDur time.Duration) []Span {
	clipped := make([]Span, 0, len(busy))
	for _, b := range busy {
		s, e := b.Start, b.End
		if s.Before(winStart) {
			s = winStart
		}
		if e.After(winEnd) {
			e = winEnd
		}
		if s.Before(e) {
			clipped = append(clipped, Span{Start: s, End: e})
		}
	}
	sort.Slice(clipped, func(i, j int) bool { return clipped[i].Start.Before(clipped[j].Start) })

	merged := make([]Span, 0, len(clipped))
	for _, b := range clipped {
		if n := len(merged); n > 0 && !b.Start.After(merged[n-1].End) {
			if b.End.After(merged[n-1].End) {
				merged[n-1].End = b.End
			}
			continue
		}
		merged = append(merged, b)
	}

	var free []Span
	cursor := winStart
	add := func(s, e time.Time) {
		if e.Sub(s) >= minDur {
			free = append(free, Span{Start: s, End: e})
		}
	}
	for _, b := range merged {
		if cursor.Before(b.Start) {
			add(cursor, b.Start)
		}
		if b.End.After(cursor) {
			cursor = b.End
		}
	}
	if cursor.Before(winEnd) {
		add(cursor, winEnd)
	}
	return free
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run 'TestOverlaps|TestFreeSlots' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/meeting/conflict.go backend/internal/domain/meeting/conflict_test.go
git commit -m "feat(meetings): pure interval core (Overlaps, FreeSlots) §4.7-4.8"
```

---

## Task 2: Repo query `ListMeetingsOverlapping`

**Files:**
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

Build-verified. Read `ListScheduleForEmail` (line ~241) first to match `meetingColsM` and `queryMeetings` usage exactly.

- [ ] **Step 1: Add the query**

Append after `ListScheduleForEmail`:

```go
// ListMeetingsOverlapping returns scheduled meetings overlapping [from,to) where any
// of emails is a participant or the organizer (by platform_users.email). Global by
// email (no workspace scope), mirroring ListScheduleForEmail. Participants are NOT
// hydrated (use ListParticipants for attribution). §4.7/§4.8
func (s *Store) ListMeetingsOverlapping(ctx context.Context, emails []string, from, to time.Time) ([]Meeting, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	return s.queryMeetings(ctx, `
		SELECT DISTINCT `+meetingColsM+`
		FROM meetings m
		LEFT JOIN meeting_participants mp ON mp.meeting_id = m.id
		LEFT JOIN platform_users pu ON pu.id = m.organizer_user_id
		WHERE (mp.email = ANY($1) OR pu.email = ANY($1))
			AND m.status = 'scheduled'
			AND m.starts_at < $3 AND m.ends_at > $2
		ORDER BY m.starts_at`, emails, from, to)
}
```

- [ ] **Step 2: Build-verify**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go
git commit -m "feat(meetings): ListMeetingsOverlapping repo query §4.7-4.8"
```

---

## Task 3: Application layer — types + conflict + free-slot methods

**Files:**
- Create: `backend/internal/application/conflict.go`

Read `internal/application/participants.go` (for `s.Store`, `SearchEmployeesGlobal`, import style) and `meeting_service.go:200` (`orDefault`). Build-verified (no DB harness).

- [ ] **Step 1: Create the application layer**

Create `backend/internal/application/conflict.go`:

```go
package application

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
)

// almatyLoc is the base timezone for free-slot windows (UTC+5). §4.8
var almatyLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}()

const (
	workStartHour = 9  // 09:00 working-day start (Almaty). §4.8
	workEndHour   = 18 // 18:00 working-day end (Almaty).
)

// Conflict is one participant's overlapping meeting. §4.7.2
type Conflict struct {
	Email       string
	PersonName  string
	MeetingName string
	Start, End  time.Time // UTC
}

// FreeSlot is a window where all queried participants are free. §4.8.4
type FreeSlot struct {
	Day        time.Time // start-of-day in Almaty
	Start, End time.Time // UTC
	Mins       int
}

// MeetingConflicts returns overlaps with [start,end) across emails (participants +
// organizer of each overlapping meeting), excluding excludeMeetingID (uuid.Nil =
// none). Attribution is done in Go: per overlapping meeting we load its participants
// and organizer email and keep those in the queried set. Global by email. §4.7
func (s *Services) MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	ms, err := s.Store.ListMeetingsOverlapping(ctx, emails, start, end)
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(emails))
	for _, e := range emails {
		want[e] = true
	}
	var out []Conflict
	for _, m := range ms {
		if m.ID == nil || *m.ID == excludeMeetingID {
			continue
		}
		if !meeting.Overlaps(start, end, m.StartsAt, m.EndsAt) {
			continue
		}
		hit := map[string]bool{}
		parts, perr := s.Store.ListParticipants(ctx, *m.ID)
		if perr != nil {
			return nil, perr
		}
		for _, p := range parts {
			if want[p.Email] {
				hit[p.Email] = true
			}
		}
		if m.OrganizerUserID != nil {
			if u, uerr := s.Store.GetUserByID(ctx, *m.OrganizerUserID); uerr == nil && want[u.Email] {
				hit[u.Email] = true
			}
		}
		for email := range hit {
			out = append(out, Conflict{
				Email:       email,
				PersonName:  s.personName(ctx, email),
				MeetingName: m.Name,
				Start:       m.StartsAt,
				End:         m.EndsAt,
			})
		}
	}
	// Deterministic order (map iteration above is unordered).
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start.Equal(out[j].Start) {
			return out[i].Email < out[j].Email
		}
		return out[i].Start.Before(out[j].Start)
	})
	return out, nil
}

// MeetingUpdateConflicts resolves a pending single-meeting edit's effective time,
// participants and organizer, then checks §4.7 conflicts (excluding the meeting
// itself). Returns nil when the edit does not change the time (overlap unchanged).
func (s *Services) MeetingUpdateConflicts(ctx context.Context, workspaceID, meetingID uuid.UUID, in UpdateMeetingInput) ([]Conflict, error) {
	if in.Date == nil || in.Start == nil || in.End == nil {
		return nil, nil // no time change → overlap set is unchanged
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		loc = almatyLoc
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.Start, loc)
	if err != nil {
		return nil, nil // malformed time is rejected later by UpdateMeeting
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.End, loc)
	if err != nil {
		return nil, nil
	}
	emails, err := s.meetingEmails(ctx, workspaceID, meetingID)
	if err != nil {
		return nil, err
	}
	return s.MeetingConflicts(ctx, emails, start.UTC(), end.UTC(), meetingID)
}

// meetingEmails returns a meeting's participant emails plus its organizer email.
func (s *Services) meetingEmails(ctx context.Context, workspaceID, meetingID uuid.UUID) ([]string, error) {
	m, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return nil, err
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var emails []string
	add := func(e string) {
		if e != "" && !seen[e] {
			seen[e] = true
			emails = append(emails, e)
		}
	}
	for _, p := range parts {
		add(p.Email)
	}
	if m.OrganizerUserID != nil {
		if u, uerr := s.Store.GetUserByID(ctx, *m.OrganizerUserID); uerr == nil {
			add(u.Email)
		}
	}
	return emails, nil
}

// personName resolves a display name for an email (best-effort; falls back to email).
func (s *Services) personName(ctx context.Context, email string) string {
	matches, err := s.Store.SearchEmployeesGlobal(ctx, email)
	if err == nil {
		for _, e := range matches {
			if e.Email == email {
				return e.FullName
			}
		}
	}
	return email
}

// FreeSlots finds windows where ALL emails are free within [from,to) (day-exclusive),
// Mon–Fri, workStartHour–workEndHour Almaty, gaps >= durMins. Global by email. §4.8
func (s *Services) FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]FreeSlot, error) {
	if len(emails) == 0 || durMins <= 0 {
		return nil, nil
	}
	ms, err := s.Store.ListMeetingsOverlapping(ctx, emails, from, to)
	if err != nil {
		return nil, err
	}
	busy := make([]meeting.Span, 0, len(ms))
	for _, m := range ms {
		busy = append(busy, meeting.Span{Start: m.StartsAt, End: m.EndsAt})
	}
	minDur := time.Duration(durMins) * time.Minute

	var out []FreeSlot
	for day := from.In(almatyLoc); day.Before(to); day = day.AddDate(0, 0, 1) {
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		sod := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, almatyLoc)
		winStart := time.Date(day.Year(), day.Month(), day.Day(), workStartHour, 0, 0, 0, almatyLoc)
		winEnd := time.Date(day.Year(), day.Month(), day.Day(), workEndHour, 0, 0, 0, almatyLoc)
		var dayBusy []meeting.Span
		for _, b := range busy {
			if meeting.Overlaps(b.Start, b.End, winStart, winEnd) {
				dayBusy = append(dayBusy, b)
			}
		}
		for _, f := range meeting.FreeSlots(dayBusy, winStart, winEnd, minDur) {
			out = append(out, FreeSlot{Day: sod, Start: f.Start, End: f.End, Mins: int(f.End.Sub(f.Start).Minutes())})
		}
	}
	return out, nil
}
```

> **Confirm before building:** `UpdateMeetingInput` has `Date, Start, End *string` (it does — `meeting_update.go`). `Workspace.TZ`, `GetWorkspace`, `GetMeeting`, `GetUserByID`, `ListParticipants`, `SearchEmployeesGlobal` all exist on `*postgres.Store` / `*Services`.

- [ ] **Step 2: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application/conflict.go
git commit -m "feat(meetings): MeetingConflicts/MeetingUpdateConflicts/FreeSlots §4.7-4.8"
```

---

## Task 4: §4.7 conflict warning in the `/edit` FSM

**Files:**
- Modify: `backend/internal/platform/meetingedit/service.go`

Read `service.go`: the `Backend` interface (line ~20), `OnCallback` switch (~65), and `apply()` (~222). The warning fires only on the **single-meeting** path (scope ≠ "series") when the datetime changed; series time-edit conflict checking is out of scope (documented).

- [ ] **Step 1: Extend the Backend interface**

Add to the `meetingedit` `Backend` interface (imports `application`, `uuid`, `context` already present):

```go
	MeetingUpdateConflicts(ctx context.Context, workspaceID, meetingID uuid.UUID, in application.UpdateMeetingInput) ([]application.Conflict, error)
```

(`*application.Services` implements this from Task 3.)

- [ ] **Step 2: Extract `doApply`, add the conflict check + `applyForce`**

Refactor `apply()`. Keep the existing scope/empty-overrides guards and the existing UpdateMeeting/UpdateSeries body, but move the body that runs **after** the guards into a new `doApply(ctx, telegramID, st)` method. Then make `apply` run the conflict check first. Concretely:

Replace the current `apply` function with:

```go
func (s *Service) apply(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	if st.SeriesID != "" && st.Scope == "" {
		return Reply{Text: "Сначала выбери: эту встречу или всю серию.", Keyboard: scopeReply().Keyboard, Edit: true}
	}
	if len(st.Overrides) == 0 {
		return Reply{Text: "Нет изменений. Выбери поле или нажми «Отмена».", Keyboard: menuKeyboard(st.Scope), Edit: true}
	}
	// §4.7: on a single-meeting time change, warn about participant/organizer overlaps.
	if st.Scope != "series" {
		if _, ok := st.Overrides["date"]; ok {
			ws, _ := uuid.Parse(st.WorkspaceID)
			mid, _ := uuid.Parse(st.MeetingID)
			conflicts, cerr := s.backend.MeetingUpdateConflicts(ctx, ws, mid, toInput(st.Overrides))
			if cerr == nil && len(conflicts) > 0 {
				return Reply{Text: formatConflictWarning(conflicts), Keyboard: conflictKeyboard(), Edit: true}
			}
		}
	}
	return s.doApply(ctx, telegramID, st)
}

// applyForce skips the §4.7 conflict warning (user chose "Да, применить").
func (s *Service) applyForce(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	return s.doApply(ctx, telegramID, st)
}
```

Create `doApply` containing the **existing** post-guard body of the old `apply` (the `ws, _ := uuid.Parse(...)` lines through the UpdateSeries/UpdateMeeting error handling and final success reply). Its signature:

```go
func (s *Service) doApply(ctx context.Context, telegramID int64, st *State) Reply {
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	// ... (verbatim existing body: if st.Scope == "series" { UpdateSeries ... } else { UpdateMeeting ... })
}
```

> Move the existing logic verbatim; do not rewrite it. Only the outer guards moved up into `apply`.

- [ ] **Step 3: Add the warning text + keyboard helpers**

Add to `service.go` (package already imports `fmt`, `strings`, `time`, `application`):

```go
func formatConflictWarning(cs []application.Conflict) string {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		loc = time.FixedZone("Almaty", 5*60*60)
	}
	var b strings.Builder
	b.WriteString("⚠ Внимание! У следующих участников уже есть встречи в это время:\n")
	for _, c := range cs {
		fmt.Fprintf(&b, "- %s — «%s» (%s–%s)\n",
			c.PersonName, c.MeetingName, c.Start.In(loc).Format("15:04"), c.End.In(loc).Format("15:04"))
	}
	b.WriteString("\nПродолжить создание встречи?")
	return b.String()
}

func conflictKeyboard() [][]Button {
	return [][]Button{{
		{Text: "Да, применить", Data: "medit:applyforce"},
		{Text: "Изменить время", Data: "medit:field:datetime"},
	}}
}
```

(`medit:field:datetime` reuses the existing `field` handler, which returns to the datetime step keeping other overrides.)

- [ ] **Step 4: Route the new callback**

In `OnCallback`'s switch, add right after the `case data == "medit:apply":` line:

```go
	case data == "medit:applyforce":
		return s.applyForce(ctx, telegramID), true
```

- [ ] **Step 5: Build + vet + existing tests**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/platform/meetingedit/ && env -u GOROOT go test ./internal/platform/meetingedit/`
Expected: builds; existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/meetingedit/
git commit -m "feat(meetings): §4.7 conflict warning on /edit single-meeting apply"
```

---

## Task 5: `/checker` FSM package (§4.8)

**Files:**
- Create: `backend/internal/platform/checker/state.go`
- Create: `backend/internal/platform/checker/redis_sessions.go`
- Create: `backend/internal/platform/checker/parse.go`
- Create: `backend/internal/platform/checker/service.go`
- Test: `backend/internal/platform/checker/parse_test.go`

Read `internal/platform/scheduleview/` (all files) — `/checker` mirrors it (Reply value+bool, `Get/Set/Del`, `[][]Button`, `NewRedisSessions`).

- [ ] **Step 1: State + Reply/Button**

Create `backend/internal/platform/checker/state.go`:

```go
// Package checker drives the /checker common-free-time bot flow (§4.8).
package checker

// Steps for the checker FSM.
const (
	stepParticipants = "participants"
	stepRange        = "range"
	stepDuration     = "duration"
)

// State is the persisted /checker conversation state.
type State struct {
	Step   string   `json:"step"`
	Emails []string `json:"emails,omitempty"` // chosen participant emails
	Cands  []string `json:"cands,omitempty"`  // last search candidates (index → email)
	From   string   `json:"from,omitempty"`   // YYYY-MM-DD (inclusive)
	To     string   `json:"to,omitempty"`     // YYYY-MM-DD (inclusive)
}

// Button is one inline-keyboard button.
type Button struct {
	Text string
	Data string
}

// Reply is what the FSM returns for the handler to send.
type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
```

- [ ] **Step 2: Redis sessions**

Copy `backend/internal/platform/scheduleview/redis_sessions.go` to `backend/internal/platform/checker/redis_sessions.go`, change the package to `checker`, and change the Redis key prefix from the scheduleview one to `"checker:"` (keep the same TTL). Keep the `Get/Set/Del` method set and the `NewRedisSessions(rdb *redis.Client)` constructor.

> Read scheduleview's file and mirror it exactly; only the package name, key prefix, and the `*State` type (already `checker.State`) change.

- [ ] **Step 3: Parse helpers + test**

Create `backend/internal/platform/checker/parse.go`:

```go
package checker

import (
	"fmt"
	"strings"
	"time"
)

func almaty() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}

// parseRange parses "YYYY-MM-DD..YYYY-MM-DD" into inclusive (from,to) dates.
// End must not precede start.
func parseRange(s string, loc *time.Location) (from, to time.Time, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "..", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("формат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД")
	}
	d1, e1 := time.ParseInLocation("2006-01-02", strings.TrimSpace(parts[0]), loc)
	d2, e2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(parts[1]), loc)
	if e1 != nil || e2 != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата (ГГГГ-ММ-ДД)")
	}
	if d2.Before(d1) {
		return time.Time{}, time.Time{}, fmt.Errorf("конец раньше начала")
	}
	return d1, d2, nil
}

var ruWeekday = map[time.Weekday]string{
	time.Monday: "Пн", time.Tuesday: "Вт", time.Wednesday: "Ср",
	time.Thursday: "Чт", time.Friday: "Пт", time.Saturday: "Сб", time.Sunday: "Вс",
}

// dayLabel renders "Пн, 02.06" for a free-slot day in loc.
func dayLabel(day time.Time, loc *time.Location) string {
	d := day.In(loc)
	return fmt.Sprintf("%s, %02d.%02d", ruWeekday[d.Weekday()], d.Day(), int(d.Month()))
}
```

Create `backend/internal/platform/checker/parse_test.go`:

```go
package checker

import (
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	loc := almaty()
	from, to, err := parseRange("2026-06-01..2026-06-03", loc)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if from.Day() != 1 || to.Day() != 3 {
		t.Fatalf("from=%v to=%v", from, to)
	}
	if _, _, err := parseRange("2026-06-03..2026-06-01", loc); err == nil {
		t.Fatal("expected error for reversed range")
	}
	if _, _, err := parseRange("bad", loc); err == nil {
		t.Fatal("expected format error")
	}
}

func TestDayLabel(t *testing.T) {
	loc := almaty()
	d := time.Date(2026, 6, 1, 0, 0, 0, 0, loc) // Monday
	if got := dayLabel(d, loc); got != "Пн, 01.06" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 4: Service**

Create `backend/internal/platform/checker/service.go`:

```go
package checker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the checker FSM needs (satisfied by *application.Services).
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	backend  Backend
	sessions sessions
}

func New(backend Backend, sess sessions) *Service {
	return &Service{backend: backend, sessions: sess}
}

// Start handles /checker: prompts for the first participant.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepParticipants})
	return Reply{Text: "Поиск общего свободного времени.\nВведи имя или email участника:"}
}

// OnText feeds free text into the active step. bool=false when no active session.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepParticipants:
		return s.search(ctx, telegramID, st, text), true
	case stepRange:
		return s.setRange(ctx, telegramID, st, text), true
	}
	return Reply{}, false
}

// OnCallback handles chk:* taps. bool=false for non-chk data.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	if !strings.HasPrefix(data, "chk:") {
		return Reply{}, false
	}
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /checker"}, true
	}
	switch {
	case strings.HasPrefix(data, "chk:add:"):
		return s.add(ctx, telegramID, st, strings.TrimPrefix(data, "chk:add:")), true
	case data == "chk:done":
		return s.done(ctx, telegramID, st), true
	case strings.HasPrefix(data, "chk:dur:"):
		return s.duration(ctx, telegramID, st, strings.TrimPrefix(data, "chk:dur:")), true
	}
	return Reply{}, true
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй ещё раз:"}
	}
	var rows [][]Button
	var cands []string
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("chk:add:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if len(cands) == 0 {
		return Reply{Text: "Ничего не найдено. Введи другой запрос:"}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	if len(st.Emails) > 0 {
		rows = append(rows, []Button{{Text: "Готово ✅", Data: "chk:done"}})
	}
	return Reply{Text: "Выбери участника (можно несколько):", Keyboard: rows}
}

func (s *Service) add(ctx context.Context, telegramID int64, st *State, idxStr string) Reply {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(st.Cands) {
		return Reply{Text: "Не найдено, поищи ещё раз:"}
	}
	email := st.Cands[i]
	for _, e := range st.Emails {
		if e == email {
			return Reply{Text: "Уже добавлен. Ищи ещё или нажми «Готово».",
				Keyboard: [][]Button{{{Text: "Готово ✅", Data: "chk:done"}}}}
		}
	}
	st.Emails = append(st.Emails, email)
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{
		Text:     fmt.Sprintf("Добавлен: %s\nУчастников: %d. Ищи ещё или нажми «Готово».", email, len(st.Emails)),
		Keyboard: [][]Button{{{Text: "Готово ✅", Data: "chk:done"}}},
	}
}

func (s *Service) done(ctx context.Context, telegramID int64, st *State) Reply {
	if len(st.Emails) == 0 {
		return Reply{Text: "Добавь хотя бы одного участника."}
	}
	st.Step = stepRange
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Введи диапазон дат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД"}
}

func (s *Service) setRange(ctx context.Context, telegramID int64, st *State, text string) Reply {
	from, to, err := parseRange(text, almaty())
	if err != nil {
		return Reply{Text: err.Error() + "\nПопробуй ещё раз:"}
	}
	st.From = from.Format("2006-01-02")
	st.To = to.Format("2006-01-02")
	st.Step = stepDuration
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Выбери длительность встречи:", Keyboard: durationKeyboard()}
}

func (s *Service) duration(ctx context.Context, telegramID int64, st *State, durStr string) Reply {
	durMins, err := strconv.Atoi(durStr)
	if err != nil || durMins <= 0 {
		return Reply{Text: "Неверная длительность."}
	}
	loc := almaty()
	from, _ := time.ParseInLocation("2006-01-02", st.From, loc)
	toIncl, _ := time.ParseInLocation("2006-01-02", st.To, loc)
	slots, err := s.backend.FreeSlots(ctx, st.Emails, from, toIncl.AddDate(0, 0, 1), durMins)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй позже."}
	}
	n := len(st.Emails)
	_ = s.sessions.Del(ctx, telegramID)
	if len(slots) == 0 {
		return Reply{Text: "Общих свободных слотов в выбранном диапазоне не найдено.\n" +
			"Попробуй: расширить диапазон дат / уменьшить длительность / изменить состав участников."}
	}
	return Reply{Text: formatSlots(slots, n, loc)}
}

func durationKeyboard() [][]Button {
	return [][]Button{
		{{Text: "15 мин", Data: "chk:dur:15"}, {Text: "30 мин", Data: "chk:dur:30"}, {Text: "45 мин", Data: "chk:dur:45"}},
		{{Text: "1 час", Data: "chk:dur:60"}, {Text: "1.5 часа", Data: "chk:dur:90"}, {Text: "2 часа", Data: "chk:dur:120"}},
	}
}

func formatSlots(slots []application.FreeSlot, n int, loc *time.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✅ Общее свободное время для %d участников:\n\n", n)
	for _, sl := range slots {
		fmt.Fprintf(&b, "📅 %s — %s–%s (%d мин свободно)\n",
			dayLabel(sl.Day, loc), sl.Start.In(loc).Format("15:04"), sl.End.In(loc).Format("15:04"), sl.Mins)
	}
	return b.String()
}
```

> No "Создать встречу на этот слот" button — out of scope (spec).

- [ ] **Step 5: Build + vet + test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/platform/checker/ && env -u GOROOT go test ./internal/platform/checker/ -v`
Expected: builds, vets, tests PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/checker/
git commit -m "feat(meetings): /checker common-free-time FSM §4.8"
```

---

## Task 6: Wire `/checker` into the dispatcher

**Files:**
- Modify: `backend/internal/infrastructure/telegram/multitenant.go`

Read `multitenant.go` to copy the `schedule` wiring pattern exactly.

- [ ] **Step 1: Import + interface + field + construction**

1. Add import: `"github.com/Jaryq-Lab/notify-bot/internal/platform/checker"`.
2. Add `checker.Backend` to the `botBackend` interface:
   ```go
   type botBackend interface {
       meetingedit.Backend
       scheduleview.Backend
       checker.Backend
   }
   ```
3. Add field to `MultiHandler`: `checker *checker.Service`.
4. In `NewMultiHandler`, after the `schedule := ...` line:
   ```go
   chk := checker.New(backend, checker.NewRedisSessions(rdb))
   ```
   and add `checker: chk,` to the returned struct literal.

- [ ] **Step 2: Command + OnText + callback routing**

5. In `Handle`'s no-command private branch, after the `h.schedule.OnText(...)` block:
   ```go
   if reply, handled := h.checker.OnText(ctx, from.ID, text); handled {
       h.sendCheckerReply(ctx, b, chatID, 0, reply)
   }
   ```
6. In the command `switch`, add a `/checker` case (guard like `/schedule`):
   ```go
   case "/checker":
       if isPrivate {
           if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
               h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
               return
           }
           h.sendCheckerReply(ctx, b, chatID, 0, h.checker.Start(ctx, from.ID))
       }
   ```
7. In `handleCallback`, after the `sched:` block:
   ```go
   if strings.HasPrefix(cq.Data, "chk:") {
       if reply, handled := h.checker.OnCallback(ctx, cq.From.ID, cq.Data); handled && cq.Message.Message != nil {
           h.sendCheckerReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
       }
   }
   ```

- [ ] **Step 3: Reply sender + markup helper**

Add (mirroring `sendSchedReply` / `toSchedMarkup`):

```go
func (h *MultiHandler) sendCheckerReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply checker.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toCheckerMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toCheckerMarkup(rows [][]checker.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}
```

- [ ] **Step 4: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/telegram/`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/telegram/multitenant.go
git commit -m "feat(meetings): wire /checker into bot dispatcher §4.8"
```

---

## Task 7: REST endpoints

**Files:**
- Create: `backend/internal/delivery/http/handlers/meeting_availability.go`
- Modify: `backend/internal/delivery/http/app.go`

Read `handlers/meetings.go` (receiver `*API`, field `App`, `c.Locals("workspace_id")`, error helpers) and `app.go:132-136`.

- [ ] **Step 1: Handlers**

Create `backend/internal/delivery/http/handlers/meeting_availability.go`:

```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func almatyLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}

type conflictsRequest struct {
	Date             string   `json:"date"`  // YYYY-MM-DD
	Start            string   `json:"start"` // HH:MM
	End              string   `json:"end"`   // HH:MM
	Participants     []string `json:"participants"`
	ExcludeMeetingID string   `json:"exclude_meeting_id"`
}

type conflictItem struct {
	Email       string    `json:"email"`
	PersonName  string    `json:"person_name"`
	MeetingName string    `json:"meeting_name"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

// MeetingConflicts is the advisory §4.7 overlap check (read-only). Workspace authz
// is enforced by RequireWorkspaceAccess on the route.
func (a *API) MeetingConflicts(c *fiber.Ctx) error {
	var req conflictsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid date/time")
	}
	exclude := uuid.Nil
	if req.ExcludeMeetingID != "" {
		exclude, _ = uuid.Parse(req.ExcludeMeetingID)
	}
	conflicts, err := a.App.MeetingConflicts(c.Context(), req.Participants, start.UTC(), end.UTC(), exclude)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "conflict check failed")
	}
	out := make([]conflictItem, 0, len(conflicts))
	for _, cf := range conflicts {
		out = append(out, conflictItem{cf.Email, cf.PersonName, cf.MeetingName, cf.Start, cf.End})
	}
	return c.JSON(fiber.Map{"conflicts": out})
}

type freeSlotsRequest struct {
	From         string   `json:"from"` // YYYY-MM-DD (inclusive)
	To           string   `json:"to"`   // YYYY-MM-DD (inclusive)
	Participants []string `json:"participants"`
	DurationMins int      `json:"duration_mins"`
}

type freeSlotItem struct {
	Day   time.Time `json:"day"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Mins  int       `json:"mins"`
}

// FreeSlots is the §4.8 common-free-time finder (read-only).
func (a *API) FreeSlots(c *fiber.Ctx) error {
	var req freeSlotsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	from, err1 := time.ParseInLocation("2006-01-02", req.From, loc)
	toIncl, err2 := time.ParseInLocation("2006-01-02", req.To, loc)
	if err1 != nil || err2 != nil || toIncl.Before(from) || req.DurationMins <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/duration")
	}
	slots, err := a.App.FreeSlots(c.Context(), req.Participants, from, toIncl.AddDate(0, 0, 1), req.DurationMins)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "free-slot search failed")
	}
	out := make([]freeSlotItem, 0, len(slots))
	for _, sl := range slots {
		out = append(out, freeSlotItem{sl.Day, sl.Start, sl.End, sl.Mins})
	}
	return c.JSON(fiber.Map{"slots": out})
}
```

> Confirm the receiver is `*API` and the application field is `App` (it is, per `meetings.go`). If `time` is already imported elsewhere in the package that's fine — this is a new file.

- [ ] **Step 2: Register routes**

In `app.go`, inside the `ws` group right after `ws.Delete("/meetings/:mid", api.DeleteMeeting)`:

```go
	ws.Post("/meetings/conflicts", api.MeetingConflicts)
	ws.Post("/meetings/free-slots", api.FreeSlots)
```

- [ ] **Step 3: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/
git commit -m "feat(meetings): REST /meetings/conflicts + /meetings/free-slots §4.7-4.8"
```

---

## Task 8: Docs + final verification

**Files:**
- Modify: `docs/MEETINGS.md`
- Modify: `PLAN.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

Add after the §4.5 deletion `>` line in the Backend block:

```markdown
> **Conflict warning + free-time checker (§4.7–4.8, done):** `/edit` warns before applying a **single-meeting time change** if any participant or the organizer has an overlapping meeting (⚠ list with names + meeting titles; **[Да, применить] / [Изменить время]**, non-blocking per §4.7.3). A new `/checker` bot flow finds common free time: pick participants (directory search) → date range → duration preset → slots when everyone is free (Mon–Fri, 09:00–18:00 Almaty, §4.8.4) or a "no slots" message (§4.8.6). Busyness is read from the internal DB (global-by-email, like §4.6) — external/personal Google events are not seen; bot "create from slot" and series-time-edit conflict checks are out of scope. Also over REST: `POST /workspaces/:id/meetings/conflicts` and `.../free-slots` for the (mocked) Mini App. Core interval math is pure (`domain/meeting.Overlaps`/`FreeSlots`); conflict attribution is in Go (`application.MeetingConflicts`).
```

- [ ] **Step 2: Update `PLAN.md`**

Read `PLAN.md`, find the §4.7 / §4.8 (conflict warning / free-time checker) entries, mark them done matching the file's existing convention.

- [ ] **Step 3: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/...` and re-run.

- [ ] **Step 4: Commit**

```bash
git add docs/MEETINGS.md PLAN.md
git commit -m "docs(meetings): document §4.7-4.8 conflict warning + free-time checker"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** §4.7.1 (Overlaps, organizer included via `meetingEmails`) → Tasks 1,3,4; §4.7.2 (warning layout) → Task 4 `formatConflictWarning`; §4.7.3 (non-blocking; "change time" returns) → Task 4 `applyForce` + `medit:field:datetime`; §4.8.2 (≥1 participant, range, duration presets) → Task 5; §4.8.3 (algorithm, weekday skip, 09:00–18:00) → Task 3 `FreeSlots`; §4.8.4 (results layout) → Task 5 `formatSlots`; §4.8.6 (no-slots message) → Task 5 `duration`. REST → Task 7.
- **Out of scope (do not implement):** Google freebusy; configurable hours; weekend inclusion; bot create-from-slot; Mini App frontend wiring; §4.7 warning on **series** time edits (only single-meeting path warns).
- **Type consistency:** `Conflict{Email,PersonName,MeetingName,Start,End}` / `FreeSlot{Day,Start,End,Mins}` defined once in Task 3, consumed in Tasks 4,5,7. `ListMeetingsOverlapping(ctx, emails, from, to)` (Task 2) → called in Task 3. `MeetingConflicts(ctx, emails, start, end, excludeMeetingID)`, `MeetingUpdateConflicts(ctx, workspaceID, meetingID, in)`, `FreeSlots(ctx, emails, from, to, durMins)` (Task 3) → consumed in Tasks 4,5,7. FSM convention: `Reply{Text, Keyboard [][]Button, Edit}`, `(Reply, bool)` returns, sessions `Get/Set/Del` — matches `scheduleview`/`meetingedit`.
- **Known approximations:** `MeetingConflicts` does an N+1 `ListParticipants`/`GetUserByID` per overlapping meeting — acceptable (conflict lists are tiny). `personName` uses `SearchEmployeesGlobal(email)` exact-match — fine for the directory size. Warning times render in `Asia/Almaty` (workspace default), not per-workspace TZ.
