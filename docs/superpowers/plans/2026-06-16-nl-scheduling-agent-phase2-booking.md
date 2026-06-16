# NL Scheduling Assistant (Phase 2: booking) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the existing read-only NL scheduling assistant actually *book* a one-off meeting — the model proposes, the user taps a Telegram confirm button, and only then does the bot create the meeting (real Google event + invites).

**Architecture:** Builds on Phase 1 (`platform/scheduler_agent`). Add a `propose_meeting` tool the model calls once it has gathered everything; the loop **intercepts** it (never dispatched to the read backend), stores a `PendingBooking` in the session, and renders a Telegram inline-button confirm card. On the confirm callback, the harness resolves org + organizer from the **authenticated BotUser** (never from the model) and calls the existing `CreateMeeting`. A new `Booker` port keeps that resolution out of the pure loop and testable. **One-off only** (`recurrence="once"`); recurring NL-booking is deferred (the Mini App handles recurring creation).

**Tech Stack:** Go 1.26, the existing `scheduler_agent` package + `application.Planner`, `go-telegram/bot` inline keyboards + callback queries, the existing `Services.ResolveMiniAppOrganization` / `EnsureMiniAppOrganizer` / `CreateMeeting`.

**Scope boundary:** One-off meetings only. No recurrence, no editing/cancelling via NL (Phase 1 is read; the Mini App + `/edit` handle mutations). The model proposes; the human confirms; the harness books. The LLM never chooses identity, org, or organizer.

**Conventions (every task):**
- Module `github.com/luckyrogue/lead-cat`; backend in `apps/backend`; run Go tooling from there with `env -u GOROOT go ...` (stale GOROOT). **Trust `go build`/`go test`/`go vet`/`golangci-lint` over IDE diagnostics.**
- Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- Stage only the files each task names; never `git add -A`; never touch `Makefile`, `deploy/*`, `.gitignore`, `.dockerignore`, or `apps/landing/**` / `apps/mini-app/**` (parallel WIP). Work on `main` (authorized). Don't push.

**Verified ground truth (do not re-derive):**
- `application.CreateMeetingInput = command.CreateInput{Dept, Type, Host, Date("YYYY-MM-DD"), Start("HH:MM"), End("HH:MM"), Recurrence, RecurrenceUntil, RecurrenceDays, Description, Participants []model.MeetingParticipant, Timezone string}`.
- `Services.CreateMeeting(ctx, organizationID, organizerID uuid.UUID, in CreateMeetingInput) (model.Meeting, error)`.
- `Services.ResolveMiniAppOrganization(ctx) (uuid.UUID, error)` (errors: `application.ErrGoogleNotConfigured`).
- `Services.EnsureMiniAppOrganizer(ctx, email string, telegramID int64) (uuid.UUID, error)` (errors: `application.ErrTelegramLinkedToOtherAccount`).
- `store.GetBotUserByTelegramID(ctx, telegramID) (model.BotUser, error)`; `model.BotUser` has `Email`, `TelegramID`, `FullName`, `Timezone`.
- `model.Meeting` has `MeetLink string`.
- Phase 1 `scheduler_agent.New(planner application.Planner, backend Backend, sess sessions) *Service`; `State{History []application.AgentMessage}`; `Reply{Text string; Keyboard [][]Button}`; `Button{Text, Data string}`. The dispatcher renders `Reply` via `sendAgentReply`/`toAgentMarkup` and calls `h.agent.OnText` as the last free-text fallback; callbacks are routed in `handleCallback`.

---

## File Structure

| File | Responsibility |
|---|---|
| `scheduler_agent/state.go` (modify) | Add `Pending *PendingBooking` to `State`; define `PendingBooking`. |
| `scheduler_agent/tools.go` (modify) | Add the `propose_meeting` spec to `ToolSpecs()` (read tools unchanged; Dispatch unchanged — propose is intercepted by the loop, never dispatched). |
| `scheduler_agent/booker.go` (new) | The `Booker` port + `PendingBooking`→confirm-card text helper (`describeBooking`). |
| `scheduler_agent/service.go` (modify) | New constructor param `booker Booker`; loop intercepts `propose_meeting` → store Pending + return confirm card; new `OnCallback` for confirm/cancel. |
| `scheduler_agent/prompt.go` (modify) | Teach the model it CAN book via `propose_meeting` (and must confirm details first). |
| `telegram/agent_booker.go` (new) | `Booker` implementation over `*postgres.Store` + `*application.Services` (the GetBotUser→Resolve→Ensure→CreateMeeting dance, error mapping). |
| `telegram/multitenant.go` (modify) | Construct the booker, pass to `scheduler_agent.New`; route the agent's confirm callbacks in `handleCallback`. |

---

## Task 1: PendingBooking state + the Booker port

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/state.go`
- Create: `apps/backend/internal/platform/scheduler_agent/booker.go`
- Test: `apps/backend/internal/platform/scheduler_agent/booker_test.go`

- [ ] **Step 1: Add PendingBooking to state.go**

Append to `state.go` and add the field to `State`:

```go
// PendingBooking is a meeting the model has proposed and is awaiting the user's
// confirm tap. One-off only in Phase 2.
type PendingBooking struct {
	Dept   string   `json:"dept,omitempty"`
	Type   string   `json:"type"`
	Date   string   `json:"date"`  // YYYY-MM-DD
	Start  string   `json:"start"` // HH:MM
	End    string   `json:"end"`   // HH:MM
	Emails []string `json:"emails"`
	Desc   string   `json:"desc,omitempty"`
}
```

Change the `State` struct to:

```go
type State struct {
	History []application.AgentMessage `json:"history,omitempty"`
	Pending *PendingBooking            `json:"pending,omitempty"`
}
```

- [ ] **Step 2: Write the failing test for the confirm-card text helper**

`booker_test.go`:

```go
package scheduler_agent

import (
	"strings"
	"testing"
)

func TestDescribeBooking(t *testing.T) {
	b := PendingBooking{
		Type:   "Sync",
		Date:   "2026-06-22",
		Start:  "10:00",
		End:    "10:30",
		Emails: []string{"mia@co.com", "alex@co.com"},
	}
	out := describeBooking(b)
	for _, want := range []string{"Sync", "2026-06-22", "10:00", "10:30", "mia@co.com", "alex@co.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("describeBooking missing %q; got:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 3: Run it — fails to compile** (`describeBooking`, `Booker` undefined):

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestDescribeBooking -v`
Expected: FAIL (undefined).

- [ ] **Step 4: Write booker.go**

```go
package scheduler_agent

import (
	"context"
	"fmt"
	"strings"
)

// Booker creates a meeting on behalf of the authenticated Telegram user. The
// implementation resolves org + organizer from the user's account — the agent
// never supplies identity. Book returns a short user-facing confirmation line.
type Booker interface {
	Book(ctx context.Context, telegramID int64, b PendingBooking) (string, error)
}

// describeBooking renders the confirm-card body for a proposed meeting (Russian,
// cozy tone to match the bot).
func describeBooking(b PendingBooking) string {
	var sb strings.Builder
	title := b.Type
	if b.Dept != "" {
		title = b.Dept + " · " + b.Type
	}
	fmt.Fprintf(&sb, "Создать встречу?\n\n📌 %s\n📅 %s, %s–%s\n👥 %s",
		title, b.Date, b.Start, b.End, strings.Join(b.Emails, ", "))
	if b.Desc != "" {
		fmt.Fprintf(&sb, "\n📝 %s", b.Desc)
	}
	return sb.String()
}
```

- [ ] **Step 5: Run the test — passes**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestDescribeBooking -v`
Expected: PASS.

- [ ] **Step 6: Verify the package still builds (State change is additive)**

Run: `env -u GOROOT go build ./internal/platform/scheduler_agent/...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/platform/scheduler_agent/state.go internal/platform/scheduler_agent/booker.go internal/platform/scheduler_agent/booker_test.go
git commit -m "feat(scheduler-agent): PendingBooking state + Booker port + confirm-card text

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: `propose_meeting` tool spec

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/tools.go`
- Test: `apps/backend/internal/platform/scheduler_agent/tools_test.go` (extend)

- [ ] **Step 1: Add the failing assertion**

Add to `tools_test.go` inside `TestToolSpecs_Shape` — change the wanted-names slice to include the new tool:

```go
	for _, want := range []string{"search_people", "find_free_slots", "check_conflicts", "propose_meeting"} {
		if !names[want] {
			t.Fatalf("missing tool spec %q", want)
		}
	}
```

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestToolSpecs_Shape -v` → FAIL (missing `propose_meeting`).

- [ ] **Step 2: Add the spec to `ToolSpecs()`** in tools.go (append to the returned slice, after `check_conflicts`):

```go
		{
			Name:        "propose_meeting",
			Description: "Propose a one-off meeting for the user to confirm. Call this ONLY after you have the title (type), date, start and end times, and at least one participant email, and have checked conflicts. This does NOT book — it shows the user a confirm button; they decide.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":   map[string]any{"type": "string", "description": "Meeting title/type, e.g. 'Design sync'."},
					"dept":   map[string]any{"type": "string", "description": "Optional department/team label."},
					"date":   map[string]any{"type": "string", "description": "Date, YYYY-MM-DD."},
					"start":  map[string]any{"type": "string", "description": "Start time, HH:MM (Almaty)."},
					"end":    map[string]any{"type": "string", "description": "End time, HH:MM (Almaty)."},
					"emails": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Participant emails (resolved via search_people)."},
					"desc":   map[string]any{"type": "string", "description": "Optional description."},
				},
				"required": []string{"type", "date", "start", "end", "emails"},
			},
		},
```

> Do NOT add a `propose_meeting` case to `Dispatch` — it is intercepted by the loop (Task 3), never dispatched to the read backend.

- [ ] **Step 3: Run tests — pass**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -v`
Expected: PASS (all existing + the extended shape test).

- [ ] **Step 4: Commit**

```bash
git add internal/platform/scheduler_agent/tools.go internal/platform/scheduler_agent/tools_test.go
git commit -m "feat(scheduler-agent): add propose_meeting tool spec

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Loop interception + confirm/cancel callback (the behavioral core)

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/service.go`
- Test: `apps/backend/internal/platform/scheduler_agent/service_test.go` (extend)

- [ ] **Step 1: Write failing tests** — append to `service_test.go`:

```go
// fakeBooker records the booking it was asked to make.
type fakeBooker struct {
	called  bool
	got     PendingBooking
	gotTgID int64
	result  string
	err     error
}

func (b *fakeBooker) Book(_ context.Context, telegramID int64, pb PendingBooking) (string, error) {
	b.called = true
	b.gotTgID = telegramID
	b.got = pb
	if b.err != nil {
		return "", b.err
	}
	return b.result, nil
}

func TestService_ProposeMeeting_ShowsConfirmCard(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "p1", Name: "propose_meeting",
			Input: []byte(`{"type":"Sync","date":"2026-06-22","start":"10:00","end":"10:30","emails":["mia@co.com"]}`)}}},
	}}
	booker := &fakeBooker{}
	sess := newMemSessions()
	svc := NewWithBooker(planner, fakeBackend{}, booker, sess)

	reply, handled := svc.OnText(context.Background(), 5, "book sync with mia tomorrow 10:00")
	if !handled {
		t.Fatal("handled=false")
	}
	if booker.called {
		t.Fatal("Booker must NOT be called until the user confirms")
	}
	if len(reply.Keyboard) == 0 {
		t.Fatal("expected a confirm keyboard")
	}
	if !strings.Contains(reply.Text, "2026-06-22") || !strings.Contains(reply.Text, "mia@co.com") {
		t.Fatalf("confirm card missing details: %q", reply.Text)
	}
	// Pending stored.
	st, _ := sess.Get(context.Background(), 5)
	if st == nil || st.Pending == nil || st.Pending.Type != "Sync" {
		t.Fatal("expected Pending booking stored in session")
	}
	// Planner stops at the proposal (one call, no extra loop).
	if planner.calls != 1 {
		t.Fatalf("planner calls = %d, want 1", planner.calls)
	}
}

func TestService_OnCallback_ConfirmBooks(t *testing.T) {
	booker := &fakeBooker{result: "Встреча создана ✅"}
	sess := newMemSessions()
	_ = sess.Set(context.Background(), 5, State{Pending: &PendingBooking{
		Type: "Sync", Date: "2026-06-22", Start: "10:00", End: "10:30", Emails: []string{"mia@co.com"},
	}})
	svc := NewWithBooker(&scriptPlanner{}, fakeBackend{}, booker, sess)

	reply, handled := svc.OnCallback(context.Background(), 5, "agent:book:yes")
	if !handled {
		t.Fatal("handled=false")
	}
	if !booker.called || booker.gotTgID != 5 || booker.got.Type != "Sync" {
		t.Fatalf("booker not called correctly: %+v", booker)
	}
	if !strings.Contains(reply.Text, "создана") {
		t.Fatalf("reply = %q", reply.Text)
	}
	// Pending cleared after booking.
	st, _ := sess.Get(context.Background(), 5)
	if st != nil && st.Pending != nil {
		t.Fatal("Pending should be cleared after confirm")
	}
}

func TestService_OnCallback_Cancel(t *testing.T) {
	booker := &fakeBooker{}
	sess := newMemSessions()
	_ = sess.Set(context.Background(), 5, State{Pending: &PendingBooking{Type: "Sync"}})
	svc := NewWithBooker(&scriptPlanner{}, fakeBackend{}, booker, sess)

	_, handled := svc.OnCallback(context.Background(), 5, "agent:book:no")
	if !handled {
		t.Fatal("handled=false")
	}
	if booker.called {
		t.Fatal("Booker must not be called on cancel")
	}
	st, _ := sess.Get(context.Background(), 5)
	if st != nil && st.Pending != nil {
		t.Fatal("Pending should be cleared on cancel")
	}
}

func TestService_OnCallback_NotOurs(t *testing.T) {
	svc := NewWithBooker(&scriptPlanner{}, fakeBackend{}, &fakeBooker{}, newMemSessions())
	_, handled := svc.OnCallback(context.Background(), 5, "chk:done")
	if handled {
		t.Fatal("must not handle callbacks that aren't agent:book:*")
	}
}
```

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -run TestService_Propose -run TestService_OnCallback -v` → FAIL (`NewWithBooker`, `OnCallback` undefined).

- [ ] **Step 2: Update service.go**

Add `booker Booker` to the struct and a new constructor; keep `New` as a thin wrapper (nil booker) so existing callers/tests compile. Add the propose interception in the loop and the `OnCallback` method.

Change the struct + constructors:

```go
type Service struct {
	planner  application.Planner
	backend  Backend
	booker   Booker
	sessions sessions
	tools    []application.AgentTool
}

// New keeps the Phase 1 signature (read-only; no booking).
func New(planner application.Planner, backend Backend, sess sessions) *Service {
	return NewWithBooker(planner, backend, nil, sess)
}

// NewWithBooker enables booking via propose_meeting + confirm callbacks.
func NewWithBooker(planner application.Planner, backend Backend, booker Booker, sess sessions) *Service {
	return &Service{planner: planner, backend: backend, booker: booker, sessions: sess, tools: ToolSpecs()}
}
```

In `OnText`, inside the loop, BEFORE the `Dispatch` fan-out, intercept a propose call. Replace the tool-call branch body so it first scans for `propose_meeting`:

```go
		// Record the assistant's tool-call turn.
		st.History = append(st.History, application.AgentMessage{
			Role: "assistant", Text: turn.Text,
			Thinking: turn.Thinking, ThinkingSignature: turn.ThinkingSignature,
			ToolCalls: turn.ToolCalls,
		})

		// A propose_meeting call is intercepted: store the pending booking and
		// show the confirm card instead of dispatching/looping.
		for _, call := range turn.ToolCalls {
			if call.Name == "propose_meeting" {
				pb, perr := parsePending(call.Input)
				if perr != nil {
					// Feed the error back so the model can fix the args.
					st.History = append(st.History, application.AgentMessage{Role: "user", ToolResults: []application.AgentToolResult{{ID: call.ID, Content: perr.Error(), IsError: true}}})
					goto runReadTools
				}
				st.Pending = &pb
				_ = s.sessions.Set(ctx, telegramID, *st)
				return Reply{
					Text: describeBooking(pb),
					Keyboard: [][]Button{{
						{Text: "Подтвердить ✅", Data: "agent:book:yes"},
						{Text: "Отмена", Data: "agent:book:no"},
					}},
				}, true
			}
		}

	runReadTools:
		results := make([]application.AgentToolResult, 0, len(turn.ToolCalls))
		for _, call := range turn.ToolCalls {
			if call.Name == "propose_meeting" {
				continue // already handled (or errored) above
			}
			out, derr := Dispatch(ctx, s.backend, call.Name, call.Input)
			if derr != nil {
				results = append(results, application.AgentToolResult{ID: call.ID, Content: derr.Error(), IsError: true})
				continue
			}
			results = append(results, application.AgentToolResult{ID: call.ID, Content: out})
		}
		st.History = append(st.History, application.AgentMessage{Role: "user", ToolResults: results})
```

> NOTE: if your Phase 1 loop body differs slightly, preserve its existing assistant-turn recording (including the Thinking fields from the prior fix) — the key additions are: the `propose_meeting` scan that returns the confirm card, and skipping `propose_meeting` in the Dispatch fan-out. Keep the `goto`/label only if you adopt the structure above; an `if/else` equivalent is fine — just ensure a bad-args propose feeds an `IsError` tool result and continues the loop, while a good propose returns the card.

Add the parse helper and `OnCallback` at the end of service.go:

```go
func parsePending(args []byte) (PendingBooking, error) {
	var in struct {
		Type   string   `json:"type"`
		Dept   string   `json:"dept"`
		Date   string   `json:"date"`
		Start  string   `json:"start"`
		End    string   `json:"end"`
		Emails []string `json:"emails"`
		Desc   string   `json:"desc"`
	}
	if err := jsonUnmarshal(args, &in); err != nil {
		return PendingBooking{}, fmt.Errorf("bad proposal arguments: %w", err)
	}
	if in.Type == "" || in.Date == "" || in.Start == "" || in.End == "" || len(in.Emails) == 0 {
		return PendingBooking{}, fmt.Errorf("proposal missing required fields (type, date, start, end, emails)")
	}
	return PendingBooking{Dept: in.Dept, Type: in.Type, Date: in.Date, Start: in.Start, End: in.End, Emails: in.Emails, Desc: in.Desc}, nil
}

// OnCallback handles the confirm/cancel taps on a booking card. Returns
// handled=false for callbacks that aren't ours (so the dispatcher can fall through).
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	switch data {
	case "agent:book:yes":
		st, err := s.sessions.Get(ctx, telegramID)
		if err != nil || st == nil || st.Pending == nil {
			return Reply{Text: "Предложение устарело 🐾 Попроси заново."}, true
		}
		pb := *st.Pending
		st.Pending = nil
		_ = s.sessions.Set(ctx, telegramID, *st)
		if s.booker == nil {
			return Reply{Text: "Бронирование сейчас недоступно."}, true
		}
		msg, berr := s.booker.Book(ctx, telegramID, pb)
		if berr != nil {
			return Reply{Text: berr.Error()}, true
		}
		return Reply{Text: msg}, true
	case "agent:book:no":
		st, err := s.sessions.Get(ctx, telegramID)
		if err == nil && st != nil {
			st.Pending = nil
			_ = s.sessions.Set(ctx, telegramID, *st)
		}
		return Reply{Text: "Хорошо, не бронирую 🐾"}, true
	default:
		return Reply{}, false
	}
}
```

Add the imports `encoding/json` (aliased helper) and `fmt` to service.go if missing. Use `encoding/json` directly — replace `jsonUnmarshal` with `json.Unmarshal` and import `encoding/json`. (The `jsonUnmarshal` name above is a placeholder for clarity; use the real `json.Unmarshal`.)

- [ ] **Step 3: Run the package tests — all pass**

Run: `env -u GOROOT go test ./internal/platform/scheduler_agent/ -v`
Expected: PASS (Phase 1 tests + the four new ones). The Phase 1 `New(...)` calls in existing tests still compile (New delegates to NewWithBooker with nil booker).

- [ ] **Step 4: Commit**

```bash
git add internal/platform/scheduler_agent/service.go internal/platform/scheduler_agent/service_test.go
git commit -m "feat(scheduler-agent): propose_meeting interception + confirm/cancel callback

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Teach the prompt it can book

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/prompt.go`

- [ ] **Step 1: Update systemPrompt**

Replace the "Ты умеешь ТОЛЬКО смотреть..." paragraph and the booking-refusal line so the model knows it can propose a booking. Set the const body to:

```go
const systemPrompt = `Ты — Lead Cat, дружелюбный помощник по расписанию в Telegram. Отвечай по-русски, кратко и по-доброму 🐾.

Ты умеешь смотреть расписание, искать свободное время и предлагать создание одной (разовой) встречи на подтверждение пользователю. Повторяющиеся встречи пока создаются только в приложении (/new) — для них предложи кнопку.

Правила:
- Имена людей сначала преврати в email через инструмент search_people. Если найдено несколько совпадений — уточни у пользователя, кого он имел в виду.
- Чтобы ответить «когда все свободны», используй find_free_slots.
- Прежде чем предлагать время для встречи, проверь его через check_conflicts.
- Когда у тебя есть название (type), дата, время начала и конца и хотя бы один участник — вызови propose_meeting. Это НЕ создаёт встречу: пользователь увидит кнопку подтверждения и сам решит. Не вызывай propose_meeting, пока не хватает данных — задай один короткий уточняющий вопрос.
- Рабочее время — 09:00–18:00 по Алматы, будни. Даты — YYYY-MM-DD, время — HH:MM.
- Не выдумывай людей, встречи или свободные слоты — опирайся только на результаты инструментов.`
```

- [ ] **Step 2: Build**

Run: `env -u GOROOT go build ./internal/platform/scheduler_agent/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/platform/scheduler_agent/prompt.go
git commit -m "feat(scheduler-agent): prompt allows proposing a one-off booking

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: The Booker implementation (real org/organizer resolution + CreateMeeting)

**Files:**
- Create: `apps/backend/internal/infrastructure/telegram/agent_booker.go`

This lives in `telegram` because it needs both `*postgres.Store` (resolve the BotUser) and `*application.Services` (org/organizer/create). It implements `scheduler_agent.Booker`.

- [ ] **Step 1: Write agent_booker.go**

```go
package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/scheduler_agent"
)

// agentBooker creates a one-off meeting for the authenticated Telegram user.
// Identity (org + organizer) is resolved from the user's account here — never
// from the model.
type agentBooker struct {
	store    *postgres.Store
	services *application.Services
}

var _ scheduler_agent.Booker = (*agentBooker)(nil)

func (b *agentBooker) Book(ctx context.Context, telegramID int64, pb scheduler_agent.PendingBooking) (string, error) {
	bu, err := b.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", fmt.Errorf("Сначала зарегистрируйся: /start")
	}
	organizationID, err := b.services.ResolveMiniAppOrganization(ctx)
	if err != nil {
		if errors.Is(err, application.ErrGoogleNotConfigured) {
			return "", fmt.Errorf("Google-календарь не подключён — обратись к администратору.")
		}
		return "", fmt.Errorf("Не удалось создать встречу, попробуй позже 🐾")
	}
	organizerID, err := b.services.EnsureMiniAppOrganizer(ctx, bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return "", fmt.Errorf("Этот Telegram привязан к другому аккаунту.")
		}
		return "", fmt.Errorf("Не удалось создать встречу, попробуй позже 🐾")
	}
	parts := make([]model.MeetingParticipant, 0, len(pb.Emails))
	for _, e := range pb.Emails {
		parts = append(parts, model.MeetingParticipant{Email: e})
	}
	in := application.CreateMeetingInput{
		Dept: pb.Dept, Type: pb.Type, Host: bu.FullName,
		Date: pb.Date, Start: pb.Start, End: pb.End,
		Recurrence: "once", Description: pb.Desc,
		Participants: parts, Timezone: bu.Timezone,
	}
	m, err := b.services.CreateMeeting(ctx, organizationID, organizerID, in)
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) {
			return "", fmt.Errorf("Проверь данные встречи — что-то не так с датой или временем.")
		}
		return "", fmt.Errorf("Не удалось создать встречу, попробуй позже 🐾")
	}
	if m.MeetLink != "" {
		return "Встреча создана ✅\n" + m.MeetLink, nil
	}
	return "Встреча создана ✅", nil
}
```

> The returned `error` carries the user-facing message intentionally (the Service surfaces `err.Error()` to the user). If you'd rather not overload `error` as a message channel, change `Booker.Book` to return `(string, error)` where a non-nil error means "show this string" — but the simplest faithful approach is the above: friendly message in the error, and the Service already does `return Reply{Text: berr.Error()}`. Log the real underlying error before wrapping if `b.services.Log != nil` (Info/Warn with `telegram_id`), so operators see the true cause — add a `b.services.Log.Warn("agent_book_failed", ...)` on each error branch.

- [ ] **Step 2: Build**

Run: `env -u GOROOT go build ./internal/infrastructure/telegram/...`
Expected: clean. (No unit test here — it's thin glue over verified Services methods; covered by build + the Task 3 fake-Booker tests for the Service logic. A live end-to-end booking is exercised manually in Task 6.)

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/telegram/agent_booker.go
git commit -m "feat(bot): agentBooker — resolve org/organizer from the user and create a one-off meeting

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Wire booking into the dispatcher

**Files:**
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go`

- [ ] **Step 1: Construct the booker and pass it to the agent**

In `NewMultiHandler`, change the agent construction from:

```go
	agent := scheduler_agent.New(planner, backend, scheduler_agent.NewRedisSessions(rdb))
```
to:

```go
	booker := &agentBooker{store: store, services: servicesFromBackend(backend)}
	agent := scheduler_agent.NewWithBooker(planner, backend, booker, scheduler_agent.NewRedisSessions(rdb))
```

`backend` is typed `botBackend` (an interface). The booker needs `*application.Services`. The cleanest fix: change the `agentBooker` to depend on the narrow method set it needs rather than the concrete `*application.Services`, OR pass `*application.Services` into `NewMultiHandler` directly. **Do the latter** — `main.go` already passes the concrete `services` as `backend`; thread it as a concrete value too. Concretely:

- Change `NewMultiHandler`'s signature to add `services *application.Services` (in addition to `backend botBackend`), OR — simpler and avoiding a second param — change the `backend botBackend` parameter to `services *application.Services` and let it satisfy `botBackend` (it already does). Verify `*application.Services` satisfies `botBackend`; if yes, replace the param type:

```go
func NewMultiHandler(store *postgres.Store, b *bot.Bot, rdb *redis.Client, adminIDs []int64, webappURL string, services *application.Services, planner application.Planner, log *zap.Logger) *MultiHandler {
```
and use `services` everywhere `backend` was used (it's passed to `meetingedit.New`, `scheduleview.New`, `checker.New`, and `scheduler_agent.NewWithBooker` — all accept it via their `Backend` interfaces). Then:

```go
	booker := &agentBooker{store: store, services: services}
	agent := scheduler_agent.NewWithBooker(planner, services, booker, scheduler_agent.NewRedisSessions(rdb))
```

`main.go` already calls `telegram.NewMultiHandler(store, tg, rdb, ..., services, planner, logger)` with the concrete `services`, so no `main.go` change is needed. Delete the now-unused `botBackend` interface only if nothing else references it; otherwise leave it.

- [ ] **Step 2: Route the confirm/cancel callbacks**

In `handleCallback`, add the agent as a fallback in the callback chain (after the existing editor/schedule/checker callback handlers, before the function returns). Find where checker's callback is handled and add:

```go
	if reply, handled := h.agent.OnCallback(ctx, cq.From.ID, cq.Data); handled {
		h.sendAgentReply(ctx, b, cq.Message.Message.Chat.ID, reply)
		_ = answerCallback(ctx, b, cq.ID)
		return
	}
```

> Match the exact field access the surrounding callback handlers use for chat ID and the callback-answer call (e.g. `cq.Message.Message.Chat.ID` and whatever helper acknowledges the callback query — copy the pattern already used by `handleCallback` for checker/schedule). If the existing handlers don't explicitly answer the callback, omit `answerCallback`.

- [ ] **Step 3: Full gates**

Run:

```bash
env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test ./internal/platform/scheduler_agent/... && golangci-lint run ./internal/platform/scheduler_agent/... ./internal/infrastructure/telegram/...
```
Expected: build+vet clean, scheduler_agent tests PASS, lint `0 issues`.

- [ ] **Step 4: Manual smoke (needs ANTHROPIC_API_KEY + running bot + a registered user with Google connected)**

DM the bot: "забронируй встречу Sync с <colleague> завтра в 10:00 на 30 минут". Expected: it resolves the person, checks conflicts, shows a confirm card; tapping **Подтвердить ✅** creates the meeting (Google event + invite) and replies with the Meet link; tapping **Отмена** clears it. Verify the meeting appears in the Mini App / admin. If it books without confirming, or books on cancel — that's a wiring bug, fix before done.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/telegram/multitenant.go
git commit -m "feat(bot): wire booking confirm/cancel callbacks + booker into the agent

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:** propose tool (T2) · pending state (T1) · confirm-card + interception + callback (T3) · prompt (T4) · real booking with identity resolved from the authenticated user, not the model (T5) · dispatcher wiring incl. callback route (T6). One-off-only scope enforced by `Recurrence:"once"` in T5 and the prompt in T4. ✓

**Placeholder scan:** the two intentional implementer notes are (a) the `goto`/label structure in T3 may be written as `if/else` — behavior specified either way; (b) T6 callback field access must match the surrounding handlers — the pattern to copy is named. Replace the illustrative `jsonUnmarshal` with `json.Unmarshal` (called out in T3 Step 2). No TODO/"handle errors"/undefined symbols remain; all Services/store/model symbols are the verified ones from the ground-truth list.

**Type consistency:** `PendingBooking` fields identical across state.go (T1), parsePending (T3), describeBooking (T1), agentBooker (T5). `Booker.Book(ctx, int64, PendingBooking) (string, error)` consistent across booker.go, service.go, fakeBooker, agentBooker. `NewWithBooker(planner, backend, booker, sess)` consistent across T3 and T6. `New` retained as a nil-booker wrapper so Phase 1 tests/callers compile.

**Risk note:** this is the first write path in the agent. The confirm gate (T3) is the safety control — the fake-Booker tests assert booking does NOT happen without a confirm tap and DOES on confirm. Keep those tests; they are the guardrail's regression net.

---

## Execution Handoff

Plan complete and saved. Two execution options:
1. **Subagent-Driven (recommended)** — fresh subagent per task + spec/quality review, continuous; final whole-feature review (this is a write path — keep the final review).
2. **Inline Execution** — batched with checkpoints.

Because this creates real calendar events, a live end-to-end smoke (Task 6 Step 4) needs `ANTHROPIC_API_KEY` + a registered test user with Google connected before it can be called truly done.
