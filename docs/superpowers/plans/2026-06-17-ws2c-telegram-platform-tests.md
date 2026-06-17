# WS2c — Telegram Platform Package Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unit-test the Telegram platform packages with in-memory fakes: `botsettings`, `botreg`, the `meeting_notifier` pure message builders, and the `checker`/`meetingedit`/`scheduleview` conversational services.

**Architecture:** White-box `_test.go` files in each package define small fakes implementing that package's own port interfaces (with compile-time `var _` assertions), then drive the Service entry points (`Start`/`OnText`/`OnCallback`/`Settings`/`Toggle`) and assert observable effects (returned text/buttons, recorded fake calls, persisted values). Pure parsers and message builders are tested directly. DB-free.

**Tech Stack:** Go 1.26, stdlib `testing`, in-memory fakes. Runs under the CI gate (`go test -race` + `golangci-lint`).

**Standing constraints (every task):** work on `main`, no branches; commit per task; stage only listed paths (never `git add -A`); `git status` before staging; run Go with `env -u GOROOT`; commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; do NOT touch `.github/workflows/_build.yml` or production code; ignore stale IDE diagnostics, trust `go test`/`golangci-lint`.

---

## Verified interfaces & entry points (read from source)

- **botsettings:** `store{GetBotUserByTelegramID(ctx,tid)(postgres.BotUser,error); SetReminderMinutes(ctx,tid,csv)error}`; `New(store)*Service`; `Settings(ctx,tid)(string,[][]Button,error)`; `Toggle(ctx,tid,minutes)(string,[][]Button,error)`; exported pure `Parse(csv string)[]int`, `Format(mins []int)string`; `Button{Text,Data}`; `Intervals` = {10,15,30,60,120,1440}.
- **botreg:** `userStore{GetBotUserByTelegramID; GetBotUserByEmail; CreateBotUser(ctx,tid,fullName,email,role)(postgres.BotUser,error)}`; `sessions{Get(ctx,tid)(*State,error); Set(ctx,tid,State)error; Del(ctx,tid)error}`; `State{Step,FullName,Email}`; `New(users,sess,adminIDs []int64)*Service`; `Start(ctx,tid)string`; `OnText(ctx,tid,text)(string,bool)`. Flow: Start (registered→"С возвращением"; else set Step=awaiting_name, ask ФИО) → OnText awaiting_name (empty→reask; else save name, Step=awaiting_email) → OnText awaiting_email (bad email→reask; existing email→reject; else create user with role admin if tid∈admins else user, Del session, "Готово, <name>!").
- **meeting_notifier (pure):** `buildMessage(name,meetLink,start,end,loc)`, `buildUpdatedMessage(...)`, `buildRemovedMessage(name,start,loc)`, `buildCancelledMessage(name,start,loc)`, `buildEventMessage(header,...)`, `tzLabel(t)`. All unexported → white-box `package meeting_notifier`.
- **checker:** `Backend{SearchEmployeesGlobal(ctx,query)([]postgres.Employee,error); FreeSlots(ctx,emails,from,to,durMins)([]application.FreeSlot,error)}`; `sessions{Get(*State)/Set(State)/Del}`; `New(backend,sess)*Service`; `Start(ctx,tid)Reply`; `OnText(ctx,tid,text)(Reply,bool)`; `OnCallback(ctx,tid,data)(Reply,bool)`; `Reply{Text string; Keyboard [][]Button}`; `Button{Text,Data}`; unexported `parseRange(s,loc)(from,to,err)`, `dayLabel(day,loc)`.
- **scheduleview:** `Backend{SearchEmployeesGlobal; EmployeeSchedule(ctx,email,from,to)([]postgres.Meeting,error)}`; same `sessions`/`Reply`/`Button`; `Start`/`OnText`/`OnCallback` same shapes; unexported `parseDate`, `parseRange` (note: `to` is `d2.AddDate(0,0,1)`), `dayWindow(now,kind,loc)(from,to,ok)` (kinds today/tomorrow/upcoming), `statusEmoji(startsAt,now)`.
- **meetingedit:** `Backend{ListEditableMeetings(ctx,tid)([]postgres.MeetingWithTZ,error); UpdateMeeting(...); UpdateSeries(...); ListParticipants; SearchEmployees; AddParticipant; RemoveParticipant; CancelMeeting; CancelSeries; MeetingUpdateConflicts}`; same `sessions`; `New(backend,sess)*Service`; `Start(ctx,tid)Reply` (lists editable meetings as `medit:pick:<id>` buttons, or a friendly empty/error message).

Each package's `state.go` defines its `State`/`Step` constants and (for the three services) `Reply`/`Button`. **Read the target package's `state.go` before writing a flow test** to get exact `Step` constants and `State` fields.

---

## File Structure

| File | Task |
|---|---|
| `internal/platform/botsettings/settings_test.go` | 1 |
| `internal/platform/botreg/service_test.go` | 2 |
| `internal/platform/meeting_notifier/message_test.go` | 3 |
| `internal/platform/checker/service_test.go` | 4 |
| `internal/platform/scheduleview/service_test.go` | 5 |
| `internal/platform/meetingedit/service_test.go` | 6 |

---

### Task 1: botsettings (Parse/Format/toggle + Service)

**Files:** Create `apps/backend/internal/platform/botsettings/settings_test.go`

- [ ] **Step 1: Write the tests**

```go
package botsettings

import (
	"context"
	"reflect"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestParse_DedupSortDropInvalid(t *testing.T) {
	got := Parse(" 60, 15 ,15, x, 30,, 60 ")
	want := []int{15, 30, 60}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %v, want %v", got, want)
	}
	if len(Parse("")) != 0 {
		t.Fatalf("empty csv should parse to empty")
	}
}

func TestFormat_SortedCompactCSV(t *testing.T) {
	if got := Format([]int{60, 15, 15, 30}); got != "15,30,60" {
		t.Fatalf("Format = %q", got)
	}
	if got := Format(nil); got != "" {
		t.Fatalf("Format(nil) = %q, want empty", got)
	}
}

func TestToggle_AddAndRemove(t *testing.T) {
	on := toggle([]int{15, 60}, 30)
	if got := format(on); got != "15,30,60" {
		t.Fatalf("toggle add = %q", got)
	}
	off := toggle([]int{15, 30, 60}, 30)
	if got := format(off); got != "15,60" {
		t.Fatalf("toggle remove = %q", got)
	}
}

type fakeStore struct {
	user    postgres.BotUser
	savedCSV string
	saveErr error
}

func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	return f.user, nil
}
func (f *fakeStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	f.savedCSV = csv
	return f.saveErr
}

var _ store = (*fakeStore)(nil)

func TestService_Toggle_Persists(t *testing.T) {
	fs := &fakeStore{user: postgres.BotUser{ReminderMinutes: "15"}}
	s := New(fs)
	text, kb, err := s.Toggle(context.Background(), 1, 60)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if fs.savedCSV != "15,60" {
		t.Fatalf("persisted CSV = %q, want 15,60", fs.savedCSV)
	}
	if text == "" || len(kb) == 0 {
		t.Fatalf("expected rendered text + keyboard")
	}
}

func TestService_Settings_RendersCurrent(t *testing.T) {
	fs := &fakeStore{user: postgres.BotUser{ReminderMinutes: "30"}}
	text, kb, err := New(fs).Settings(context.Background(), 1)
	if err != nil || text == "" || len(kb) == 0 {
		t.Fatalf("settings: %v text=%q kb=%d", err, text, len(kb))
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/platform/botsettings/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/botsettings/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/botsettings/settings_test.go
git commit -m "$(cat <<'EOF'
test(botsettings): Parse/Format/toggle round-trips + Service persistence

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: botreg registration flow

**Files:** Create `apps/backend/internal/platform/botreg/service_test.go`

- [ ] **Step 1: Write the tests**

```go
package botreg

import (
	"context"
	"errors"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeUsers struct {
	byTelegram map[int64]postgres.BotUser
	byEmail    map[string]postgres.BotUser
	created    []postgres.BotUser
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byTelegram: map[int64]postgres.BotUser{}, byEmail: map[string]postgres.BotUser{}}
}
func (f *fakeUsers) GetBotUserByTelegramID(_ context.Context, tid int64) (postgres.BotUser, error) {
	if u, ok := f.byTelegram[tid]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeUsers) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeUsers) CreateBotUser(_ context.Context, tid int64, fullName, email, role string) (postgres.BotUser, error) {
	u := postgres.BotUser{TelegramID: tid, FullName: fullName, Email: email, Role: role}
	f.created = append(f.created, u)
	f.byTelegram[tid] = u
	return u, nil
}

var _ userStore = (*fakeUsers)(nil)

type fakeSessions struct {
	m map[int64]State
}

func newFakeSessions() *fakeSessions { return &fakeSessions{m: map[int64]State{}} }
func (f *fakeSessions) Get(_ context.Context, tid int64) (*State, error) {
	if s, ok := f.m[tid]; ok {
		c := s
		return &c, nil
	}
	return nil, nil
}
func (f *fakeSessions) Set(_ context.Context, tid int64, s State) error { f.m[tid] = s; return nil }
func (f *fakeSessions) Del(_ context.Context, tid int64) error          { delete(f.m, tid); return nil }

var _ sessions = (*fakeSessions)(nil)

func TestRegistration_HappyPath_NonAdmin(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	s := New(users, sess, nil)
	ctx := context.Background()

	if msg := s.Start(ctx, 100); msg == "" {
		t.Fatal("Start should prompt for name")
	}
	if _, ok := sess.m[100]; !ok {
		t.Fatal("Start should set a session")
	}
	if _, ok := s.OnText(ctx, 100, "Иванов Иван"); !ok {
		t.Fatal("name step should handle text")
	}
	if _, ok := s.OnText(ctx, 100, "ivan@corp.io"); !ok {
		t.Fatal("email step should handle text")
	}
	if len(users.created) != 1 {
		t.Fatalf("want 1 user created, got %d", len(users.created))
	}
	if users.created[0].Role != "user" || users.created[0].FullName != "Иванов Иван" || users.created[0].Email != "ivan@corp.io" {
		t.Fatalf("created user wrong: %+v", users.created[0])
	}
	if _, ok := sess.m[100]; ok {
		t.Fatal("session should be deleted after registration")
	}
}

func TestRegistration_AdminRole(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	s := New(users, sess, []int64{42})
	ctx := context.Background()
	s.Start(ctx, 42)
	s.OnText(ctx, 42, "Admin User")
	s.OnText(ctx, 42, "admin@corp.io")
	if len(users.created) != 1 || users.created[0].Role != "admin" {
		t.Fatalf("admin id should create admin role: %+v", users.created)
	}
}

func TestStart_AlreadyRegistered(t *testing.T) {
	users := newFakeUsers()
	users.byTelegram[7] = postgres.BotUser{TelegramID: 7}
	sess := newFakeSessions()
	s := New(users, sess, nil)
	msg := s.Start(context.Background(), 7)
	if msg == "" {
		t.Fatal("should greet returning user")
	}
	if _, ok := sess.m[7]; ok {
		t.Fatal("should not start a session for a registered user")
	}
}

func TestRegistration_RejectsBadEmailAndDuplicate(t *testing.T) {
	users := newFakeUsers()
	users.byEmail["taken@corp.io"] = postgres.BotUser{Email: "taken@corp.io"}
	sess := newFakeSessions()
	s := New(users, sess, nil)
	ctx := context.Background()
	s.Start(ctx, 5)
	s.OnText(ctx, 5, "Some Name")
	if _, ok := s.OnText(ctx, 5, "not-an-email"); !ok {
		t.Fatal("bad email should be handled")
	}
	if len(users.created) != 0 {
		t.Fatal("bad email must not create a user")
	}
	if _, ok := s.OnText(ctx, 5, "taken@corp.io"); !ok {
		t.Fatal("duplicate email should be handled")
	}
	if len(users.created) != 0 {
		t.Fatal("duplicate email must not create a user")
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/platform/botreg/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/botreg/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/botreg/service_test.go
git commit -m "$(cat <<'EOF'
test(botreg): /start registration flow, admin role, dedupe, bad-email rejection

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: meeting_notifier message builders

**Files:** Create `apps/backend/internal/platform/meeting_notifier/message_test.go`

- [ ] **Step 1: Write the tests**

```go
package meeting_notifier

import (
	"strings"
	"testing"
	"time"
)

func almaty(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatalf("load loc: %v", err)
	}
	return loc
}

func TestBuildMessage_WithAndWithoutLink(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)  // 10:00 Almaty
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)   // 10:30
	with := buildMessage("Sync", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"📅 Новая встреча", "«Sync»", "01.06.2026", "10:00–10:30", "UTC+5", "🔗 https://meet.google.com/abc"} {
		if !strings.Contains(with, want) {
			t.Fatalf("missing %q in:\n%s", want, with)
		}
	}
	without := buildMessage("Sync", "", start, end, loc)
	if strings.Contains(without, "🔗") {
		t.Fatalf("no link icon expected without meet link:\n%s", without)
	}
}

func TestBuildUpdatedRemovedCancelled(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)
	if !strings.Contains(buildUpdatedMessage("S", "", start, end, loc), "✏️ Встреча изменена") {
		t.Fatal("updated header")
	}
	if !strings.Contains(buildRemovedMessage("S", start, loc), "➖") {
		t.Fatal("removed header")
	}
	if !strings.Contains(buildCancelledMessage("S", start, loc), "❌ Встреча отменена") {
		t.Fatal("cancelled header")
	}
}

func TestTzLabel(t *testing.T) {
	whole := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", 5*3600))
	if got := tzLabel(whole); got != "UTC+5" {
		t.Fatalf("tzLabel whole = %q", got)
	}
	half := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", 5*3600+1800))
	if got := tzLabel(half); got != "UTC+5:30" {
		t.Fatalf("tzLabel half = %q", got)
	}
	neg := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", -4*3600))
	if got := tzLabel(neg); got != "UTC-4" {
		t.Fatalf("tzLabel neg = %q", got)
	}
}
```

NOTE: `buildMessage` renders the time range with an en-dash `–` (U+2013), as in `s.Format("15:04")+"–"+e.Format("15:04")` → `10:00–10:30`. Use the en-dash in the assertion exactly (copy it from `message.go`).

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/platform/meeting_notifier/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/meeting_notifier/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/meeting_notifier/message_test.go
git commit -m "$(cat <<'EOF'
test(meeting_notifier): notification message builders + tzLabel

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: checker service + parse

**Files:** Create `apps/backend/internal/platform/checker/service_test.go`

**First:** read `internal/platform/checker/state.go` for the exact `State` fields and `Step` constants (`stepParticipants`, `stepRange`) and the `Reply`/`Button` shapes.

- [ ] **Step 1: Write fakes + Start + OnText(search) + parse tests**

Write a `fakeBackend` implementing checker's `Backend` (`SearchEmployeesGlobal`, `FreeSlots`) returning canned data, and a map-backed `fakeSessions` (`Get`/`Set`/`Del` with `*State`/`State`), each with a `var _ Backend = ...` / `var _ sessions = ...` assertion. Then:

- `TestStart_SetsParticipantsStep`: `Start` returns a non-empty `Reply.Text` and the session is set to the participants step.
- `TestOnText_Search_ReturnsMatches`: with the session at the participants step, `OnText` with a query calls `SearchEmployeesGlobal` and returns a `Reply` reflecting the canned employees (buttons or text). Assert the backend was called and a `Reply` with `ok=true` is returned.
- `TestParseRange_ValidAndErrors`: `parseRange("2026-06-01..2026-06-03", loc)` returns from<to, nil; malformed (`"2026-06-01"` no `..`), bad date, and reversed range each return an error. Use `time.UTC` (or build a fixed loc) for `loc`.
- `TestDayLabel`: `dayLabel` renders the RU weekday + `DD.MM` for a known date.
- `TestOnText_NoSession_NotHandled`: `OnText` with no session returns `(Reply{}, false)`.

Drive `OnCallback` (the `chk:` slot-selection branch) only as far as you can trace cleanly from `state.go`/`service.go`; cover the reachable transitions and note any branch you intentionally left for a future pass. Do not fabricate callback data formats — copy them from `service.go`.

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/platform/checker/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/checker/...` — `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/checker/service_test.go
git commit -m "$(cat <<'EOF'
test(checker): start/search flow + parseRange/dayLabel

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: scheduleview service + parse

**Files:** Create `apps/backend/internal/platform/scheduleview/service_test.go`

**First:** read `internal/platform/scheduleview/state.go` for `State` fields, `Step`/`AwaitingKind` constants (`stepAwait`, `awaitSearch`), and `Reply`/`Button`.

- [ ] **Step 1: Write fakes + Start + OnText + parse tests**

`fakeBackend` implementing scheduleview's `Backend` (`SearchEmployeesGlobal`, `EmployeeSchedule`) + map-backed `fakeSessions`, with `var _` assertions. Then:

- `TestStart_SetsAwaitSearch`: `Start` returns a prompt and sets the session to the await/search step.
- `TestOnText_Search`: at await/search, `OnText` with a query calls `SearchEmployeesGlobal` and returns a `Reply` (ok=true).
- `TestParseDate_ValidAndError`: `parseDate("2026-06-01", loc)` ok; `parseDate("nope", loc)` errors.
- `TestParseRange_EndExclusive`: `parseRange("2026-06-01..2026-06-03", loc)` returns `to == 2026-06-04 00:00` (note the `AddDate(0,0,1)`); reversed range errors.
- `TestDayWindow`: `dayWindow(now,"today",loc)` / `"tomorrow"` / `"upcoming"` return the right bounds and `ok=true`; unknown kind returns `ok=false`.
- `TestStatusEmoji`: future start → `🔜`; past → `✅`.

Drive `OnCallback` (`sched:pick:`/`sched:periods`/`sched:back`/`sched:d:`) as far as cleanly traceable; copy data prefixes from `service.go`.

- [ ] **Step 2: Run + lint** (same commands, `scheduleview` path) — all PASS, `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/scheduleview/service_test.go
git commit -m "$(cat <<'EOF'
test(scheduleview): start/search flow + parseDate/parseRange/dayWindow/statusEmoji

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: meetingedit service (Start + reachable flow)

**Files:** Create `apps/backend/internal/platform/meetingedit/service_test.go`

**First:** read `internal/platform/meetingedit/state.go` and `service.go` for the `State`/`Step` constants, `Reply`/`Button`, and the `medit:` callback data formats.

- [ ] **Step 1: Write the fake + Start + reachable transitions**

Write a `fakeBackend` implementing **all ten** `meetingedit.Backend` methods. Most return canned/empty values; record calls for the mutating ones (`UpdateMeeting`, `AddParticipant`, `RemoveParticipant`, `CancelMeeting`, etc.). Add `var _ Backend = (*fakeBackend)(nil)` and a map-backed `fakeSessions` with `var _ sessions = ...`. Then:

- `TestStart_ListsEditableMeetings`: `ListEditableMeetings` returns 2 meetings → `Start` returns a `Reply` whose `Keyboard` has a `medit:pick:<id>` button per meeting.
- `TestStart_EmptyAndError`: zero meetings → friendly "no meetings" text; backend error → friendly error text (no panic).
- `TestPick_LoadsEditMenu` (if cleanly traceable): `OnCallback` with `medit:pick:<id>` advances state and returns the field-selection `Reply`. Use a real `uuid` for `<id>` and seed the fake's `ListEditableMeetings`/`ListParticipants` accordingly.

meetingedit is the most branch-heavy package. Cover `Start` (all three outcomes) fully; for the deeper pick→field→submit→confirm chain, cover the transitions you can drive cleanly from `state.go`/`service.go` and explicitly note (in the test file as a comment, and in your report) which branches are left for a later pass. Do NOT fabricate callback formats or state fields — copy them from source. If the flow proves too entangled to drive meaningfully via the public methods, report DONE_WITH_CONCERNS describing the blocker rather than forcing brittle coverage.

- [ ] **Step 2: Run + lint** (same commands, `meetingedit` path) — all PASS, `0 issues.`

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/meetingedit/service_test.go
git commit -m "$(cat <<'EOF'
test(meetingedit): Start (list/empty/error) + reachable edit transitions

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Full verification

**Files:** none.

- [ ] **Step 1:** `cd apps/backend && env -u GOROOT go test -race ./internal/platform/...` — all PASS.
- [ ] **Step 2:** `cd apps/backend && env -u GOROOT go test ./...` — module-wide green (Docker permitting for the postgres package).
- [ ] **Step 3:** `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...` — `0 issues.`
- [ ] **Step 4:** `git status --short` — clean (only human's pre-existing items, if any).
- [ ] **Step 5 (informational):** after the human pushes, confirm the CI run is green (`gh run watch`).

---

## Notes on execution order
Tasks 1–3 (botsettings, botreg, message builders) are fully specified and independent — do them first. Tasks 4–6 (the conversational services) each require reading that package's `state.go`/`service.go` first; they're independent of each other. Task 7 verifies the whole. Each task is its own package, so there is no cross-task shared fixture.
