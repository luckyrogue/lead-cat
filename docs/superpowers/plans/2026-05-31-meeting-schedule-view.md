# Employee schedule view (§4.6) + participant UNIQUE — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only `/schedule` bot flow to view any employee's meetings (by email or directory search, filtered by date window), and harden `meeting_participants` with a `UNIQUE (meeting_id, email)` constraint.

**Architecture:** A new pure-helper + FSM package `scheduleview` (mirrors `meetingedit`) drives the conversation and calls thin `application` query delegates (`SearchEmployeesGlobal`, `EmployeeSchedule`) backed by two new repo queries. Date windows are computed in Asia/Almaty by pure helpers. A migration adds the participant UNIQUE constraint + `ON CONFLICT DO NOTHING`.

**Tech Stack:** Go, go-telegram/bot, pgx, goose, redis, google/uuid.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-schedule-view-design.md`

**Conventions:** Run Go from `backend/` with `env -u GOROOT`. Module `github.com/luckyrogue/lead-cat`. Build check: `env -u GOROOT go build ./...`.

---

## Task 1: participant UNIQUE constraint + ON CONFLICT

**Files:**

- Create: `backend/migrations/20260531120000_meeting_participants_unique.sql`
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` (`AddParticipants`)

- [ ] **Step 1: Write the migration.** Create `backend/migrations/20260531120000_meeting_participants_unique.sql`:

```sql
-- +goose Up
ALTER TABLE meeting_participants ADD CONSTRAINT meeting_participants_unique UNIQUE (meeting_id, email);

-- +goose Down
ALTER TABLE meeting_participants DROP CONSTRAINT IF EXISTS meeting_participants_unique;
```

- [ ] **Step 2: Make AddParticipants idempotent.** In `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`, change the INSERT in `AddParticipants` to:

```go
func (s *Store) AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []MeetingParticipant) error {
	for _, p := range ps {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO meeting_participants (meeting_id, employee_id, email)
			VALUES ($1, $2, $3) ON CONFLICT (meeting_id, email) DO NOTHING`, meetingID, p.EmployeeID, p.Email); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 3: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
      Expected: clean. (The migration is applied at runtime via goose; no DB harness — build/vet is the gate.)

- [ ] **Step 4: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/migrations/20260531120000_meeting_participants_unique.sql backend/internal/infrastructure/persistence/postgres/meeting_repo.go && git commit -m "feat(meetings): UNIQUE(meeting_id,email) + ON CONFLICT on AddParticipants

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: repo — global employee search + schedule query

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/employee_repo.go`
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Add the global employee search.** In `backend/internal/infrastructure/persistence/postgres/employee_repo.go`, add (after `ListEmployees`):

```go
// SearchEmployeesGlobal finds directory entries across all workspaces whose name
// or email contains query (case-insensitive), capped at 20.
func (s *Store) SearchEmployeesGlobal(ctx context.Context, query string) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, full_name, email, dept, has_telegram
		FROM employees
		WHERE full_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%'
		ORDER BY full_name LIMIT 20`, query)
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
```

- [ ] **Step 2: Add the schedule query.** In `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`, add (after `ListMeetingsByOrganizerTelegram`):

```go
// ListScheduleForEmail returns the scheduled meetings in [from,to) where email is
// a participant or the organizer (by platform_users.email).
func (s *Store) ListScheduleForEmail(ctx context.Context, email string, from, to time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT DISTINCT `+meetingColsM+`
		FROM meetings m
		LEFT JOIN meeting_participants mp ON mp.meeting_id = m.id
		LEFT JOIN platform_users pu ON pu.id = m.organizer_user_id
		WHERE (mp.email = $1 OR pu.email = $1)
			AND m.status = 'scheduled' AND m.starts_at >= $2 AND m.starts_at < $3
		ORDER BY m.starts_at`, email, from, to)
}
```

(`queryMeetings` and `meetingColsM` already exist in this file; `queryMeetings` uses `scanMeeting` which scans the 14 `meetingColsM` columns. `SELECT DISTINCT` over exactly those columns is consistent.)

- [ ] **Step 3: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
      Expected: clean.

- [ ] **Step 4: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/persistence/postgres/ && git commit -m "feat(meetings): repo SearchEmployeesGlobal + ListScheduleForEmail

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: application — schedule query delegates

**Files:**

- Modify: `backend/internal/application/participants.go` (add the two delegates near `SearchEmployees`)

- [ ] **Step 1: Add the delegates.** In `backend/internal/application/participants.go`, add (after `SearchEmployees`):

```go
// SearchEmployeesGlobal finds directory entries across all workspaces (for the
// bot schedule view, which has no workspace context).
func (s *Services) SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error) {
	return s.Store.SearchEmployeesGlobal(ctx, query)
}

// EmployeeSchedule returns the scheduled meetings in [from,to) for an email
// (participant or organizer).
func (s *Services) EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]postgres.Meeting, error) {
	return s.Store.ListScheduleForEmail(ctx, email, from, to)
}
```

Add `"time"` to the import block of `participants.go` (it is not currently imported there).

- [ ] **Step 2: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/`
      Expected: clean.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/participants.go && git commit -m "feat(meetings): application SearchEmployeesGlobal + EmployeeSchedule

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: `scheduleview` — pure date/window/status helpers

**Files:**

- Create: `backend/internal/platform/scheduleview/parse.go`
- Test: `backend/internal/platform/scheduleview/parse_test.go`

- [ ] **Step 1: Write the failing test.** Create `backend/internal/platform/scheduleview/parse_test.go`:

```go
package scheduleview

import (
	"testing"
	"time"
)

func almaty(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestParseDate(t *testing.T) {
	loc := almaty(t)
	d, err := parseDate("2026-06-02", loc)
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2026 || d.Month() != 6 || d.Day() != 2 || d.Hour() != 0 {
		t.Fatalf("bad date: %v", d)
	}
	if _, err := parseDate("2026/06/02", loc); err == nil {
		t.Fatal("expected error for bad format")
	}
}

func TestParseRange(t *testing.T) {
	loc := almaty(t)
	from, to, err := parseRange("2026-06-01..2026-06-03", loc)
	if err != nil {
		t.Fatal(err)
	}
	// to is exclusive end = 2026-06-04 00:00
	if from.Day() != 1 || to.Day() != 4 {
		t.Fatalf("range from=%v to=%v", from, to)
	}
	if _, _, err := parseRange("2026-06-03..2026-06-01", loc); err == nil {
		t.Fatal("expected error: end before start")
	}
	if _, _, err := parseRange("2026-06-01", loc); err == nil {
		t.Fatal("expected error: missing range separator")
	}
}

func TestDayWindow(t *testing.T) {
	loc := almaty(t)
	now := time.Date(2026, 6, 2, 13, 0, 0, 0, loc)
	from, to, ok := dayWindow(now, "today", loc)
	if !ok || from.Day() != 2 || from.Hour() != 0 || to.Day() != 3 {
		t.Fatalf("today: from=%v to=%v ok=%v", from, to, ok)
	}
	from, to, ok = dayWindow(now, "tomorrow", loc)
	if !ok || from.Day() != 3 || to.Day() != 4 {
		t.Fatalf("tomorrow: from=%v to=%v", from, to)
	}
	from, _, ok = dayWindow(now, "upcoming", loc)
	if !ok || !from.Equal(now) {
		t.Fatalf("upcoming from should equal now, got %v", from)
	}
	if _, _, ok := dayWindow(now, "bogus", loc); ok {
		t.Fatal("unknown kind must return ok=false")
	}
}

func TestStatusEmoji(t *testing.T) {
	now := time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC)
	if statusEmoji(now.Add(time.Hour), now) != "🔜" {
		t.Fatal("future should be 🔜")
	}
	if statusEmoji(now.Add(-time.Hour), now) != "✅" {
		t.Fatal("past should be ✅")
	}
}
```

- [ ] **Step 2: Run, verify fail.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/scheduleview/ -v`
      Expected: FAIL — undefined parseDate/parseRange/dayWindow/statusEmoji.

- [ ] **Step 3: Implement.** Create `backend/internal/platform/scheduleview/parse.go`:

```go
// Package scheduleview drives the read-only /schedule bot flow (§4.6).
package scheduleview

import (
	"fmt"
	"strings"
	"time"
)

// parseDate parses "YYYY-MM-DD" as a start-of-day time in loc.
func parseDate(s string, loc *time.Location) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверная дата (ГГГГ-ММ-ДД)")
	}
	return d, nil
}

// parseRange parses "YYYY-MM-DD..YYYY-MM-DD" into [from, to) where to is the day
// AFTER the end date (exclusive). End must not precede start.
func parseRange(s string, loc *time.Location) (from, to time.Time, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "..", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("формат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД")
	}
	d1, err := parseDate(parts[0], loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	d2, err := parseDate(parts[1], loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if d2.Before(d1) {
		return time.Time{}, time.Time{}, fmt.Errorf("конец раньше начала")
	}
	return d1, d2.AddDate(0, 0, 1), nil
}

// dayWindow returns the [from,to) window for a preset kind. ok is false for an
// unknown kind.
func dayWindow(now time.Time, kind string, loc *time.Location) (from, to time.Time, ok bool) {
	base := now.In(loc)
	sod := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc)
	switch kind {
	case "today":
		return sod, sod.AddDate(0, 0, 1), true
	case "tomorrow":
		return sod.AddDate(0, 0, 1), sod.AddDate(0, 0, 2), true
	case "upcoming":
		return now, now.AddDate(1, 0, 0), true
	}
	return time.Time{}, time.Time{}, false
}

// statusEmoji marks an upcoming (🔜) vs past (✅) meeting relative to now.
func statusEmoji(startsAt, now time.Time) string {
	if startsAt.After(now) {
		return "🔜"
	}
	return "✅"
}
```

- [ ] **Step 4: Run, verify pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/scheduleview/ -v`
      Expected: all 4 PASS.

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/scheduleview/ && git commit -m "feat(meetings): scheduleview date/window/status helpers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: `scheduleview` — FSM service, state, sessions, render

**Files:**

- Create: `backend/internal/platform/scheduleview/state.go`
- Create: `backend/internal/platform/scheduleview/redis_sessions.go`
- Create: `backend/internal/platform/scheduleview/service.go`
- Test: `backend/internal/platform/scheduleview/service_test.go`

- [ ] **Step 1: state.go.** Create `backend/internal/platform/scheduleview/state.go`:

```go
package scheduleview

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step          string   `json:"step"`
	EmployeeEmail string   `json:"employee_email,omitempty"`
	AwaitingKind  string   `json:"awaiting_kind,omitempty"` // search | date | range
	Cands         []string `json:"cands,omitempty"`         // candidate emails (index → email)
}

const (
	awaitSearch = "search"
	awaitDate   = "date"
	awaitRange  = "range"
)

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

- [ ] **Step 2: redis_sessions.go.** Create `backend/internal/platform/scheduleview/redis_sessions.go`:

```go
package scheduleview

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisSessions stores FSM state in Redis with a TTL, keyed by Telegram ID.
type RedisSessions struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisSessions(rdb *redis.Client) *RedisSessions {
	return &RedisSessions{rdb: rdb, ttl: 15 * time.Minute}
}

func (r *RedisSessions) key(telegramID int64) string {
	return "sched:" + strconv.FormatInt(telegramID, 10)
}

func (r *RedisSessions) Get(ctx context.Context, telegramID int64) (*State, error) {
	raw, err := r.rdb.Get(ctx, r.key(telegramID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *RedisSessions) Set(ctx context.Context, telegramID int64, s State) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, r.key(telegramID), raw, r.ttl).Err()
}

func (r *RedisSessions) Del(ctx context.Context, telegramID int64) error {
	return r.rdb.Del(ctx, r.key(telegramID)).Err()
}
```

- [ ] **Step 3: Write the failing service test.** Create `backend/internal/platform/scheduleview/service_test.go`:

```go
package scheduleview

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeBackend struct {
	employees []postgres.Employee
	meetings  []postgres.Meeting
	gotEmail  string
	gotFrom   time.Time
	gotTo     time.Time
}

func (f *fakeBackend) SearchEmployeesGlobal(_ context.Context, query string) ([]postgres.Employee, error) {
	var out []postgres.Employee
	for _, e := range f.employees {
		if strings.Contains(strings.ToLower(e.FullName), strings.ToLower(query)) || strings.Contains(strings.ToLower(e.Email), strings.ToLower(query)) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeBackend) EmployeeSchedule(_ context.Context, email string, from, to time.Time) ([]postgres.Meeting, error) {
	f.gotEmail, f.gotFrom, f.gotTo = email, from, to
	return f.meetings, nil
}

type memSessions struct{ m map[int64]*State }

func newMemSessions() *memSessions { return &memSessions{m: map[int64]*State{}} }
func (s *memSessions) Get(_ context.Context, tg int64) (*State, error) { return s.m[tg], nil }
func (s *memSessions) Set(_ context.Context, tg int64, st State) error { c := st; s.m[tg] = &c; return nil }
func (s *memSessions) Del(_ context.Context, tg int64) error          { delete(s.m, tg); return nil }

func TestScheduleFlow_PickAndToday(t *testing.T) {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Almaty")
	be := &fakeBackend{
		employees: []postgres.Employee{{FullName: "Иван Иванов", Email: "ivan@corp.kz"}},
		meetings: []postgres.Meeting{{
			Name: "Планёрка", StartsAt: time.Date(2026, 6, 2, 14, 0, 0, 0, loc).UTC(), EndsAt: time.Date(2026, 6, 2, 15, 0, 0, 0, loc).UTC(),
		}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(70)

	if r := svc.Start(ctx, tg); !strings.Contains(r.Text, "email") {
		t.Fatalf("start: %+v", r)
	}
	if r, ok := svc.OnText(ctx, tg, "иван"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("search: %+v ok=%v", r, ok)
	}
	if r, ok := svc.OnCallback(ctx, tg, "sched:pick:0"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("pick: %+v ok=%v", r, ok)
	}
	r, ok := svc.OnCallback(ctx, tg, "sched:d:today")
	if !ok || !strings.Contains(r.Text, "Планёрка") {
		t.Fatalf("today list: %+v ok=%v", r, ok)
	}
	if be.gotEmail != "ivan@corp.kz" {
		t.Fatalf("schedule queried for %q", be.gotEmail)
	}
	if !be.gotTo.After(be.gotFrom) {
		t.Fatalf("window not valid: from=%v to=%v", be.gotFrom, be.gotTo)
	}
}

func TestScheduleFlow_RawEmailThenDate(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{}
	svc := New(be, newMemSessions())
	const tg = int64(71)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "bob@corp.kz") // no directory matches, raw email candidate
	svc.OnCallback(ctx, tg, "sched:pick:0")
	svc.OnCallback(ctx, tg, "sched:d:date")
	if r, ok := svc.OnText(ctx, tg, "2026-06-02"); !ok || !strings.Contains(r.Text, "Расписание") {
		t.Fatalf("date list: %+v ok=%v", r, ok)
	}
	if be.gotEmail != "bob@corp.kz" {
		t.Fatalf("schedule queried for %q", be.gotEmail)
	}
}

func TestScheduleFlow_BadDate(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{}
	svc := New(be, newMemSessions())
	const tg = int64(72)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "bob@corp.kz")
	svc.OnCallback(ctx, tg, "sched:pick:0")
	svc.OnCallback(ctx, tg, "sched:d:date")
	if r, ok := svc.OnText(ctx, tg, "nope"); !ok || !strings.Contains(r.Text, "дата") {
		t.Fatalf("expected date error, got %+v ok=%v", r, ok)
	}
}
```

- [ ] **Step 4: Run, verify fail.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/scheduleview/ -run TestScheduleFlow -v`
      Expected: FAIL — undefined New/Service.

- [ ] **Step 5: Implement service.go.** Create `backend/internal/platform/scheduleview/service.go`:

```go
package scheduleview

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the FSM needs (satisfied by *application.Services).
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]postgres.Meeting, error)
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

// Start handles /schedule: prompts for the employee to look up.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepAwait, AwaitingKind: awaitSearch})
	return Reply{Text: "Чьё расписание показать? Введи email сотрудника или часть имени:"}
}

// OnCallback handles sched:* taps. The bool is false for non-sched data.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "sched:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "sched:pick:")), true
	case data == "sched:periods":
		return s.periods(ctx, telegramID), true
	case data == "sched:back":
		return s.Start(ctx, telegramID), true
	case strings.HasPrefix(data, "sched:d:"):
		return s.period(ctx, telegramID, strings.TrimPrefix(data, "sched:d:")), true
	}
	return Reply{}, false
}

// OnText feeds free text into the active awaiting state. The bool is false when
// there is no active schedule session.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwait {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.AwaitingKind {
	case awaitSearch:
		return s.search(ctx, telegramID, st, text), true
	case awaitDate:
		d, perr := parseDate(text, almaty())
		if perr != nil {
			return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
		}
		return s.list(ctx, st, d, d.AddDate(0, 0, 1), text), true
	case awaitRange:
		from, to, perr := parseRange(text, almaty())
		if perr != nil {
			return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
		}
		return s.list(ctx, st, from, to, text), true
	}
	return Reply{}, false
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй ещё раз:"}
	}
	var cands []string
	var rows [][]Button
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("sched:pick:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if addr, perr := mail.ParseAddress(query); perr == nil {
		email := strings.ToLower(addr.Address)
		if !seen[email] {
			rows = append(rows, []Button{{Text: "Расписание " + email, Data: fmt.Sprintf("sched:pick:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: "Ничего не найдено. Введи корректный email или часть имени:"}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Выбери сотрудника:", Keyboard: rows}
}

func (s *Service) pick(ctx context.Context, telegramID int64, idxStr string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	email, ok := indexInto(st.Cands, idxStr)
	if !ok {
		return Reply{Text: "Не найдено, начни заново: /schedule"}
	}
	st.EmployeeEmail = email
	st.AwaitingKind = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return periodReply(email, true)
}

func (s *Service) periods(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	return periodReply(st.EmployeeEmail, true)
}

func (s *Service) period(ctx context.Context, telegramID int64, kind string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	switch kind {
	case "date":
		st.AwaitingKind = awaitDate
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: "Введи дату ГГГГ-ММ-ДД:"}
	case "range":
		st.AwaitingKind = awaitRange
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: "Введи диапазон ГГГГ-ММ-ДД..ГГГГ-ММ-ДД:"}
	}
	from, to, ok := dayWindow(time.Now(), kind, almaty())
	if !ok {
		return Reply{}
	}
	return s.list(ctx, st, from, to, periodLabel(kind))
}

func (s *Service) list(ctx context.Context, st *State, from, to time.Time, period string) Reply {
	ms, err := s.backend.EmployeeSchedule(ctx, st.EmployeeEmail, from, to)
	if err != nil {
		return Reply{Text: "Не удалось получить расписание, попробуй позже."}
	}
	text := scheduleText(st.EmployeeEmail, period, ms, time.Now(), almaty())
	return Reply{Text: text, Keyboard: [][]Button{{{Text: "⬅ Периоды", Data: "sched:periods"}}}}
}

func periodReply(email string, edit bool) Reply {
	return Reply{
		Text: "Расписание " + email + ". Выбери период:",
		Edit: edit,
		Keyboard: [][]Button{
			{{Text: "Сегодня", Data: "sched:d:today"}, {Text: "Завтра", Data: "sched:d:tomorrow"}},
			{{Text: "Все предстоящие", Data: "sched:d:upcoming"}},
			{{Text: "Конкретная дата", Data: "sched:d:date"}, {Text: "Диапазон", Data: "sched:d:range"}},
			{{Text: "⬅ Другой сотрудник", Data: "sched:back"}},
		},
	}
}

func periodLabel(kind string) string {
	switch kind {
	case "today":
		return "сегодня"
	case "tomorrow":
		return "завтра"
	case "upcoming":
		return "все предстоящие"
	}
	return kind
}

func scheduleText(email, period string, ms []postgres.Meeting, now time.Time, loc *time.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Расписание %s: %s\n", email, period)
	if len(ms) == 0 {
		b.WriteString("Встреч нет.")
		return b.String()
	}
	for _, m := range ms {
		s := m.StartsAt.In(loc)
		e := m.EndsAt.In(loc)
		fmt.Fprintf(&b, "%s «%s» — %s %s–%s\n", statusEmoji(m.StartsAt, now), m.Name, s.Format("02.01.2006"), s.Format("15:04"), e.Format("15:04"))
	}
	return b.String()
}

func indexInto(list []string, idxStr string) (string, bool) {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(list) {
		return "", false
	}
	return list[i], true
}

func almaty() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.UTC
	}
	return loc
}
```

Add the missing `stepAwait` constant to `state.go`'s const block:

```go
const (
	stepAwait   = "await"
	awaitSearch = "search"
	awaitDate   = "date"
	awaitRange  = "range"
)
```

- [ ] **Step 6: Run tests, verify pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/scheduleview/ -v && env -u GOROOT go build ./...`
      Expected: `TestParse*`, `TestDayWindow`, `TestStatusEmoji`, `TestScheduleFlow_*` PASS; build OK.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/scheduleview/ && git commit -m "feat(meetings): scheduleview FSM service

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: wire `/schedule` into MultiHandler

**Files:**

- Modify: `backend/internal/infrastructure/telegram/multitenant.go`

- [ ] **Step 1: Add the import + field.** In `backend/internal/infrastructure/telegram/multitenant.go`, add the import `"github.com/luckyrogue/lead-cat/internal/platform/scheduleview"`, and add a field to `MultiHandler`:

```go
type MultiHandler struct {
	store     *postgres.Store
	executor  *scenario_executor.Executor
	registrar *botreg.Service
	settings  *botsettings.Service
	editor    *meetingedit.Service
	schedule  *scheduleview.Service
	log       *zap.Logger
}
```

- [ ] **Step 2: Build it in NewMultiHandler.** `NewMultiHandler` already receives `editorBackend meetingedit.Backend` (which is `*application.Services`). The same `*application.Services` satisfies `scheduleview.Backend`. Change the signature's backend param name to a neutral type and build both. Replace the `editorBackend meetingedit.Backend` parameter with `app appBackend` where `appBackend` is a small composite interface, OR (simpler) keep `editorBackend meetingedit.Backend` and add a second param. Use this concrete approach — change the param to accept the concrete needs via a tiny local interface:

Add near the top of `multitenant.go` (after imports):

```go
// botBackend is the application surface the bot FSMs need (satisfied by *application.Services).
type botBackend interface {
	meetingedit.Backend
	scheduleview.Backend
}
```

Change `NewMultiHandler`'s signature param from `editorBackend meetingedit.Backend` to `backend botBackend`, and build both services:

```go
func NewMultiHandler(store *postgres.Store, cipher *crypto.TokenCipher, b *bot.Bot, rdb *redis.Client, adminIDs []int64, otpLog bool, backend botBackend, log *zap.Logger) *MultiHandler {
	otp := platformauth.NewOTP(rdb, log, otpLog)
	registrar := botreg.New(store, otp, botreg.NewRedisSessions(rdb), adminIDs)
	settings := botsettings.New(store)
	editor := meetingedit.New(backend, meetingedit.NewRedisSessions(rdb))
	schedule := scheduleview.New(backend, scheduleview.NewRedisSessions(rdb))
	return &MultiHandler{
		store:     store,
		executor:  scenario_executor.New(store, cipher, b, log),
		registrar: registrar,
		settings:  settings,
		editor:    editor,
		schedule:  schedule,
		log:       log,
	}
}
```

(The `main.go` call site passes `services` positionally, which satisfies `botBackend` — no main.go change needed since the arg is already `*application.Services`.)

- [ ] **Step 3: Route /schedule + sched callbacks + text.** In `Handle`, add a `/schedule` case in the `switch cmd` block (after `/edit`):

```go
	case "/schedule":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			h.sendSchedReply(ctx, b, chatID, 0, h.schedule.Start(ctx, from.ID))
		}
```

In the no-command text branch, add schedule after the editor (and add `return` after the editor send so the chain is exclusive):

```go
			if reply, handled := h.editor.OnText(ctx, from.ID, text); handled {
				h.sendEditorReply(ctx, b, chatID, 0, reply)
				return
			}
			if reply, handled := h.schedule.OnText(ctx, from.ID, text); handled {
				h.sendSchedReply(ctx, b, chatID, 0, reply)
			}
```

In `handleCallback`, add a `sched:` branch (after the existing `medit:` branch):

```go
	if strings.HasPrefix(cq.Data, "sched:") {
		if reply, handled := h.schedule.OnCallback(ctx, cq.From.ID, cq.Data); handled && cq.Message.Message != nil {
			h.sendSchedReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
```

- [ ] **Step 4: Add the send helper + markup converter.** Next to `sendEditorReply`/`toMeditMarkup`, add:

```go
func (h *MultiHandler) sendSchedReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply scheduleview.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toSchedMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toSchedMarkup(rows [][]scheduleview.Button) models.InlineKeyboardMarkup {
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

- [ ] **Step 5: Build + vet + test.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test ./internal/infrastructure/telegram/ ./internal/platform/scheduleview/`
      Expected: clean; tests PASS. Fix any unused-import error the compiler reports.

- [ ] **Step 6: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/telegram/multitenant.go && git commit -m "feat(meetings): wire /schedule bot flow

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: full verification + docs

**Files:**

- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Run the full suite.** From the repo root: `make test && make lint && make build`. (Fallback: `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && env -u GOROOT go build ./...`.) ALSO run `cd backend && gofmt -l .` — if it lists any file, `gofmt -w` it and re-run lint. If a real failure occurs, STOP and report.

- [ ] **Step 2: Document the feature.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Participant management (§4.3, done)" line, add:

```markdown
> **Employee schedule view (§4.6, done):** a read-only `/schedule` bot flow — look up an employee (global directory search or raw email), then view their scheduled meetings (where they participate or organize) filtered by Сегодня/Завтра/Все предстоящие/конкретная дата/диапазон. Day windows are computed in Asia/Almaty; rows show 🔜 upcoming / ✅ past. Also: `meeting_participants` now has a `UNIQUE (meeting_id, email)` constraint (AddParticipants uses `ON CONFLICT DO NOTHING`). Recurring-series editing (§4.4.2) and employee-CSV seeding remain planned.
```

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §4.6 schedule view + participant UNIQUE

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** Part A UNIQUE + ON CONFLICT (Task 1) · global employee search + schedule query (Task 2) · application delegates (Task 3) · pure date/window/status helpers (Task 4) · FSM search→pick→period→list incl. date/range (Task 5) · wiring /schedule + sched callbacks + text chain + gate (Task 6) · testing (Tasks 4,5,7) · docs (Task 7). Out-of-scope (§4.4.2, CSV seeding) recorded in spec + Task 7 note. All covered.
- **Type consistency:** `scheduleview.Backend{SearchEmployeesGlobal, EmployeeSchedule}` (Task 5) matches the `*application.Services` delegates (Task 3) and the repo methods (Task 2). `Reply`/`Button` (Task 5) consumed by `sendSchedReply`/`toSchedMarkup` (Task 6). `dayWindow`/`parseDate`/`parseRange`/`statusEmoji` (Task 4) used in `service.go` (Task 5). `botBackend` composite (Task 6) is satisfied by `*application.Services` (already passed to `NewMultiHandler`). `meetingColsM` (existing) reused by `ListScheduleForEmail` (Task 2).
- **No placeholders:** every code/command step is concrete. The one conditional (remove unused imports if the compiler flags) in Task 6 is explicit and build-checked.
- **Known small dup:** `toSchedMarkup`/`sendSchedReply` mirror `toMeditMarkup`/`sendEditorReply` (third inline-markup converter). Acceptable per KISS / consistency with the existing wiring; a shared `botkb` extraction is a possible follow-up, not in scope here.

```

```
