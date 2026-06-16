# NL Scheduling Assistant (Phase 1: read-only) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a registered Telegram user type natural language ("когда у Миа и Алекса есть общий час на следующей неделе?") and get an answer, by running a Claude tool-use loop over the existing read-only scheduling services.

**Architecture:** A provider-neutral `Planner` port in `application`, implemented by an `anthropic-sdk-go` adapter in `infrastructure/llm/anthropic`. A new `platform/scheduler_agent` package owns the agentic loop, a small read-only tool surface that maps onto existing `Services` methods (`SearchEmployeesGlobal`, `FreeSlots`, `MeetingConflicts`), and a Redis-backed per-`telegramID` transcript — mirroring the existing `checker` package exactly. The bot dispatcher routes free text (text that isn't a `/command` and isn't claimed by another stateful flow) to the agent. **No writes, no booking** — that is Phase 2.

**Tech Stack:** Go 1.26, `github.com/anthropics/anthropic-sdk-go`, `claude-opus-4-8` with adaptive thinking, `go-telegram/bot`, Redis (go-redis), Postgres. Clean architecture (depguard forbids `application → infrastructure`).

**Scope boundary (read before starting):** This plan is read-only. The tools never mutate state. Caller-identity resolution, the `book_meeting` write tool, and the inline-button confirm card are out of scope and belong to a Phase 2 plan. Do not add them here.

**Conventions for every task:**
- Module path is `github.com/luckyrogue/lead-cat`. Backend lives in `apps/backend`; run all Go commands from `apps/backend`.
- Run Go tooling as `env -u GOROOT go ...` (the repo's in-flight refactor leaves a stale `GOROOT`). **Ignore IDE/LSP diagnostics about `go.mod` or undefined symbols — trust `go build`/`go test`/`go vet` output only.**
- Commit messages end with the trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- Stage only the explicit files named in each commit step. Never `git add -A`. Never touch `Makefile`, `deploy/*`, or `.gitignore`.

---

## File Structure

| File | Responsibility |
|---|---|
| `apps/backend/internal/application/planner.go` | Provider-neutral port: `Planner` interface + `AgentTool` / `AgentToolCall` / `AgentToolResult` / `AgentMessage` / `AgentTurn` types. No SDK imports. |
| `apps/backend/internal/platform/scheduler_agent/state.go` | `State` (transcript), `Reply`, `Button` — mirrors `checker/state.go`. |
| `apps/backend/internal/platform/scheduler_agent/redis_sessions.go` | Redis-backed `sessions` store — mirrors `checker/redis_sessions.go`. |
| `apps/backend/internal/platform/scheduler_agent/tools.go` | `ToolSpecs()` (the tool list as data) + pure `Dispatch()` (decode args → call backend → format text). The unit-test surface. |
| `apps/backend/internal/platform/scheduler_agent/prompt.go` | The `systemPrompt` constant. |
| `apps/backend/internal/platform/scheduler_agent/service.go` | `Service` with `Start` / `OnText`: the agentic loop driving `Planner` + `Dispatch` + sessions. |
| `apps/backend/internal/infrastructure/llm/anthropic/planner.go` | Implements `application.Planner` via `anthropic-sdk-go`. The only file that imports the SDK. |
| `apps/backend/internal/infrastructure/telegram/multitenant.go` (modify) | Construct the agent, add it to the free-text fallback chain, render its `Reply`. |
| `apps/backend/cmd/server/main.go` (modify) | Construct the Anthropic planner and pass it into `NewMultiHandler`. |

---

## Task 1: Add the anthropic-sdk-go dependency

**Files:**
- Modify: `apps/backend/go.mod`, `apps/backend/go.sum`

- [ ] **Step 1: Add the dependency**

Run (from `apps/backend`):

```bash
env -u GOROOT go get github.com/anthropics/anthropic-sdk-go@latest
```

Expected: `go.mod` gains a `github.com/anthropics/anthropic-sdk-go vX.Y.Z` require line; `go.sum` updated.

- [ ] **Step 2: Verify it builds**

Run: `env -u GOROOT go build ./...`
Expected: builds clean (no usages yet, so this just confirms the module resolves).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build(backend): add anthropic-sdk-go for the scheduling agent

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Define the provider-neutral Planner port

The `application` layer must not import the SDK (depguard `application-no-infra`). It defines a neutral contract; the adapter (Task 7) maps it to the SDK.

**Files:**
- Create: `apps/backend/internal/application/planner.go`

- [ ] **Step 1: Write the port file**

```go
package application

import (
	"context"
	"encoding/json"
)

// AgentTool is one tool the planner may call, described in provider-neutral form.
// InputSchema is a JSON Schema object (map form).
type AgentTool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// AgentToolCall is a model request to run a tool. Input is the raw JSON arguments.
type AgentToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// AgentToolResult is the outcome of running a tool, fed back to the model.
type AgentToolResult struct {
	ID      string
	Content string
	IsError bool
}

// AgentMessage is one entry in the running transcript. An assistant message may
// carry ToolCalls; the matching user message carries ToolResults.
type AgentMessage struct {
	Role        string            `json:"role"` // "user" | "assistant"
	Text        string            `json:"text,omitempty"`
	ToolCalls   []AgentToolCall   `json:"tool_calls,omitempty"`
	ToolResults []AgentToolResult `json:"tool_results,omitempty"`
}

// AgentTurn is one assistant turn returned by the planner. When ToolCalls is
// non-empty the caller must run them and call Plan again; otherwise Text is the
// final answer.
type AgentTurn struct {
	Text      string
	ToolCalls []AgentToolCall
}

// Planner performs one stateless model round-trip: given the system prompt, the
// full transcript, and the tool list, it returns the next assistant turn.
type Planner interface {
	Plan(ctx context.Context, system string, history []AgentMessage, tools []AgentTool) (AgentTurn, error)
}
```

- [ ] **Step 2: Verify it builds**

Run: `env -u GOROOT go build ./internal/application/...`
Expected: builds clean.

- [ ] **Step 3: Commit**

```bash
git add internal/application/planner.go
git commit -m "feat(application): add provider-neutral Planner port for the scheduling agent

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Session state + Redis store (mirror checker)

**Files:**
- Create: `apps/backend/internal/platform/scheduler_agent/state.go`
- Create: `apps/backend/internal/platform/scheduler_agent/redis_sessions.go`

- [ ] **Step 1: Write state.go**

```go
package scheduler_agent

import "github.com/luckyrogue/lead-cat/internal/application"

// State is the per-user conversation transcript, persisted in Redis between turns.
type State struct {
	History []application.AgentMessage `json:"history,omitempty"`
}

type Button struct {
	Text string
	Data string
}

type Reply struct {
	Text     string
	Keyboard [][]Button
}
```

- [ ] **Step 2: Write redis_sessions.go** (mirrors `internal/platform/checker/redis_sessions.go`)

```go
package scheduler_agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionTTL = 30 * time.Minute

type RedisSessions struct {
	rdb *redis.Client
}

func NewRedisSessions(rdb *redis.Client) *RedisSessions {
	return &RedisSessions{rdb: rdb}
}

func (r *RedisSessions) key(telegramID int64) string {
	return "agent:session:" + itoa(telegramID)
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
	return r.rdb.Set(ctx, r.key(telegramID), raw, sessionTTL).Err()
}

func (r *RedisSessions) Del(ctx context.Context, telegramID int64) error {
	return r.rdb.Del(ctx, r.key(telegramID)).Err()
}

func itoa(v int64) string {
	return strconvFormatInt(v)
}
```

- [ ] **Step 3: Add the itoa helper without pulling strconv into the redis file twice**

Replace the `itoa` placeholder at the bottom of `redis_sessions.go` with a direct `strconv` call — delete the `itoa`/`strconvFormatInt` lines and import `strconv`:

Final `redis_sessions.go` import block and key method:

```go
import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)
```

```go
func (r *RedisSessions) key(telegramID int64) string {
	return "agent:session:" + strconv.FormatInt(telegramID, 10)
}
```

Delete the trailing `itoa` and `strconvFormatInt` definitions entirely.

- [ ] **Step 4: Verify it builds**

Run: `env -u GOROOT go build ./internal/platform/scheduler_agent/...`
Expected: builds clean.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/scheduler_agent/state.go internal/platform/scheduler_agent/redis_sessions.go
git commit -m "feat(scheduler-agent): session state + Redis transcript store

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Tool specs + pure Dispatch (the unit-test core)

`Dispatch` decodes a tool call's JSON arguments, calls the backend, and returns human-readable text for the model. It is pure (no Telegram, no LLM) and is where the real tests live.

**Files:**
- Create: `apps/backend/internal/platform/scheduler_agent/tools.go`
- Test: `apps/backend/internal/platform/scheduler_agent/tools_test.go`

- [ ] **Step 1: Write the failing test**

```go
package scheduler_agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type fakeBackend struct {
	employees []model.Employee
	slots     []application.FreeSlot
	conflicts []application.Conflict
}

func (f fakeBackend) SearchEmployeesGlobal(_ context.Context, _ string) ([]model.Employee, error) {
	return f.employees, nil
}
func (f fakeBackend) FreeSlots(_ context.Context, _ []string, _, _ time.Time, _ int) ([]application.FreeSlot, error) {
	return f.slots, nil
}
func (f fakeBackend) MeetingConflicts(_ context.Context, _ []string, _, _ time.Time, _ uuid.UUID) ([]application.Conflict, error) {
	return f.conflicts, nil
}

func TestDispatch_SearchPeople(t *testing.T) {
	be := fakeBackend{employees: []model.Employee{
		{FullName: "Mia Cat", Email: "mia@co.com"},
		{FullName: "Alex Paw", Email: "alex@co.com"},
	}}
	out, err := Dispatch(context.Background(), be, "search_people", []byte(`{"query":"a"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "mia@co.com") || !strings.Contains(out, "alex@co.com") {
		t.Fatalf("expected both emails in output, got:\n%s", out)
	}
}

func TestDispatch_FindFreeSlots(t *testing.T) {
	day := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	be := fakeBackend{slots: []application.FreeSlot{
		{Day: day, Start: day.Add(9 * time.Hour), End: day.Add(10 * time.Hour), Mins: 60},
	}}
	out, err := Dispatch(context.Background(), be, "find_free_slots",
		[]byte(`{"emails":["mia@co.com"],"from":"2026-06-22","to":"2026-06-26","duration_mins":30}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "2026-06-22") {
		t.Fatalf("expected the slot day in output, got:\n%s", out)
	}
}

func TestDispatch_CheckConflicts_None(t *testing.T) {
	be := fakeBackend{}
	out, err := Dispatch(context.Background(), be, "check_conflicts",
		[]byte(`{"emails":["mia@co.com"],"start":"2026-06-22 10:00","end":"2026-06-22 10:30"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no conflicts") {
		t.Fatalf("expected a no-conflicts message, got:\n%s", out)
	}
}

func TestDispatch_UnknownTool(t *testing.T) {
	_, err := Dispatch(context.Background(), fakeBackend{}, "delete_everything", []byte(`{}`))
	if err == nil {
		t.Fatal("expected an error for an unknown tool")
	}
}

func TestDispatch_BadArgs(t *testing.T) {
	_, err := Dispatch(context.Background(), fakeBackend{}, "find_free_slots", []byte(`{"from":"not-a-date"}`))
	if err == nil {
		t.Fatal("expected an error for unparseable arguments")
	}
}

func TestToolSpecs_Shape(t *testing.T) {
	specs := ToolSpecs()
	names := map[string]bool{}
	for _, s := range specs {
		names[s.Name] = true
		if s.Description == "" || s.InputSchema == nil {
			t.Fatalf("tool %q missing description or schema", s.Name)
		}
	}
	for _, want := range []string{"search_people", "find_free_slots", "check_conflicts"} {
		if !names[want] {
			t.Fatalf("missing tool spec %q", want)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestDispatch -v`
Expected: FAIL to compile — `Dispatch`, `ToolSpecs`, and `Backend` are undefined.

- [ ] **Step 3: Write tools.go**

```go
package scheduler_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// almatyLoc is the default working timezone for parsing/formatting tool times in
// Phase 1 (per-user tz arrives with the booking phase).
var almatyLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}()

// Backend is the read-only slice of application.Services the agent needs.
// *application.Services satisfies this.
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error)
	FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
	MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, exclude uuid.UUID) ([]application.Conflict, error)
}

// ToolSpecs returns the read-only tool surface as provider-neutral specs.
func ToolSpecs() []application.AgentTool {
	return []application.AgentTool{
		{
			Name:        "search_people",
			Description: "Find colleagues by name or email substring. Call this first to resolve any person the user names into an email address before scheduling.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Name or email fragment to search for."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "find_free_slots",
			Description: "Find common free working-hours slots for the given participant emails across a date range. Use this to answer 'when is everyone free' questions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"emails":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Participant emails (resolve names via search_people first)."},
					"from":          map[string]any{"type": "string", "description": "Inclusive start date, format YYYY-MM-DD."},
					"to":            map[string]any{"type": "string", "description": "Inclusive end date, format YYYY-MM-DD."},
					"duration_mins": map[string]any{"type": "integer", "description": "Required free-block length in minutes."},
				},
				"required": []string{"emails", "from", "to", "duration_mins"},
			},
		},
		{
			Name:        "check_conflicts",
			Description: "Check whether any of the given participants already have a meeting overlapping a specific time window. Call this before telling the user a time is free to book.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"emails": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Participant emails."},
					"start":  map[string]any{"type": "string", "description": "Window start, format 'YYYY-MM-DD HH:MM' (Almaty time)."},
					"end":    map[string]any{"type": "string", "description": "Window end, format 'YYYY-MM-DD HH:MM' (Almaty time)."},
				},
				"required": []string{"emails", "start", "end"},
			},
		},
	}
}

// Dispatch runs one tool call and returns text for the model. A returned error
// means the call could not be executed (bad args / unknown tool); the caller
// surfaces it back to the model as an error tool result.
func Dispatch(ctx context.Context, be Backend, name string, args json.RawMessage) (string, error) {
	switch name {
	case "search_people":
		return dispatchSearch(ctx, be, args)
	case "find_free_slots":
		return dispatchFreeSlots(ctx, be, args)
	case "check_conflicts":
		return dispatchConflicts(ctx, be, args)
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

func dispatchSearch(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	emps, err := be.SearchEmployeesGlobal(ctx, in.Query)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	var b strings.Builder
	n := 0
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		fmt.Fprintf(&b, "- %s <%s>\n", e.FullName, e.Email)
		n++
	}
	if n == 0 {
		return fmt.Sprintf("No people found matching %q.", in.Query), nil
	}
	return b.String(), nil
}

func dispatchFreeSlots(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Emails       []string `json:"emails"`
		From         string   `json:"from"`
		To           string   `json:"to"`
		DurationMins int      `json:"duration_mins"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	from, err := time.ParseInLocation("2006-01-02", in.From, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'from' date (want YYYY-MM-DD): %w", err)
	}
	to, err := time.ParseInLocation("2006-01-02", in.To, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'to' date (want YYYY-MM-DD): %w", err)
	}
	// `to` is inclusive; FreeSlots scans [from, to) so push the end out one day.
	slots, err := be.FreeSlots(ctx, in.Emails, from, to.AddDate(0, 0, 1), in.DurationMins)
	if err != nil {
		return "", err
	}
	if len(slots) == 0 {
		return "No common free slots in that range. Suggest a wider range, a shorter duration, or fewer participants.", nil
	}
	var b strings.Builder
	for _, sl := range slots {
		fmt.Fprintf(&b, "- %s %s–%s (%d min free)\n",
			sl.Day.In(almatyLoc).Format("2006-01-02 Mon"),
			sl.Start.In(almatyLoc).Format("15:04"),
			sl.End.In(almatyLoc).Format("15:04"),
			sl.Mins)
	}
	return b.String(), nil
}

func dispatchConflicts(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Emails []string `json:"emails"`
		Start  string   `json:"start"`
		End    string   `json:"end"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", in.Start, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'start' (want 'YYYY-MM-DD HH:MM'): %w", err)
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", in.End, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'end' (want 'YYYY-MM-DD HH:MM'): %w", err)
	}
	conflicts, err := be.MeetingConflicts(ctx, in.Emails, start.UTC(), end.UTC(), uuid.Nil)
	if err != nil {
		return "", err
	}
	if len(conflicts) == 0 {
		return "No conflicts — everyone is free in that window.", nil
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Start.Before(conflicts[j].Start) })
	var b strings.Builder
	b.WriteString("Conflicts found:\n")
	for _, c := range conflicts {
		fmt.Fprintf(&b, "- %s busy with %q at %s–%s\n",
			c.PersonName, c.MeetingName,
			c.Start.In(almatyLoc).Format("2006-01-02 15:04"),
			c.End.In(almatyLoc).Format("15:04"))
	}
	return b.String(), nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -v`
Expected: PASS (all `TestDispatch_*` and `TestToolSpecs_Shape`).

- [ ] **Step 5: Commit**

```bash
git add internal/platform/scheduler_agent/tools.go internal/platform/scheduler_agent/tools_test.go
git commit -m "feat(scheduler-agent): read-only tool specs + pure Dispatch

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: System prompt

**Files:**
- Create: `apps/backend/internal/platform/scheduler_agent/prompt.go`

- [ ] **Step 1: Write prompt.go**

```go
package scheduler_agent

// systemPrompt steers the read-only scheduling assistant. Russian to match the
// bot's existing copy; tone is the cozy Lead Cat persona.
const systemPrompt = `Ты — Lead Cat, дружелюбный помощник по расписанию в Telegram. Отвечай по-русски, кратко и по-доброму 🐾.

Ты умеешь ТОЛЬКО смотреть расписание и искать свободное время. Ты НЕ создаёшь, не переносишь и не отменяешь встречи — если просят об этом, вежливо предложи кнопку «Новая встреча» через команду /new.

Правила:
- Имена людей сначала преврати в email через инструмент search_people. Если найдено несколько совпадений — уточни у пользователя, кого он имел в виду.
- Чтобы ответить «когда все свободны», используй find_free_slots.
- Прежде чем сказать, что время свободно, проверь его через check_conflicts.
- Рабочее время — 09:00–18:00 по Алматы, будни. Даты передавай в инструменты в формате YYYY-MM-DD, время — 'YYYY-MM-DD HH:MM'.
- Если данных не хватает (нет участников, неясен диапазон) — задай один короткий уточняющий вопрос вместо догадок.
- Не выдумывай людей, встречи или свободные слоты — опирайся только на результаты инструментов.`
```

- [ ] **Step 2: Verify it builds**

Run: `env -u GOROOT go build ./internal/platform/scheduler_agent/...`
Expected: builds clean.

- [ ] **Step 3: Commit**

```bash
git add internal/platform/scheduler_agent/prompt.go
git commit -m "feat(scheduler-agent): system prompt for the read-only assistant

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: The agentic loop Service (the behavioral core)

`Service.OnText` runs the loop: append the user message, call the planner, run any tool calls via `Dispatch`, feed results back, repeat until the planner returns a final text answer or the iteration cap is hit. Tested with a fake planner + fake backend — no SDK, no Redis.

**Files:**
- Create: `apps/backend/internal/platform/scheduler_agent/service.go`
- Test: `apps/backend/internal/platform/scheduler_agent/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
package scheduler_agent

import (
	"context"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// memSessions is an in-memory sessions store for tests.
type memSessions struct{ m map[int64]State }

func newMemSessions() *memSessions { return &memSessions{m: map[int64]State{}} }
func (s *memSessions) Get(_ context.Context, id int64) (*State, error) {
	st, ok := s.m[id]
	if !ok {
		return nil, nil
	}
	cp := st
	return &cp, nil
}
func (s *memSessions) Set(_ context.Context, id int64, st State) error { s.m[id] = st; return nil }
func (s *memSessions) Del(_ context.Context, id int64) error           { delete(s.m, id); return nil }

// scriptPlanner returns pre-scripted turns, one per call.
type scriptPlanner struct {
	turns []application.AgentTurn
	calls int
	got   []int // number of history messages seen on each call
}

func (p *scriptPlanner) Plan(_ context.Context, _ string, history []application.AgentMessage, _ []application.AgentTool) (application.AgentTurn, error) {
	p.got = append(p.got, len(history))
	turn := p.turns[p.calls]
	p.calls++
	return turn, nil
}

func TestService_OnText_DirectAnswer(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{{Text: "Привет! Кого ищем?"}}}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 42, "привет")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if reply.Text != "Привет! Кого ищем?" {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 1 {
		t.Fatalf("planner called %d times, want 1", planner.calls)
	}
}

func TestService_OnText_RunsToolThenAnswers(t *testing.T) {
	be := fakeBackend{employees: []model.Employee{{FullName: "Mia", Email: "mia@co.com"}}}
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "t1", Name: "search_people", Input: []byte(`{"query":"mia"}`)}}},
		{Text: "Нашёл Mia <mia@co.com>."},
	}}
	svc := New(planner, be, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 7, "найди mia")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if !strings.Contains(reply.Text, "mia@co.com") {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 2 {
		t.Fatalf("planner called %d times, want 2", planner.calls)
	}
	// Second call must see: user msg, assistant tool-call msg, user tool-result msg.
	if planner.got[1] != 3 {
		t.Fatalf("second Plan saw %d history messages, want 3", planner.got[1])
	}
}

func TestService_OnText_UnknownToolBecomesErrorResult(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "t1", Name: "nope", Input: []byte(`{}`)}}},
		{Text: "Извини, не получилось."},
	}}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, _ := svc.OnText(context.Background(), 9, "do x")
	if reply.Text != "Извини, не получилось." {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 2 {
		t.Fatalf("planner called %d times, want 2", planner.calls)
	}
}

func TestService_OnText_IterationCap(t *testing.T) {
	// A planner that always asks for a tool would loop forever without the cap.
	loopTurn := application.AgentTurn{ToolCalls: []application.AgentToolCall{{ID: "t", Name: "search_people", Input: []byte(`{"query":"x"}`)}}}
	turns := make([]application.AgentTurn, 20)
	for i := range turns {
		turns[i] = loopTurn
	}
	planner := &scriptPlanner{turns: turns}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 1, "loop")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if reply.Text == "" {
		t.Fatal("expected a graceful fallback message, got empty")
	}
	if planner.calls > maxIterations {
		t.Fatalf("planner called %d times, exceeds cap %d", planner.calls, maxIterations)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestService -v`
Expected: FAIL to compile — `New`, `Service`, `maxIterations` undefined.

- [ ] **Step 3: Write service.go**

```go
package scheduler_agent

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application"
)

const maxIterations = 6

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	planner  application.Planner
	backend  Backend
	sessions sessions
	tools    []application.AgentTool
}

func New(planner application.Planner, backend Backend, sess sessions) *Service {
	return &Service{planner: planner, backend: backend, sessions: sess, tools: ToolSpecs()}
}

// OnText handles a free-text message. It always returns handled=true for a
// registered user's private message (the agent is the catch-all), so wire it
// LAST in the dispatcher fallback chain.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		st = &State{}
	}
	st.History = append(st.History, application.AgentMessage{Role: "user", Text: text})

	for i := 0; i < maxIterations; i++ {
		turn, perr := s.planner.Plan(ctx, systemPrompt, st.History, s.tools)
		if perr != nil {
			return Reply{Text: "Не получилось обработать запрос, попробуй ещё раз чуть позже 🐾"}, true
		}
		if len(turn.ToolCalls) == 0 {
			st.History = append(st.History, application.AgentMessage{Role: "assistant", Text: turn.Text})
			_ = s.sessions.Set(ctx, telegramID, *st)
			return Reply{Text: turn.Text}, true
		}
		// Record the assistant's tool-call turn, then run the tools.
		st.History = append(st.History, application.AgentMessage{Role: "assistant", Text: turn.Text, ToolCalls: turn.ToolCalls})
		results := make([]application.AgentToolResult, 0, len(turn.ToolCalls))
		for _, call := range turn.ToolCalls {
			out, derr := Dispatch(ctx, s.backend, call.Name, call.Input)
			if derr != nil {
				results = append(results, application.AgentToolResult{ID: call.ID, Content: derr.Error(), IsError: true})
				continue
			}
			results = append(results, application.AgentToolResult{ID: call.ID, Content: out})
		}
		st.History = append(st.History, application.AgentMessage{Role: "user", ToolResults: results})
	}

	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Это оказалось сложновато 🐾 Попробуй переформулировать или уточнить участников и даты."}, true
}

// Start resets the conversation and greets — used if the agent ever gets a
// dedicated command. The dispatcher's free-text path calls OnText directly.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: "Спроси меня про расписание — например: «когда у Миа и Алекса есть общий час на следующей неделе?» 🐾"}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -v`
Expected: PASS (all `TestService_*` plus the Task 4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/platform/scheduler_agent/service.go internal/platform/scheduler_agent/service_test.go
git commit -m "feat(scheduler-agent): agentic loop service (tool loop, iteration cap)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Anthropic adapter implementing the Planner port

Maps the neutral transcript to `anthropic-sdk-go` messages, calls `claude-opus-4-8` with adaptive thinking, and maps the response back. This is the only SDK consumer. LLM calls aren't unit-testable without a live key, so this task is verified by `go build` + `go vet` plus an optional env-gated smoke test.

**Files:**
- Create: `apps/backend/internal/infrastructure/llm/anthropic/planner.go`
- Test: `apps/backend/internal/infrastructure/llm/anthropic/planner_smoke_test.go`

- [ ] **Step 1: Write planner.go**

```go
package anthropic

import (
	"context"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// Planner implements application.Planner using the Anthropic Messages API.
type Planner struct {
	client sdk.Client
	model  sdk.Model
}

// New builds a Planner. apiKey may be empty to fall back to ANTHROPIC_API_KEY.
func New(apiKey string) *Planner {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Planner{
		client: sdk.NewClient(opts...),
		model:  sdk.ModelClaudeOpus4_8,
	}
}

var _ application.Planner = (*Planner)(nil)

func (p *Planner) Plan(ctx context.Context, system string, history []application.AgentMessage, tools []application.AgentTool) (application.AgentTurn, error) {
	adaptive := sdk.ThinkingConfigAdaptiveParam{}
	resp, err := p.client.Messages.New(ctx, sdk.MessageNewParams{
		Model:     p.model,
		MaxTokens: 16000,
		System:    []sdk.TextBlockParam{{Text: system}},
		Thinking:  sdk.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		Messages:  toSDKMessages(history),
		Tools:     toSDKTools(tools),
	})
	if err != nil {
		return application.AgentTurn{}, fmt.Errorf("anthropic plan: %w", err)
	}
	if resp.StopReason == sdk.StopReasonRefusal {
		return application.AgentTurn{Text: "Извини, не могу с этим помочь 🐾"}, nil
	}
	return fromSDKResponse(resp), nil
}

func toSDKTools(tools []application.AgentTool) []sdk.ToolUnionParam {
	out := make([]sdk.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		props, _ := t.InputSchema["properties"].(map[string]any)
		tool := sdk.ToolParam{
			Name:        t.Name,
			Description: sdk.String(t.Description),
			InputSchema: sdk.ToolInputSchemaParam{Properties: props},
		}
		out = append(out, sdk.ToolUnionParam{OfTool: &tool})
	}
	return out
}

func toSDKMessages(history []application.AgentMessage) []sdk.MessageParam {
	out := make([]sdk.MessageParam, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "assistant":
			var blocks []sdk.ContentBlockParamUnion
			if m.Text != "" {
				blocks = append(blocks, sdk.NewTextBlock(m.Text))
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, sdk.ContentBlockParamUnion{
					OfToolUse: &sdk.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: tc.Input,
					},
				})
			}
			out = append(out, sdk.NewAssistantMessage(blocks...))
		default: // "user"
			if len(m.ToolResults) > 0 {
				blocks := make([]sdk.ContentBlockParamUnion, 0, len(m.ToolResults))
				for _, tr := range m.ToolResults {
					blocks = append(blocks, sdk.NewToolResultBlock(tr.ID, tr.Content, tr.IsError))
				}
				out = append(out, sdk.NewUserMessage(blocks...))
				continue
			}
			out = append(out, sdk.NewUserMessage(sdk.NewTextBlock(m.Text)))
		}
	}
	return out
}

func fromSDKResponse(resp *sdk.Message) application.AgentTurn {
	var turn application.AgentTurn
	for _, block := range resp.Content {
		switch v := block.AsAny().(type) {
		case sdk.TextBlock:
			turn.Text += v.Text
		case sdk.ToolUseBlock:
			turn.ToolCalls = append(turn.ToolCalls, application.AgentToolCall{
				ID:    v.ID,
				Name:  v.Name,
				Input: []byte(v.JSON.Input.Raw()),
			})
		}
	}
	return turn
}
```

> **Note for the implementer:** the exact SDK symbol names above (`sdk.ToolUseBlockParam`, `sdk.NewToolResultBlock`, `v.JSON.Input.Raw()`, `sdk.NewAssistantMessage`) are taken from the anthropic-sdk-go Go reference. If `go build` reports a name mismatch (the SDK is newly added), open `$(go env GOMODCACHE)/github.com/anthropics/anthropic-sdk-go@*/` and grep for the constructor/field, or WebFetch `https://github.com/anthropics/anthropic-sdk-go` — do not invent a different name. Fix the call to match the installed version; the shape (text blocks + tool_use blocks out, tool_result blocks in) is correct.

- [ ] **Step 2: Write the env-gated smoke test**

```go
package anthropic

import (
	"context"
	"os"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// TestPlanner_Smoke hits the live API; skipped unless ANTHROPIC_API_KEY is set.
func TestPlanner_Smoke(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping live smoke test")
	}
	p := New("")
	turn, err := p.Plan(context.Background(), "You are a helpful assistant. Reply in one short word.",
		[]application.AgentMessage{{Role: "user", Text: "Say hi."}}, nil)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if turn.Text == "" && len(turn.ToolCalls) == 0 {
		t.Fatal("empty turn from live API")
	}
}
```

- [ ] **Step 3: Verify it builds and vets**

Run: `env -u GOROOT go build ./internal/infrastructure/llm/anthropic/... && env -u GOROOT go vet ./internal/infrastructure/llm/anthropic/...`
Expected: builds and vets clean. (If a symbol name mismatches, fix per the implementer note, then re-run.)

- [ ] **Step 4: Run the test (skips without a key)**

Run: `env -u GOROOT go test ./internal/infrastructure/llm/anthropic/... -run TestPlanner_Smoke -v`
Expected: `--- SKIP` when no key is set; PASS if a key is exported.

- [ ] **Step 5: Confirm depguard still passes (no infra leak into application)**

Run: `golangci-lint run ./internal/application/... ./internal/infrastructure/llm/...`
Expected: `0 issues`. (The adapter imports `application`, never the reverse.)

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/llm/anthropic/planner.go internal/infrastructure/llm/anthropic/planner_smoke_test.go
git commit -m "feat(llm): anthropic adapter implementing the Planner port

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Wire the agent into the bot dispatcher

Add the agent to the `botBackend` union (so `*application.Services` satisfies it), construct it in `NewMultiHandler`, route free text to it as the **last** fallback, render its `Reply`, and build the planner in `main`.

**Files:**
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go`
- Modify: `apps/backend/cmd/server/main.go`

- [ ] **Step 1: Extend the imports and backend union in multitenant.go**

Add the import:

```go
"github.com/luckyrogue/lead-cat/internal/platform/scheduler_agent"
```

Extend `botBackend`:

```go
type botBackend interface {
	meetingedit.Backend
	scheduleview.Backend
	checker.Backend
	scheduler_agent.Backend
}
```

- [ ] **Step 2: Add the field, constructor param, and construction**

Add to the `MultiHandler` struct (after `checker *checker.Service`):

```go
	agent *scheduler_agent.Service
```

Change the `NewMultiHandler` signature to accept the planner:

```go
func NewMultiHandler(store *postgres.Store, b *bot.Bot, rdb *redis.Client, adminIDs []int64, webappURL string, backend botBackend, planner application.Planner, log *zap.Logger) *MultiHandler {
```

Add the `application` import to multitenant.go:

```go
"github.com/luckyrogue/lead-cat/internal/application"
```

Construct the agent inside `NewMultiHandler` (after `chk := checker.New(...)`):

```go
	agent := scheduler_agent.New(planner, backend, scheduler_agent.NewRedisSessions(rdb))
```

Add `agent: agent,` to the returned `&MultiHandler{...}` literal.

- [ ] **Step 3: Route free text to the agent as the last fallback**

In `Handle`, the free-text block currently ends with the `checker.OnText` call. Replace that tail so the agent runs when checker doesn't claim the text. The existing block is:

```go
			if reply, handled := h.checker.OnText(ctx, from.ID, text); handled {
				h.sendCheckerReply(ctx, b, chatID, 0, reply)
			}
		}
		return
```

Replace with:

```go
			if reply, handled := h.checker.OnText(ctx, from.ID, text); handled {
				h.sendCheckerReply(ctx, b, chatID, 0, reply)
				return
			}
			// Final fallback: only for registered users, hand free text to the agent.
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
				h.sendAgentReply(ctx, b, chatID, h.agent.OnText(ctx, from.ID, text))
			}
		}
		return
```

> Note: `h.agent.OnText` returns `(Reply, bool)`; pass both into `sendAgentReply` which ignores the bool (the agent always handles). Adjust the call to `reply, _ := h.agent.OnText(...)` then `h.sendAgentReply(ctx, b, chatID, reply)` if your linter dislikes passing a 2-tuple. Use the explicit form:

```go
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
				reply, _ := h.agent.OnText(ctx, from.ID, text)
				h.sendAgentReply(ctx, b, chatID, reply)
			}
```

- [ ] **Step 4: Add the sendAgentReply renderer**

Add near `sendCheckerReply` in multitenant.go:

```go
func (h *MultiHandler) sendAgentReply(ctx context.Context, b *bot.Bot, chatID int64, reply scheduler_agent.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toAgentMarkup(reply.Keyboard)
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toAgentMarkup(rows [][]scheduler_agent.Button) models.InlineKeyboardMarkup {
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

- [ ] **Step 5: Build the planner and pass it in main.go**

In `cmd/server/main.go`, add the import:

```go
llmanthropic "github.com/luckyrogue/lead-cat/internal/infrastructure/llm/anthropic"
```

Before the `telegram.NewMultiHandler(...)` call (currently line ~138), construct the planner (reads `ANTHROPIC_API_KEY` from env via the empty-string fallback):

```go
	planner := llmanthropic.New(os.Getenv("ANTHROPIC_API_KEY"))
```

Update the call to pass `planner` in the new position (after `services`, before `logger`):

```go
		tgHandler = telegram.NewMultiHandler(store, tg, rdb, cfg.BotAdminTelegramIDs, cfg.WebappURL, services, planner, logger)
```

Ensure `os` is imported in main.go (it almost certainly already is; if `go build` complains, add `"os"`).

- [ ] **Step 6: Build, vet, lint, test the whole backend**

Run:

```bash
env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test ./internal/platform/scheduler_agent/... && golangci-lint run ./internal/platform/scheduler_agent/... ./internal/infrastructure/telegram/... ./internal/infrastructure/llm/...
```

Expected: build + vet clean, scheduler_agent tests PASS, lint `0 issues`.

- [ ] **Step 7: Manual smoke (optional, needs a key + running bot)**

With `ANTHROPIC_API_KEY` set and the bot running against a test Telegram bot, DM it (as a registered user): "когда у <colleague> есть свободный час на этой неделе?" Expected: it resolves the person, finds slots, and replies in Russian. If it answers without calling tools or hallucinates names, tighten `systemPrompt` (Task 5) — that's prompt tuning, not a code bug.

- [ ] **Step 8: Commit**

```bash
git add internal/infrastructure/telegram/multitenant.go cmd/server/main.go
git commit -m "feat(bot): route free text to the NL scheduling agent (read-only)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage** (against the Phase-1 architecture):
- Workflow tier / self-hosted loop → Tasks 6 (loop) + 7 (adapter). ✓
- Single-agent pattern, neutral port → Task 2. ✓
- Tool surface = `search_people` / `find_free_slots` / `check_conflicts` mapping to `Services` → Task 4. ✓
- Read tools auto-run; no write tool → enforced by omission (no `book_meeting` in `ToolSpecs`). ✓
- Redis transcript per telegramID mirroring checker → Task 3. ✓
- Manual loop + iteration cap + refusal handling → Tasks 6 (cap) + 7 (refusal). ✓
- Dispatcher free-text fallback, registered-users-only → Task 8. ✓
- Clean architecture (no infra in application; depguard) → Task 2 (no SDK import) + Task 7 step 5. ✓
- Identity-injection guardrail, confirm-before-write, caller "my schedule" → **deferred to Phase 2** (read tools are email-keyed and need no org/user; documented in the scope boundary). ✓ (intentional gap)

**Placeholder scan:** every code step contains complete code; the one implementer note (Task 7) is a fallback for SDK symbol drift, not a missing implementation.

**Type consistency:** `application.AgentTool/AgentToolCall/AgentToolResult/AgentMessage/AgentTurn/Planner` defined in Task 2 are used verbatim in Tasks 4/6/7. `Backend` defined in Task 4 is referenced in Tasks 6/8. `Reply`/`Button`/`State` defined in Task 3 used in 6/8. `New(planner, backend, sess)` signature consistent across 6 and 8. `maxIterations` defined in 6, asserted in 6's test. `*application.Services` satisfies `scheduler_agent.Backend` because its methods (`SearchEmployeesGlobal`→`[]model.Employee`, `FreeSlots`→`[]application.FreeSlot`, `MeetingConflicts(..., uuid.UUID)`→`[]application.Conflict`) match exactly (verified against the live source).

---

## Execution Handoff

Plan complete. Phase 2 (booking: caller-identity resolution, the gated `book_meeting` write tool, and the inline-button confirm card on the dispatcher callback path) is a separate plan to be written after Phase 1 merges.
