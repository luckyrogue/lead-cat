# §4.7–4.8 Conflict Warning + Free-Time Checker — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a time-conflict warning on meeting edit (§4.7) and a common-free-time checker (§4.8) to the meetings feature, exposed via both the Telegram bot and REST.

**Architecture:** A pure interval core in `domain/meeting` (`Overlaps`, `FreeSlots`), one new global-by-email repo query (`ListMeetingsOverlapping`), two read-only application methods (`MeetingConflicts`, `FreeSlots`) returning `[]Conflict` / `[]FreeSlot`, then four surfaces: the `/edit` FSM warning, a new `/checker` FSM, and two REST endpoints. Internal DB is the busyness source; queries are global-by-email with a hardcoded `Asia/Almaty` location, mirroring the shipped §4.6 `/schedule` flow.

**Tech Stack:** Go, Fiber, telebot, pgx/Postgres, Redis-backed FSM sessions, zap.

**Spec:** `docs/superpowers/specs/2026-05-31-meetings-conflict-checker-4-7-4-8-design.md`

**Conventions (read before starting):**
- Run all checks from the **repo root**: `make test && make lint && make build`. Run Go directly as `env -u GOROOT go ...` from `backend/`. **`make lint` includes a gofmt check** — always run it before committing; per-task `go test`/`go vet` will not catch gofmt drift.
- Backend test convention: **pure logic is unit-tested; I/O paths (repo, REST handlers, bot wiring) are build-verified only** (no DB harness in the postgres package).
- Bot FSMs live in `internal/platform/<name>` and share one shape: `Backend` interface + `Sessions` interface + `Service{Backend, Sessions}` with `Start`/`OnText`/`OnCallback` returning `*Reply{Text string, Keyboard *Keyboard, Edit bool}`; `Keyboard{Rows [][]Button}`, `Button{Text, Data string}`. Wired in `internal/infrastructure/telegram/multitenant.go`.
- Do **not** touch `frontend/vite.config.ts` (long-standing local-only edit).
- All times stored/compared in UTC; user-facing rendering uses `Asia/Almaty`.

**File structure (created/modified):**
- Create `backend/internal/domain/meeting/conflict.go` — `Overlaps`, `FreeSlots` (pure).
- Create `backend/internal/domain/meeting/conflict_test.go` — unit tests.
- Modify `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` — add `ListMeetingsOverlapping`.
- Create `backend/internal/application/conflict.go` — `Conflict`, `FreeSlot` types, `MeetingConflicts`, `FreeSlots`, constants.
- Modify `backend/internal/platform/meetingedit/service.go`, `state.go`, `keyboard.go` — §4.7 warning.
- Create `backend/internal/platform/checker/{state,service,parse,keyboard}.go` (+ `parse_test.go`) — `/checker` FSM.
- Modify `backend/internal/infrastructure/telegram/multitenant.go` — wire `/checker`.
- Create `backend/internal/delivery/http/handlers/meeting_availability.go` — REST handlers.
- Modify `backend/internal/delivery/http/app.go` — register two routes.
- Modify `docs/MEETINGS.md`, `PLAN.md` — status.

---

## Task 1: Pure interval core (`Overlaps`, `FreeSlots`)

**Files:**
- Create: `backend/internal/domain/meeting/conflict.go`
- Test: `backend/internal/domain/meeting/conflict_test.go`

This is pure logic — full TDD with real assertions.

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/domain/meeting/conflict_test.go`:

```go
package meeting

import (
	"testing"
	"time"
)

func t(h, m int) time.Time {
	return time.Date(2026, 6, 1, h, m, 0, 0, time.UTC)
}

func TestOverlaps(t1 *testing.T) {
	cases := []struct {
		name                   string
		aS, aE, bS, bE         time.Time
		want                   bool
	}{
		{"disjoint before", t(9, 0), t(10, 0), t(10, 0), t(11, 0), false}, // touching edge = no overlap
		{"disjoint after", t(11, 0), t(12, 0), t(9, 0), t(10, 0), false},
		{"partial", t(9, 0), t(10, 0), t(9, 30), t(11, 0), true},
		{"full contain", t(9, 0), t(12, 0), t(10, 0), t(11, 0), true},
		{"identical", t(9, 0), t(10, 0), t(9, 0), t(10, 0), true},
	}
	for _, c := range cases {
		t1.Run(c.name, func(t1 *testing.T) {
			if got := Overlaps(c.aS, c.aE, c.bS, c.bE); got != c.want {
				t1.Fatalf("Overlaps=%v want %v", got, c.want)
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

func TestFreeSlots(t1 *testing.T) {
	win, winEnd := t(9, 0), t(18, 0)
	min := 30 * time.Minute

	t1.Run("no busy = whole window", func(t1 *testing.T) {
		got := FreeSlots(nil, win, winEnd, min)
		want := []Span{{win, winEnd}}
		if !spansEqual(got, want) {
			t1.Fatalf("got %v want %v", got, want)
		}
	})

	t1.Run("busy spanning window = none", func(t1 *testing.T) {
		busy := []Span{{t(8, 0), t(19, 0)}}
		if got := FreeSlots(busy, win, winEnd, min); len(got) != 0 {
			t1.Fatalf("got %v want empty", got)
		}
	})

	t1.Run("gaps around one meeting", func(t1 *testing.T) {
		busy := []Span{{t(11, 0), t(12, 30)}}
		got := FreeSlots(busy, win, winEnd, min)
		want := []Span{{t(9, 0), t(11, 0)}, {t(12, 30), t(18, 0)}}
		if !spansEqual(got, want) {
			t1.Fatalf("got %v want %v", got, want)
		}
	})

	t1.Run("merge overlapping busy", func(t1 *testing.T) {
		busy := []Span{{t(11, 0), t(12, 0)}, {t(11, 30), t(13, 0)}}
		got := FreeSlots(busy, win, winEnd, min)
		want := []Span{{t(9, 0), t(11, 0)}, {t(13, 0), t(18, 0)}}
		if !spansEqual(got, want) {
			t1.Fatalf("got %v want %v", got, want)
		}
	})

	t1.Run("min-duration filter drops short gap", func(t1 *testing.T) {
		busy := []Span{{t(9, 15), t(17, 45)}} // leaves 9:00-9:15 (15m) and 17:45-18:00 (15m)
		if got := FreeSlots(busy, win, winEnd, min); len(got) != 0 {
			t1.Fatalf("got %v want empty (both gaps < 30m)", got)
		}
	})

	t1.Run("busy outside window clipped", func(t1 *testing.T) {
		busy := []Span{{t(6, 0), t(9, 30)}, {t(18, 30), t(20, 0)}}
		got := FreeSlots(busy, win, winEnd, min)
		want := []Span{{t(9, 30), t(18, 0)}}
		if !spansEqual(got, want) {
			t1.Fatalf("got %v want %v", got, want)
		}
	})

	t1.Run("unsorted busy input", func(t1 *testing.T) {
		busy := []Span{{t(15, 0), t(16, 0)}, {t(10, 0), t(11, 0)}}
		got := FreeSlots(busy, win, winEnd, min)
		want := []Span{{t(9, 0), t(10, 0)}, {t(11, 0), t(15, 0)}, {t(16, 0), t(18, 0)}}
		if !spansEqual(got, want) {
			t1.Fatalf("got %v want %v", got, want)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env -u GOROOT go test ./internal/domain/meeting/ -run 'TestOverlaps|TestFreeSlots' -v`
Expected: FAIL — `undefined: Overlaps`, `undefined: FreeSlots`.

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

// FreeSlots returns the gaps in the working window [winStart,winEnd) not covered
// by busy, keeping only gaps with duration >= minDur. busy spans are merged and
// clipped to the window; input need not be sorted. Result is chronological. §4.8.3
func FreeSlots(busy []Span, winStart, winEnd time.Time, minDur time.Duration) []Span {
	// Clip busy to the window and drop empties.
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

	// Merge overlapping/adjacent busy spans.
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

	// Walk the gaps between merged busy spans.
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
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/meeting/conflict.go backend/internal/domain/meeting/conflict_test.go
git commit -m "feat(meetings): pure interval core (Overlaps, FreeSlots) §4.7-4.8"
```

---

## Task 2: Repo query `ListMeetingsOverlapping`

**Files:**
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

Build-verified (no DB harness). First **read** `ListScheduleForEmail` (~line 241) and the `queryMeetingsWithParticipants` helper in this file to match the exact patterns (column const `meetingColsM`, participant hydration).

- [ ] **Step 1: Add the query**

Append to `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` (after `ListScheduleForEmail`). Use the same participant-hydrating helper that `ListScheduleForEmail` uses — if `ListScheduleForEmail` calls `queryMeetings`, use `queryMeetings`; if it hydrates participants, use the same hydrating helper. The body below assumes `queryMeetings` (adjust to match):

```go
// ListMeetingsOverlapping returns scheduled meetings overlapping [from,to) where any
// of emails is a participant or the organizer (by platform_users.email). Global by
// email (no workspace scope), mirroring ListScheduleForEmail. §4.7/§4.8
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

> **If** `MeetingConflicts` (Task 3) needs each meeting's participant emails to attribute conflicts and `queryMeetings` does NOT hydrate participants, use `queryMeetingsWithParticipants` instead so `Meeting.Participants` is populated. Verify which helper hydrates participants before choosing.

- [ ] **Step 2: Build-verify**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go
git commit -m "feat(meetings): ListMeetingsOverlapping repo query §4.7-4.8"
```

---

## Task 3: Application types + `MeetingConflicts` + `FreeSlots`

**Files:**
- Create: `backend/internal/application/conflict.go`

First **read** `backend/internal/application/participants.go` (for the `Services` receiver, `EmployeeSchedule` pattern, imports) and confirm how an organizer's display name/email is reachable (employee directory + `platform_users`). `PersonName` resolution is best-effort: prefer a directory lookup, fall back to the raw email.

- [ ] **Step 1: Create the application layer**

Create `backend/internal/application/conflict.go`:

```go
package application

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"lead-cat/backend/internal/domain/meeting"
	"lead-cat/backend/internal/infrastructure/persistence/postgres"
)

// almatyLoc is the base timezone for meeting availability (UTC+5). §4.8
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
// organizer), excluding excludeMeetingID (uuid.Nil = none). Global by email. §4.7
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
		if m.ID != nil && *m.ID == excludeMeetingID {
			continue
		}
		if !meeting.Overlaps(start, end, m.StartsAt, m.EndsAt) {
			continue
		}
		// Attribute to each queried email that is on this meeting (participant or organizer).
		for _, p := range m.Participants {
			if want[p.Email] {
				out = append(out, Conflict{
					Email:       p.Email,
					PersonName:  s.personName(ctx, p.Email),
					MeetingName: m.Name,
					Start:       m.StartsAt,
					End:         m.EndsAt,
				})
			}
		}
	}
	return out, nil
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
		wd := day.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			continue
		}
		sod := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, almatyLoc)
		winStart := time.Date(day.Year(), day.Month(), day.Day(), workStartHour, 0, 0, 0, almatyLoc)
		winEnd := time.Date(day.Year(), day.Month(), day.Day(), workEndHour, 0, 0, 0, almatyLoc)
		// Collect busy spans intersecting this day's window.
		var dayBusy []meeting.Span
		for _, b := range busy {
			if meeting.Overlaps(b.Start, b.End, winStart, winEnd) {
				dayBusy = append(dayBusy, b)
			}
		}
		for _, f := range meeting.FreeSlots(dayBusy, winStart, winEnd, minDur) {
			out = append(out, FreeSlot{
				Day:   sod,
				Start: f.Start,
				End:   f.End,
				Mins:  int(f.End.Sub(f.Start).Minutes()),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, nil
}
```

> **Adjust to reality:** confirm `s.Store` is the field name, that `postgres.Meeting` exposes `ID *uuid.UUID`, `StartsAt`, `EndsAt`, `Name`, `Participants []postgres.MeetingParticipant`, and `MeetingParticipant.Email`. If `Meeting.Participants` is NOT hydrated by the helper chosen in Task 2, switch Task 2 to `queryMeetingsWithParticipants`. Confirm `postgres.Employee` has `FullName`/`Email`.

- [ ] **Step 2: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application/conflict.go
git commit -m "feat(meetings): MeetingConflicts + FreeSlots application methods §4.7-4.8"
```

---

## Task 4: §4.7 conflict warning in the `/edit` FSM

**Files:**
- Modify: `backend/internal/platform/meetingedit/service.go`
- Modify: `backend/internal/platform/meetingedit/keyboard.go`

First **read** `service.go` fully: the `Backend` interface, `OnCallback` switch (routes by `parts[1]`), `handleApply`/`applyResult`, `handleField` (datetime returns to the date step), and how the current meeting + participants + organizer email are available in `State` / via the backend. The warning fires when the user taps **Apply** after changing the datetime.

- [ ] **Step 1: Extend the Backend interface**

In `service.go`, add to the `meetingedit` `Backend` interface:

```go
	MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]application.Conflict, error)
```

(Ensure `time` and `application` are imported. `*application.Services` already implements this from Task 3.)

- [ ] **Step 2: Add a forced-apply callback + conflict check**

In `OnCallback`'s switch, add a case alongside `"apply"`:

```go
	case "applyforce":
		return s.applyResult(ctx, telegramID, st)
```

Change `handleApply` so that — when a datetime override is present — it checks conflicts before applying. Replace the body of `handleApply` with:

```go
func (s *Service) handleApply(ctx context.Context, telegramID int64, st *State) (*Reply, error) {
	if st.SeriesID != "" && st.Scope == "" {
		return s.scopeReply(), nil
	}
	if r, err := s.maybeConflictWarning(ctx, st); err != nil {
		return nil, err
	} else if r != nil {
		return r, nil
	}
	return s.applyResult(ctx, telegramID, st)
}

// maybeConflictWarning returns a §4.7 warning Reply if the (possibly edited) time
// overlaps any participant's or the organizer's other meetings; nil = no conflict.
func (s *Service) maybeConflictWarning(ctx context.Context, st *State) (*Reply, error) {
	// Resolve the effective start/end: edited datetime override if present, else current.
	start, end, ok := st.effectiveSpan()
	if !ok {
		return nil, nil // no datetime info to check
	}
	emails := st.participantEmails() // participants + organizer email
	if len(emails) == 0 {
		return nil, nil
	}
	mID, _ := uuid.Parse(st.MeetingID)
	conflicts, err := s.Backend.MeetingConflicts(ctx, emails, start, end, mID)
	if err != nil {
		return nil, err
	}
	if len(conflicts) == 0 {
		return nil, nil
	}
	return &Reply{Text: formatConflictWarning(conflicts), Keyboard: conflictKeyboard()}, nil
}
```

> **Adapt to the real `State`:** implement `effectiveSpan()` and `participantEmails()` as small helpers (or inline) using whatever the FSM already stores. The datetime override lives in `st.Overrides` (parsed via the existing `parseDateTime`); current values are in `st.Cur`. Participant emails are in `st.PartList`; the organizer's email must be included — if it's not already in state, load it via the backend's meeting-for-edit method. Convert the local datetime to UTC the same way `applyResult`/`UpdateMeeting` does. If wiring the exact emails proves heavy, at minimum check `st.PartList`; note any omission in the task's commit message.

- [ ] **Step 3: Add the warning text + keyboard**

Add to `service.go` (or a small new file `conflict.go` in the package):

```go
func formatConflictWarning(cs []application.Conflict) string {
	loc, _ := time.LoadLocation("Asia/Almaty")
	var b strings.Builder
	b.WriteString("⚠ Внимание! У следующих участников уже есть встречи в это время:\n")
	for _, c := range cs {
		s := c.Start.In(loc).Format("15:04")
		e := c.End.In(loc).Format("15:04")
		fmt.Fprintf(&b, "- %s — «%s» (%s–%s)\n", c.PersonName, c.MeetingName, s, e)
	}
	b.WriteString("\nПродолжить создание встречи?")
	return b.String()
}
```

In `keyboard.go` add:

```go
func conflictKeyboard() *Keyboard {
	return &Keyboard{Rows: [][]Button{{
		{Text: "Да, применить", Data: "medit:applyforce"},
		{Text: "Изменить время", Data: "medit:field:datetime"},
	}}}
}
```

(`medit:field:datetime` reuses the existing handler that returns the user to the datetime step keeping other overrides.)

- [ ] **Step 4: Build + vet + existing tests**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/platform/meetingedit/ && env -u GOROOT go test ./internal/platform/meetingedit/`
Expected: builds; existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/meetingedit/
git commit -m "feat(meetings): §4.7 conflict warning on /edit apply"
```

---

## Task 5: `/checker` FSM package (§4.8)

**Files:**
- Create: `backend/internal/platform/checker/state.go`
- Create: `backend/internal/platform/checker/keyboard.go`
- Create: `backend/internal/platform/checker/parse.go`
- Create: `backend/internal/platform/checker/service.go`
- Test: `backend/internal/platform/checker/parse_test.go`

First **read** `internal/platform/scheduleview/` (all four files) — `/checker` mirrors it closely. Reuse the `parseRange` date convention from `scheduleview/parse.go`.

- [ ] **Step 1: State**

Create `backend/internal/platform/checker/state.go`:

```go
// Package checker drives the /checker common-free-time bot flow (§4.8).
package checker

// Step values for the checker FSM.
const (
	stepParticipants = "participants"
	stepRange        = "range"
	stepDuration     = "duration"
)

// State is the persisted /checker conversation state.
type State struct {
	Step    string   `json:"step"`
	Emails  []string `json:"emails"`  // chosen participant emails
	Names   []string `json:"names"`   // parallel display names
	From    string   `json:"from"`    // YYYY-MM-DD
	To      string   `json:"to"`      // YYYY-MM-DD
}
```

- [ ] **Step 2: Keyboard**

Create `backend/internal/platform/checker/keyboard.go`:

```go
package checker

// Keyboard / Button mirror the other FSM packages.
type Keyboard struct {
	Rows [][]Button
}

type Button struct {
	Text string
	Data string
}

// durationKeyboard offers the §4.8.2 duration presets (minutes).
func durationKeyboard() *Keyboard {
	return &Keyboard{Rows: [][]Button{
		{{Text: "15 мин", Data: "chk:dur:15"}, {Text: "30 мин", Data: "chk:dur:30"}, {Text: "45 мин", Data: "chk:dur:45"}},
		{{Text: "1 час", Data: "chk:dur:60"}, {Text: "1.5 часа", Data: "chk:dur:90"}, {Text: "2 часа", Data: "chk:dur:120"}},
	}}
}

// doneKeyboard lets the user finish picking participants.
func doneKeyboard() *Keyboard {
	return &Keyboard{Rows: [][]Button{{{Text: "Готово", Data: "chk:done"}}}}
}
```

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

// parseRange parses "YYYY-MM-DD..YYYY-MM-DD" into [from,to) where to is the day
// AFTER the end date (exclusive). End must not precede start.
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
	return d1, d2.AddDate(0, 0, 1), nil
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
	if from.Day() != 1 || to.Day() != 4 { // to is exclusive (day after end)
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

- [ ] **Step 4: Run parse tests (fail → pass)**

Run: `cd backend && env -u GOROOT go test ./internal/platform/checker/ -v`
Expected first: FAIL to compile (no `service.go` yet is fine; package still compiles with these files). If it compiles, tests PASS. If `service.go` references are needed to compile, proceed to Step 5 then re-run.

- [ ] **Step 5: Service**

Create `backend/internal/platform/checker/service.go`:

```go
package checker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lead-cat/backend/internal/application"
	"lead-cat/backend/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the checker FSM needs.
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
}

// Sessions persists FSM state across bot updates (Redis-backed).
type Sessions interface {
	Load(ctx context.Context, telegramID int64) (*State, error)
	Save(ctx context.Context, telegramID int64, st *State) error
	Clear(ctx context.Context, telegramID int64) error
}

// Service implements the /checker conversation.
type Service struct {
	Backend  Backend
	Sessions Sessions
}

// New builds the checker service.
func New(b Backend, s Sessions) *Service { return &Service{Backend: b, Sessions: s} }

// Reply is one bot response.
type Reply struct {
	Text     string
	Keyboard *Keyboard
	Edit     bool
}

// Start begins the /checker flow.
func (s *Service) Start(ctx context.Context, telegramID int64) (*Reply, error) {
	st := &State{Step: stepParticipants}
	if err := s.Sessions.Save(ctx, telegramID, st); err != nil {
		return nil, err
	}
	return &Reply{Text: "Поиск общего свободного времени.\nВведите имя или email участника:"}, nil
}

// OnText handles a free-text message during the flow.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (*Reply, error) {
	st, err := s.Sessions.Load(ctx, telegramID)
	if err != nil || st == nil {
		return nil, err
	}
	switch st.Step {
	case stepParticipants:
		return s.handleSearch(ctx, telegramID, st, text)
	case stepRange:
		return s.handleRange(ctx, telegramID, st, text)
	}
	return nil, nil
}

// OnCallback handles an inline-keyboard tap during the flow.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (*Reply, error) {
	st, err := s.Sessions.Load(ctx, telegramID)
	if err != nil || st == nil {
		return nil, err
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return nil, nil
	}
	switch parts[1] {
	case "add":
		return s.handleAdd(ctx, telegramID, st, parts[2])
	case "done":
		return s.handleDone(ctx, telegramID, st)
	case "dur":
		return s.handleDuration(ctx, telegramID, st, parts[2])
	}
	return nil, nil
}

func (s *Service) handleSearch(ctx context.Context, telegramID int64, st *State, text string) (*Reply, error) {
	matches, err := s.Backend.SearchEmployeesGlobal(ctx, strings.TrimSpace(text))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return &Reply{Text: "Ничего не найдено. Попробуйте другой запрос:"}, nil
	}
	kb := &Keyboard{}
	for _, e := range matches {
		kb.Rows = append(kb.Rows, []Button{{Text: fmt.Sprintf("%s (%s)", e.FullName, e.Email), Data: "chk:add:" + e.Email}})
	}
	if len(st.Emails) > 0 {
		kb.Rows = append(kb.Rows, doneKeyboard().Rows...)
	}
	return &Reply{Text: "Выберите участника (можно несколько):", Keyboard: kb}, nil
}

func (s *Service) handleAdd(ctx context.Context, telegramID int64, st *State, email string) (*Reply, error) {
	for _, e := range st.Emails {
		if e == email {
			return &Reply{Text: "Уже добавлен. Ищите ещё или нажмите «Готово».", Keyboard: doneKeyboard()}, nil
		}
	}
	st.Emails = append(st.Emails, email)
	if err := s.Sessions.Save(ctx, telegramID, st); err != nil {
		return nil, err
	}
	return &Reply{
		Text:     fmt.Sprintf("Добавлен: %s\nУчастников: %d. Ищите ещё или нажмите «Готово».", email, len(st.Emails)),
		Keyboard: doneKeyboard(),
	}, nil
}

func (s *Service) handleDone(ctx context.Context, telegramID int64, st *State) (*Reply, error) {
	if len(st.Emails) == 0 {
		return &Reply{Text: "Добавьте хотя бы одного участника."}, nil
	}
	st.Step = stepRange
	if err := s.Sessions.Save(ctx, telegramID, st); err != nil {
		return nil, err
	}
	return &Reply{Text: "Введите диапазон дат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД"}, nil
}

func (s *Service) handleRange(ctx context.Context, telegramID int64, st *State, text string) (*Reply, error) {
	from, to, err := parseRange(text, almaty())
	if err != nil {
		return &Reply{Text: err.Error()}, nil
	}
	st.From = from.Format("2006-01-02")
	st.To = to.AddDate(0, 0, -1).Format("2006-01-02") // store inclusive end for display
	st.Step = stepDuration
	if err := s.Sessions.Save(ctx, telegramID, st); err != nil {
		return nil, err
	}
	return &Reply{Text: "Выберите длительность встречи:", Keyboard: durationKeyboard()}, nil
}

func (s *Service) handleDuration(ctx context.Context, telegramID int64, st *State, durStr string) (*Reply, error) {
	durMins, err := strconv.Atoi(durStr)
	if err != nil || durMins <= 0 {
		return &Reply{Text: "Неверная длительность."}, nil
	}
	loc := almaty()
	from, _ := time.ParseInLocation("2006-01-02", st.From, loc)
	toIncl, _ := time.ParseInLocation("2006-01-02", st.To, loc)
	to := toIncl.AddDate(0, 0, 1) // exclusive upper bound

	slots, err := s.Backend.FreeSlots(ctx, st.Emails, from, to, durMins)
	if err != nil {
		return nil, err
	}
	s.Sessions.Clear(ctx, telegramID)
	if len(slots) == 0 {
		return &Reply{Text: "Общих свободных слотов в выбранном диапазоне не найдено.\n" +
			"Попробуйте: расширить диапазон дат / уменьшить длительность / изменить состав участников."}, nil
	}
	return &Reply{Text: formatSlots(slots, len(st.Emails), loc)}, nil
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

- [ ] **Step 6: Build + vet + test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/platform/checker/ && env -u GOROOT go test ./internal/platform/checker/ -v`
Expected: builds, vets, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/platform/checker/
git commit -m "feat(meetings): /checker common-free-time FSM §4.8"
```

---

## Task 6: Wire `/checker` into the dispatcher

**Files:**
- Modify: `backend/internal/infrastructure/telegram/multitenant.go`

First **read** `multitenant.go`: the `Dispatcher` struct, `NewDispatcher` (each FSM wired via `newSessions[State](rdb, "prefix")`), `HandleText` command switch, `routeText` OnText chain, `HandleCallback` prefix switch.

- [ ] **Step 1: Add the field, wiring, and routes**

1. Import `"lead-cat/backend/internal/platform/checker"`.
2. Add field to `Dispatcher`: `checker *checker.Service`.
3. In `NewDispatcher`, add: `checker: checker.New(app, newSessions[checker.State](rdb, "chk")),`
4. In `HandleText`'s switch, add: `case text == "/checker": return adapt(d.checker.Start(ctx, telegramID))`
5. In `routeText`, add (same shape as the others): `if r, err := d.checker.OnText(ctx, telegramID, text); err != nil || r != nil { return adapt(r, err) }`
6. In `HandleCallback`'s prefix switch, add: `case "chk": return adapt(d.checker.OnCallback(ctx, telegramID, data))`

> `adapt` converts `*Reply` → `(string, *Keyboard, error)`. The `checker.Reply`/`checker.Keyboard` types are package-local but structurally identical; if `adapt` is generic/duck-typed it just works, otherwise check how `scheduleview.Reply` is adapted and mirror it exactly (there may be a per-package `adapt` or a shared conversion — match the existing pattern for `d.schedule`).

- [ ] **Step 2: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/telegram/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/infrastructure/telegram/multitenant.go
git commit -m "feat(meetings): wire /checker into bot dispatcher §4.8"
```

---

## Task 7: REST endpoints

**Files:**
- Create: `backend/internal/delivery/http/handlers/meeting_availability.go`
- Modify: `backend/internal/delivery/http/app.go`

First **read** `handlers/meetings.go` (handler receiver type, how it reads the JSON body, success/error response helpers) and `app.go` lines ~132–136 (the `ws` group registration).

- [ ] **Step 1: Handlers**

Create `backend/internal/delivery/http/handlers/meeting_availability.go`. Match the existing handler receiver/response conventions in `meetings.go` (the snippet below assumes an `*API` receiver and `c.JSON`; adapt names):

```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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

// MeetingConflicts is the advisory §4.7 overlap check (read-only).
func (a *API) MeetingConflicts(c *fiber.Ctx) error {
	var req conflictsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc, _ := time.LoadLocation("Asia/Almaty")
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid date/time")
	}
	exclude := uuid.Nil
	if req.ExcludeMeetingID != "" {
		exclude, _ = uuid.Parse(req.ExcludeMeetingID)
	}
	conflicts, err := a.app.MeetingConflicts(c.Context(), req.Participants, start.UTC(), end.UTC(), exclude)
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
	From         string   `json:"from"` // YYYY-MM-DD
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
	loc, _ := time.LoadLocation("Asia/Almaty")
	from, err1 := time.ParseInLocation("2006-01-02", req.From, loc)
	toIncl, err2 := time.ParseInLocation("2006-01-02", req.To, loc)
	if err1 != nil || err2 != nil || toIncl.Before(from) || req.DurationMins <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/duration")
	}
	slots, err := a.app.FreeSlots(c.Context(), req.Participants, from, toIncl.AddDate(0, 0, 1), req.DurationMins)
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

> **Adapt:** the receiver type (`*API` vs other), the application field name (`a.app` vs `a.services`), and error/response helpers must match `meetings.go`. The `:id` workspace param is not used in the query — authz is already enforced by `RequireWorkspaceAccess` on the route group.

- [ ] **Step 2: Register routes**

In `backend/internal/delivery/http/app.go`, inside the `ws` group (right after the existing `ws.Delete("/meetings/:mid", ...)` line):

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

Add a status line under the Backend "done" block (after the §4.5 deletion line), matching the existing `>` quote style:

```markdown
> **Conflict warning + free-time checker (§4.7–4.8, done):** `/edit` now warns before applying a time change if any participant or the organizer has an overlapping meeting (⚠ list with names + meeting titles; **[Да, применить] / [Изменить время]**, non-blocking per §4.7.3). A new `/checker` bot flow finds common free time: pick participants (directory search) → date range → duration preset → list of slots when everyone is free (Mon–Fri, 09:00–18:00 Almaty, §4.8.4) or a "no slots" message (§4.8.6). Busyness is read from the internal DB (global-by-email, like §4.6) — external/personal Google events are not seen; bot "create from slot" is out of scope. Also exposed over REST: `POST /workspaces/:id/meetings/conflicts` and `.../free-slots` for the (mocked) Mini App. Core interval math is pure (`domain/meeting.Overlaps`/`FreeSlots`).
```

- [ ] **Step 2: Update `PLAN.md`**

Find the §4.7 / §4.8 (conflict warning / free-time checker) checklist entries and mark them done, matching the file's existing convention (read it first).

- [ ] **Step 3: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` reports gofmt issues, run `cd backend && env -u GOROOT gofmt -w ./internal/...` and re-run.

- [ ] **Step 4: Commit**

```bash
git add docs/MEETINGS.md PLAN.md
git commit -m "docs(meetings): document §4.7-4.8 conflict warning + free-time checker"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** §4.7.1 (Overlaps + organizer included) → Tasks 1,3,4; §4.7.2 (warning layout) → Task 4 `formatConflictWarning`; §4.7.3 (non-blocking, change-time returns) → Task 4 `applyforce` + `medit:field:datetime`; §4.8.2 (params: ≥1 participant, range, duration presets) → Task 5; §4.8.3 (algorithm, weekday skip, 09:00–18:00) → Task 3 `FreeSlots`; §4.8.4 (results layout) → Task 5 `formatSlots`; §4.8.6 (no-slots message) → Task 5 `handleDuration`. REST → Task 7.
- **Out of scope (do not implement):** Google freebusy, configurable hours, bot create-from-slot, Mini App frontend wiring.
- **Type consistency:** `Conflict{Email,PersonName,MeetingName,Start,End}` and `FreeSlot{Day,Start,End,Mins}` are defined once in Task 3 and consumed verbatim in Tasks 4, 5, 7. `ListMeetingsOverlapping(ctx, emails, from, to)` defined in Task 2, called in Task 3. `MeetingConflicts(ctx, emails, start, end, excludeMeetingID)` / `FreeSlots(ctx, emails, from, to, durMins)` defined in Task 3, consumed in Tasks 4, 5, 7.
- **Verify-before-coding hooks** are flagged inline with `> Adapt:` notes — confirm field/method names against the real files before writing each task's code.
