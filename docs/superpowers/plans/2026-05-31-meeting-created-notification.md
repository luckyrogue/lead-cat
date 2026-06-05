# §5a Meeting-created notification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a meeting is created, DM its recipients (registered participants + organizer) a Telegram message with the name, time, and Meet link — durably via asynq, deduped, best-effort.

**Architecture:** `CreateMeeting` enqueues a `meeting:created` asynq job (best-effort). A worker in `meeting_notifier` resolves recipients through a new shared `meetingrecipients.Resolve` helper (also adopted by the reminder engine), dedups each send via a sentinel row in `meeting_reminders`, and sends the DM. Time is rendered in the workspace timezone.

**Tech Stack:** Go, asynq (hibiken/asynq), go-telegram/bot, pgx, zap, google/uuid.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-created-notification-design.md`

**Conventions:**

- Run Go commands from `backend/` with `env -u GOROOT` prefix.
- Module path: `github.com/Jaryq-Lab/notify-bot`.
- Build check: `env -u GOROOT go build ./...`

---

## Task 1: Shared recipient resolver (`meetingrecipients`)

**Files:**

- Create: `backend/internal/platform/meetingrecipients/recipients.go`
- Test: `backend/internal/platform/meetingrecipients/recipients_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/platform/meetingrecipients/recipients_test.go`:

```go
package meetingrecipients

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type fakeStore struct {
	parts      []postgres.MeetingParticipant
	byEmail    map[string]postgres.BotUser
	orgTG      int64
	orgLinked  bool
}

func (f *fakeStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.parts, nil
}
func (f *fakeStore) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return postgres.BotUser{}, errors.New("not found")
	}
	return u, nil
}
func (f *fakeStore) GetUserTelegramID(_ context.Context, _ uuid.UUID) (int64, bool, error) {
	return f.orgTG, f.orgLinked, nil
}

func TestResolve(t *testing.T) {
	org := uuid.New()

	// participant "a@x" is registered (tg 111), "b@x" has no bot_user (skipped),
	// organizer is linked (tg 999) and not a participant.
	f := &fakeStore{
		parts: []postgres.MeetingParticipant{{Email: "a@x"}, {Email: "b@x"}, {Email: ""}},
		byEmail: map[string]postgres.BotUser{
			"a@x": {TelegramID: 111, ReminderMinutes: "30"},
		},
		orgTG:     999,
		orgLinked: true,
	}
	m := postgres.Meeting{ID: uuid.New(), OrganizerUserID: &org}

	got := Resolve(context.Background(), f, m)
	if len(got) != 2 {
		t.Fatalf("want 2 recipients, got %d: %+v", len(got), got)
	}
	if got[0].TelegramID != 111 || got[0].ReminderMinutes != "30" || got[0].IsOrganizer {
		t.Fatalf("bad participant recipient: %+v", got[0])
	}
	if got[1].TelegramID != 999 || !got[1].IsOrganizer {
		t.Fatalf("bad organizer recipient: %+v", got[1])
	}
}

func TestResolveOrganizerAlsoParticipant(t *testing.T) {
	org := uuid.New()
	f := &fakeStore{
		parts:   []postgres.MeetingParticipant{{Email: "a@x"}},
		byEmail: map[string]postgres.BotUser{"a@x": {TelegramID: 999, ReminderMinutes: "15"}},
		orgTG:   999, orgLinked: true,
	}
	m := postgres.Meeting{ID: uuid.New(), OrganizerUserID: &org}

	got := Resolve(context.Background(), f, m)
	if len(got) != 1 {
		t.Fatalf("organizer who is a participant must not duplicate; got %d: %+v", len(got), got)
	}
	if got[0].IsOrganizer {
		t.Fatalf("existing participant entry should win (not organizer): %+v", got[0])
	}
}
```

- [ ] **Step 2: Run the test, verify it fails to compile**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingrecipients/ -v`
Expected: FAIL — `undefined: Resolve` / package has no non-test files.

- [ ] **Step 3: Write the implementation**

Create `backend/internal/platform/meetingrecipients/recipients.go`:

```go
// Package meetingrecipients resolves the Telegram recipients of a meeting:
// registered participants (by email) plus the organizer (if their Telegram is
// linked). Shared by the reminder engine and the meeting-created notifier.
package meetingrecipients

import (
	"context"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Store is the subset of *postgres.Store this package needs.
type Store interface {
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
}

// Recipient is one notification target.
type Recipient struct {
	TelegramID      int64
	ReminderMinutes string // from bot_users; empty for the organizer
	IsOrganizer     bool
}

// Resolve returns the meeting's recipients: registered participants (skipping
// those without a bot_users record) plus the organizer when linked and not
// already a participant. Order: participants first, organizer last.
func Resolve(ctx context.Context, store Store, m postgres.Meeting) []Recipient {
	var out []Recipient
	seen := map[int64]bool{}

	parts, err := store.ListParticipants(ctx, m.ID)
	if err != nil {
		return out
	}
	for _, p := range parts {
		if p.Email == "" {
			continue
		}
		u, err := store.GetBotUserByEmail(ctx, p.Email)
		if err != nil {
			continue
		}
		if seen[u.TelegramID] {
			continue
		}
		seen[u.TelegramID] = true
		out = append(out, Recipient{TelegramID: u.TelegramID, ReminderMinutes: u.ReminderMinutes})
	}

	if m.OrganizerUserID != nil {
		if tg, linked, err := store.GetUserTelegramID(ctx, *m.OrganizerUserID); err == nil && linked {
			if !seen[tg] {
				seen[tg] = true
				out = append(out, Recipient{TelegramID: tg, IsOrganizer: true})
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingrecipients/ -v`
Expected: PASS (`TestResolve`, `TestResolveOrganizerAlsoParticipant`).

- [ ] **Step 5: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingrecipients/ && git commit -m "feat(meetings): shared meeting recipient resolver

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Adopt the resolver in the reminder engine

**Files:**

- Modify: `backend/internal/platform/reminder_scheduler/scheduler.go` (the `recipients` method, lines ~80-107)

- [ ] **Step 1: Replace the `recipients` method**

In `backend/internal/platform/reminder_scheduler/scheduler.go`, replace the entire `recipients` method (the function starting at `func (s *Scheduler) recipients(`) with:

```go
// recipients maps telegram_id -> reminder offsets: registered participants use
// their own settings; the organizer uses the default. Resolution is delegated to
// the shared meetingrecipients helper.
func (s *Scheduler) recipients(ctx context.Context, m postgres.Meeting) map[int64][]int {
	out := map[int64][]int{}
	recs, err := meetingrecipients.Resolve(ctx, s.store, m)
	if err != nil {
		s.log.Warn("resolve recipients", zap.String("meeting_id", m.ID.String()), zap.Error(err))
		return out
	}
	for _, r := range recs {
		if r.IsOrganizer {
			out[r.TelegramID] = defaultOrganizerOffsets
		} else {
			out[r.TelegramID] = botsettings.Parse(r.ReminderMinutes)
		}
	}
	return out
}
```

(`zap` is already imported in `scheduler.go`.)

- [ ] **Step 2: Add the import**

In the same file's import block, add the meetingrecipients import (keep the existing `botsettings` and `postgres` imports):

```go
	"github.com/Jaryq-Lab/notify-bot/internal/platform/meetingrecipients"
```

- [ ] **Step 3: Build and run existing tests**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/platform/reminder_scheduler/ -v`
Expected: build OK; `TestDueOffsets`, `TestOffsetLabel`, `TestMessage` PASS. (`recipients` has no unit test — it requires a DB; build + the existing pure tests are the gate.)

If the build reports `GetUserTelegramID`/`ListParticipants`/`GetBotUserByEmail` no longer used elsewhere in the file, that's expected — they were only used by the old `recipients` body and are now reached via the resolver. No other code in this file should reference them.

- [ ] **Step 4: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/reminder_scheduler/scheduler.go && git commit -m "refactor(reminders): use shared meetingrecipients resolver

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Queue plumbing — `meeting:created` task + multi-handler server

**Files:**

- Modify: `backend/internal/infrastructure/queue/asynq/queue.go`
- Modify: `backend/cmd/server/main.go` (the `asynqqueue.NewServer(...)` call, line ~112)

- [ ] **Step 1: Add the task type, payload, enqueue, and parser**

In `backend/internal/infrastructure/queue/asynq/queue.go`, after the existing `EnqueueRun` method (line ~44), add:

```go
const TaskMeetingCreated = "meeting:created"

type MeetingCreatedPayload struct {
	WorkspaceID string `json:"workspace_id"`
	MeetingID   string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingCreatedPayload{
		WorkspaceID: workspaceID.String(),
		MeetingID:   meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingCreated, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingCreated(t *asynq.Task) (MeetingCreatedPayload, error) {
	var p MeetingCreatedPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
```

- [ ] **Step 2: Change `NewServer` to register multiple handlers**

In the same file, replace the existing `NewServer` function with:

```go
func NewServer(redisURL string, log *zap.Logger, handlers map[string]asynq.HandlerFunc) (*Server, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, err
	}
	srv := asynq.NewServer(opt, asynq.Config{Concurrency: 4})
	mux := asynq.NewServeMux()
	for taskType, h := range handlers {
		mux.HandleFunc(taskType, h)
	}
	return &Server{server: srv, mux: mux}, nil
}
```

- [ ] **Step 3: Update the call site in main.go**

In `backend/cmd/server/main.go`, replace:

```go
	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, asynqHandler)
```

with:

```go
	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, map[string]asynq.HandlerFunc{
		asynqqueue.TaskRunScenario: asynqHandler,
	})
```

- [ ] **Step 4: Build**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...`
Expected: build OK. (The `meeting:created` handler is wired in Task 6.)

- [ ] **Step 5: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/queue/asynq/queue.go backend/cmd/server/main.go && git commit -m "feat(queue): meeting:created task + multi-handler asynq server

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Notifier package (`meeting_notifier`)

**Files:**

- Create: `backend/internal/platform/meeting_notifier/message.go`
- Create: `backend/internal/platform/meeting_notifier/notifier.go`
- Test: `backend/internal/platform/meeting_notifier/message_test.go`

- [ ] **Step 1: Write the failing message test**

Create `backend/internal/platform/meeting_notifier/message_test.go`:

```go
package meeting_notifier

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessage(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	end := time.Date(2026, 5, 31, 15, 0, 0, 0, loc)

	m := buildMessage("Разработка | Планёрка", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"Новая встреча", "Разработка | Планёрка", "31.05.2026", "14:00–15:00", "https://meet.google.com/abc"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}

	if strings.Contains(buildMessage("X", "", start, end, loc), "🔗") {
		t.Fatal("no link line when meet link empty")
	}

	// stored times are UTC; rendering must convert to Almaty (+5).
	startUTC := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC) // 14:00 Almaty
	m2 := buildMessage("X", "", startUTC, startUTC.Add(time.Hour), loc)
	if !strings.Contains(m2, "14:00–15:00") {
		t.Fatalf("UTC not converted to Almaty: %q", m2)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v`
Expected: FAIL — `undefined: buildMessage`.

- [ ] **Step 3: Write the message builder**

Create `backend/internal/platform/meeting_notifier/message.go`:

```go
package meeting_notifier

import (
	"fmt"
	"time"
)

// buildMessage renders the creation DM. Times are converted to loc (workspace
// timezone, Almaty by default). The link line is omitted when meetLink is empty.
func buildMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	msg := fmt.Sprintf("📅 Новая встреча\n«%s»\n🗓 %s, %s–%s (Алматы)",
		name,
		s.Format("02.01.2006"),
		s.Format("15:04"),
		e.Format("15:04"))
	if meetLink != "" {
		msg += "\n🔗 " + meetLink
	}
	return msg
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v`
Expected: PASS (`TestBuildMessage`).

- [ ] **Step 5: Write the notifier**

Create `backend/internal/platform/meeting_notifier/notifier.go`:

```go
// Package meeting_notifier sends a Telegram DM when a meeting is created.
package meeting_notifier

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/meetingrecipients"
)

// offsetCreated is the sentinel offset reused in meeting_reminders to dedup the
// creation notice. It never collides with real reminder offsets (10/15/30/60/120/1440).
const offsetCreated = -1

type Notifier struct {
	store *postgres.Store
	bot   *bot.Bot
	log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, log *zap.Logger) *Notifier {
	return &Notifier{store: store, bot: b, log: log}
}

// HandleCreated DMs the meeting's recipients. Returns an error only when the
// meeting/workspace cannot be read (asynq should retry); a single failed send is
// logged and skipped so a retry does not re-DM everyone else.
func (n *Notifier) HandleCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		loc = time.UTC
	}
	text := buildMessage(m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		claimed, err := n.store.TryClaimReminder(ctx, m.ID, r.TelegramID, offsetCreated)
		if err != nil || !claimed {
			continue
		}
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting created",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
```

- [ ] **Step 6: Build and test**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v`
Expected: build OK; `TestBuildMessage` PASS. (`HandleCreated` needs a DB + bot — build-verified, exercised manually.)

- [ ] **Step 7: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meeting_notifier/ && git commit -m "feat(meetings): meeting-created notifier (DM + dedup)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Enqueue from `CreateMeeting` + logger on `Services`

**Files:**

- Modify: `backend/internal/application/services.go` (the `Services` struct, line ~20)
- Modify: `backend/internal/delivery/http/app.go` (Services construction, line ~93)
- Modify: `backend/internal/application/meeting_service.go` (`CreateMeeting`, end of function ~line 119)

- [ ] **Step 1: Add `Log` to the `Services` struct**

In `backend/internal/application/services.go`, change the struct:

```go
type Services struct {
	Store    *postgres.Store
	Cipher   *crypto.TokenCipher
	Queue    *asynqqueue.Client
	Calendar CalendarProvider
	Log      *zap.Logger
}
```

Add the import to the same file's import block:

```go
	"go.uber.org/zap"
```

- [ ] **Step 2: Pass the logger when constructing `Services`**

In `backend/internal/delivery/http/app.go` line ~93, change:

```go
		App:     &application.Services{Store: store, Cipher: cipher, Queue: queue, Calendar: calProvider},
```

to:

```go
		App:     &application.Services{Store: store, Cipher: cipher, Queue: queue, Calendar: calProvider, Log: log},
```

- [ ] **Step 3: Enqueue at the end of `CreateMeeting`**

In `backend/internal/application/meeting_service.go`, replace the final lines of `CreateMeeting`:

```go
	if len(in.Participants) > 0 {
		if err := s.Store.AddParticipants(ctx, m.ID, in.Participants); err != nil {
			return m, err
		}
		m.Participants = in.Participants
	}
	return m, nil
}
```

with:

```go
	if len(in.Participants) > 0 {
		if err := s.Store.AddParticipants(ctx, m.ID, in.Participants); err != nil {
			return m, err
		}
		m.Participants = in.Participants
	}
	// Best-effort: the meeting is already created; a failed enqueue only loses
	// the creation notification, so log and still return the meeting.
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingCreated(ctx, workspaceID, m.ID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting created",
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return m, nil
}
```

Add the import to `meeting_service.go`'s import block:

```go
	"go.uber.org/zap"
```

- [ ] **Step 4: Build**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...`
Expected: build OK.

- [ ] **Step 5: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/services.go backend/internal/application/meeting_service.go backend/internal/delivery/http/app.go && git commit -m "feat(meetings): enqueue meeting-created notification on create

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Wire the notifier handler in main.go

**Files:**

- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Construct the notifier and its handler**

In `backend/cmd/server/main.go`, immediately before the `asynqSrv, err := asynqqueue.NewServer(...)` call (the map introduced in Task 3), add:

```go
	notifier := meeting_notifier.New(store, tg, logger)
	meetingCreatedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingCreated(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.WorkspaceID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleCreated(c, wid, mid)
	}
```

- [ ] **Step 2: Register the handler in the server map**

Change the `NewServer` map (from Task 3) to include the meeting handler:

```go
	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, map[string]asynq.HandlerFunc{
		asynqqueue.TaskRunScenario:    asynqHandler,
		asynqqueue.TaskMeetingCreated: meetingCreatedHandler,
	})
```

- [ ] **Step 3: Add the import**

In `main.go`'s import block, add:

```go
	"github.com/Jaryq-Lab/notify-bot/internal/platform/meeting_notifier"
```

- [ ] **Step 4: Build**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...`
Expected: build OK.

- [ ] **Step 5: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/cmd/server/main.go && git commit -m "feat(meetings): register meeting-created asynq handler

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Full verification + docs

**Files:**

- Modify: `docs/MEETINGS.md` (Backend planned section)

- [ ] **Step 1: Run the full suite**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && make test && make lint && make build`
Expected: all green. (If `make` targets are unavailable, fall back to `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && env -u GOROOT go build ./...`.)

- [ ] **Step 2: Update the feature status doc**

In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list (after the "Reminder engine (done)" line), add:

```markdown
> **Meeting-created notification (§5a, done):** on create, a `meeting:created` asynq job DMs the recipients (registered participants + organizer) the meeting name, time (workspace TZ), and Meet link. Recipient resolution is shared with the reminder engine (`meetingrecipients.Resolve`); dedup reuses `meeting_reminders` with a sentinel offset. Best-effort send; needs the bot polling.
```

- [ ] **Step 3: Commit**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §5a meeting-created notification

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** flow (Tasks 3,5,6) · queue+task (Task 3) · worker/HandleCreated (Task 4) · message builder (Task 4) · dedup sentinel (Task 4) · shared resolver (Tasks 1,2) · wiring + Services.Log (Tasks 5,6) · tests (Tasks 1,4) · docs (Task 7). All covered.
- **Type consistency:** `meetingrecipients.Resolve(ctx, Store, postgres.Meeting) []Recipient`; `Recipient{TelegramID, ReminderMinutes, IsOrganizer}` used identically in Tasks 1, 2, 4. `EnqueueMeetingCreated(ctx, workspaceID, meetingID uuid.UUID)` / `ParseMeetingCreated` / `MeetingCreatedPayload{WorkspaceID, MeetingID}` consistent across Tasks 3, 5, 6. `offsetCreated = -1`, `buildMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location)` consistent in Task 4.
- **No placeholders:** every code/command step is concrete.

```

```
