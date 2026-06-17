# WS2d-1 — meeting_notifier Orchestration Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract `sender`/`store` interfaces in `meeting_notifier` (behavior-preserving; `main.go` unchanged) and unit-test all five `Handle*` orchestration methods with in-memory fakes.

**Architecture:** A new `ports.go` declares the two interfaces + compile-time assertions that `*bot.Bot` and `*postgres.Store` satisfy them; `notifier.go` changes only its two field types and `New`'s parameter types. Tests live in-package (`package meeting_notifier`) with a `fakeStore` (drives `meetingrecipients.Resolve`) and a `fakeSender` (records sent messages).

**Tech Stack:** Go 1.26, stdlib `testing`/`database/sql` (for `sql.ErrNoRows`), go-telegram/bot types, uuid.

**Standing constraints (every task):** work on `main`, no branches; commit per task; stage only listed paths (never `git add -A`); `git status` before staging; run Go with `env -u GOROOT`; commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; do NOT touch `.github/workflows/_build.yml`; ignore stale IDE diagnostics, trust `go build`/`test`/`golangci-lint`.

---

## Verified facts

- `store` union (7 methods; `*postgres.Store` satisfies all): `GetMeeting(ctx,orgID,id)(postgres.Meeting,error)`, `GetOrganization(ctx,id)(postgres.Organization,error)`, `GetBotUserByEmail(ctx,email)(postgres.BotUser,error)`, `GetBotUserByTelegramID(ctx,tid)(postgres.BotUser,error)`, `GetUserTelegramID(ctx,userID)(int64,bool,error)`, `ListParticipants(ctx,meetingID)([]postgres.MeetingParticipant,error)`, `TryClaimReminder(ctx,meetingID,tid,offset)(bool,error)`. (First five = direct calls + `meetingrecipients.Store`; the union is assignable to `meetingrecipients.Store`.)
- `sender`: `SendMessage(ctx, *bot.SendMessageParams)(*models.Message,error)` (`*bot.Bot` satisfies).
- `meetingrecipients.Resolve(ctx, store, m)`: lists participants → for each non-empty email, `GetBotUserByEmail`; **any error → skip that participant**; dedup by `TelegramID`. Then if `m.OrganizerUserID != nil`, `GetUserTelegramID` → if `linked` and not already seen, add an organizer recipient (enriched via `GetBotUserByTelegramID`).
- `postgres.IsNotFound(err)` = `errors.Is(err, sql.ErrNoRows)` → fake returns `sql.ErrNoRows` for not-found.
- `postgres.ReminderOffsetCreated = -1`.
- `bot.SendMessageParams.ChatID` is `any`; the notifier sets it to `r.TelegramID` (int64).
- Message builders (WS2c-tested) and their headers: created `📅 Новая встреча`, updated `✏️ Встреча изменена`, cancelled `❌ Встреча отменена`, removed `➖`, participant-added `➕ Вас добавили на встречу`.

---

### Task 1: Extract interfaces (refactor)

**Files:**
- Create: `apps/backend/internal/platform/meeting_notifier/ports.go`
- Modify: `apps/backend/internal/platform/meeting_notifier/notifier.go` (field types + `New` signature only)

- [ ] **Step 1: Create `ports.go`**

```go
package meeting_notifier

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// sender is the subset of *bot.Bot the notifier needs.
type sender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

// store is the subset of *postgres.Store the notifier needs. It is a superset
// of meetingrecipients.Store, so it can be passed to meetingrecipients.Resolve.
type store interface {
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (postgres.Meeting, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (postgres.Organization, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	TryClaimReminder(ctx context.Context, meetingID uuid.UUID, telegramID int64, offset int) (bool, error)
}

var (
	_ sender = (*bot.Bot)(nil)
	_ store  = (*postgres.Store)(nil)
)
```

- [ ] **Step 2: Rewire `notifier.go`**

Change ONLY the struct field types and the `New` signature. Replace:

```go
type Notifier struct {
	store *postgres.Store
	bot   *bot.Bot
	log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, log *zap.Logger) *Notifier {
	return &Notifier{store: store, bot: b, log: log}
}
```

with:

```go
type Notifier struct {
	store store
	bot   sender
	log   *zap.Logger
}

func New(st store, b sender, log *zap.Logger) *Notifier {
	return &Notifier{store: st, bot: b, log: log}
}
```

Leave every `Handle*`/`notifyParticipant` body byte-identical (they use `n.store.X`, `n.bot.SendMessage`, `postgres.*`, `bot.SendMessageParams` — all still valid). Do NOT change imports — `notifier.go` still references `postgres.*` and `bot.SendMessageParams`.

- [ ] **Step 3: Verify build, vet, main.go untouched**

Run: `cd apps/backend && env -u GOROOT go build ./...`
Expected: clean — this proves `*postgres.Store` and `*bot.Bot` satisfy the interfaces (the `var _` assertions + `cmd/server/main.go`'s `meeting_notifier.New(store, tg, logger)` call still compile) and that the `store` union is assignable to `meetingrecipients.Store`.

Run: `cd apps/backend && env -u GOROOT go vet ./internal/platform/meeting_notifier/... ./cmd/server/...`
Expected: no issues.

Run: `git status --short` — confirm `cmd/server/main.go` is NOT modified (only `ports.go` new + `notifier.go` modified).

- [ ] **Step 4: Lint + commit**

Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/meeting_notifier/...` — `0 issues.`

```bash
git add apps/backend/internal/platform/meeting_notifier/ports.go \
  apps/backend/internal/platform/meeting_notifier/notifier.go
git commit -m "$(cat <<'EOF'
refactor(meeting_notifier): depend on sender/store interfaces

Extracts the bot + store ports so the send-orchestration is unit-testable.
Behavior-preserving: *bot.Bot and *postgres.Store satisfy the interfaces
(compile-time asserted), main.go is unchanged.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Orchestration tests

**Files:**
- Create: `apps/backend/internal/platform/meeting_notifier/notifier_test.go`

- [ ] **Step 1: Write fakes + tests**

```go
package meeting_notifier

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type sentMsg struct {
	ChatID int64
	Text   string
}

type fakeSender struct {
	sent []sentMsg
	err  error
}

func (f *fakeSender) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	cid, _ := p.ChatID.(int64)
	f.sent = append(f.sent, sentMsg{ChatID: cid, Text: p.Text})
	return &models.Message{}, f.err
}

var _ sender = (*fakeSender)(nil)

type fakeStore struct {
	meeting      postgres.Meeting
	getMeetErr   error
	org          postgres.Organization
	participants []postgres.MeetingParticipant
	byEmail      map[string]postgres.BotUser
	byTID        map[int64]postgres.BotUser
	orgTID       int64
	orgLinked    bool
	claimed      map[int64]bool
	claimErr     error
}

func (f *fakeStore) GetMeeting(_ context.Context, _, _ uuid.UUID) (postgres.Meeting, error) {
	return f.meeting, f.getMeetErr
}
func (f *fakeStore) GetOrganization(_ context.Context, _ uuid.UUID) (postgres.Organization, error) {
	return f.org, nil
}
func (f *fakeStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.participants, nil
}
func (f *fakeStore) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return postgres.BotUser{}, sql.ErrNoRows
}
func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, tid int64) (postgres.BotUser, error) {
	if u, ok := f.byTID[tid]; ok {
		return u, nil
	}
	return postgres.BotUser{}, sql.ErrNoRows
}
func (f *fakeStore) GetUserTelegramID(_ context.Context, _ uuid.UUID) (int64, bool, error) {
	return f.orgTID, f.orgLinked, nil
}
func (f *fakeStore) TryClaimReminder(_ context.Context, _ uuid.UUID, tid int64, _ int) (bool, error) {
	if f.claimErr != nil {
		return false, f.claimErr
	}
	return f.claimed[tid], nil
}

var _ store = (*fakeStore)(nil)

// baseStore seeds a meeting with one participant (tid 600) and a linked
// organizer (tid 500), so Resolve yields recipients [600, 500].
func baseStore() *fakeStore {
	org := uuid.New()
	return &fakeStore{
		meeting: postgres.Meeting{
			ID: uuid.New(), Name: "Sync", MeetLink: "https://meet.google.com/x",
			StartsAt: time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
			EndsAt:   time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
			OrganizerUserID: &org,
		},
		org:          postgres.Organization{TZ: "Asia/Almaty"},
		participants: []postgres.MeetingParticipant{{Email: "p@x.io"}},
		byEmail:      map[string]postgres.BotUser{"p@x.io": {TelegramID: 600, Email: "p@x.io"}},
		byTID:        map[int64]postgres.BotUser{500: {TelegramID: 500, Email: "org@x.io"}},
		orgTID:       500,
		orgLinked:    true,
	}
}

func newNotifier(st store, snd sender) *Notifier { return New(st, snd, zap.NewNop()) }

func TestHandleCreated_SendsToClaimedOnly(t *testing.T) {
	fs := baseStore()
	fs.claimed = map[int64]bool{600: true, 500: false} // only the participant is claimed
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCreated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("created: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 600 {
		t.Fatalf("want exactly one send to 600, got %+v", snd.sent)
	}
	if !strings.Contains(snd.sent[0].Text, "📅 Новая встреча") || !strings.Contains(snd.sent[0].Text, "Sync") {
		t.Fatalf("unexpected text: %q", snd.sent[0].Text)
	}
}

func TestHandleCreated_ClaimError_Aborts(t *testing.T) {
	fs := baseStore()
	fs.claimErr = errors.New("redis down")
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCreated(context.Background(), uuid.New(), fs.meeting.ID); err == nil {
		t.Fatal("expected claim error to propagate")
	}
}

func TestHandleParticipantAdded_RoutesToUser(t *testing.T) {
	fs := baseStore()
	fs.byEmail["new@x.io"] = postgres.BotUser{TelegramID: 700, Email: "new@x.io"}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantAdded(context.Background(), uuid.New(), fs.meeting.ID, "new@x.io"); err != nil {
		t.Fatalf("added: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 700 {
		t.Fatalf("want one send to 700, got %+v", snd.sent)
	}
	if !strings.Contains(snd.sent[0].Text, "➕ Вас добавили на встречу") {
		t.Fatalf("unexpected added text: %q", snd.sent[0].Text)
	}
}

func TestHandleParticipantRemoved_RoutesToUser(t *testing.T) {
	fs := baseStore()
	fs.byEmail["gone@x.io"] = postgres.BotUser{TelegramID: 800, Email: "gone@x.io"}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantRemoved(context.Background(), uuid.New(), fs.meeting.ID, "gone@x.io"); err != nil {
		t.Fatalf("removed: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 800 || !strings.Contains(snd.sent[0].Text, "➖") {
		t.Fatalf("unexpected removed send: %+v", snd.sent)
	}
}

func TestHandleParticipant_NotFound_NoSend(t *testing.T) {
	fs := baseStore() // "missing@x.io" is not in byEmail → sql.ErrNoRows → IsNotFound
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantAdded(context.Background(), uuid.New(), fs.meeting.ID, "missing@x.io"); err != nil {
		t.Fatalf("want nil on not-found, got %v", err)
	}
	if len(snd.sent) != 0 {
		t.Fatalf("want zero sends, got %+v", snd.sent)
	}
}

func TestHandleCancelled_AllRecipients(t *testing.T) {
	fs := baseStore()
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCancelled(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("cancelled: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 recipients, got %d: %+v", len(snd.sent), snd.sent)
	}
	for _, m := range snd.sent {
		if !strings.Contains(m.Text, "❌ Встреча отменена") {
			t.Fatalf("unexpected cancelled text: %q", m.Text)
		}
	}
}

func TestHandleUpdated_AllRecipients(t *testing.T) {
	fs := baseStore()
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleUpdated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("updated: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 recipients, got %d", len(snd.sent))
	}
	if !strings.Contains(snd.sent[0].Text, "✏️ Встреча изменена") {
		t.Fatalf("unexpected updated text: %q", snd.sent[0].Text)
	}
}

func TestHandleUpdated_TZFallback(t *testing.T) {
	fs := baseStore()
	fs.org = postgres.Organization{TZ: ""} // blank → fallback, must not error
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleUpdated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("tz fallback should not error: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 sends despite blank tz, got %d", len(snd.sent))
	}
}
```

NOTE: verify the `📅`/`➕`/`➖`/`❌`/`✏️` header strings against `message.go` byte-for-byte (copy from source if any assertion fails — the builders are correct).

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/platform/meeting_notifier/ -v` — all PASS (incl. WS2c's `message_test.go`).
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/platform/meeting_notifier/...` — `0 issues.`

If a test reveals a real behavior mismatch (e.g. cancelled/updated also gate on claims, or organizer dedup differs), REPORT it — do not weaken assertions blindly.

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/meeting_notifier/notifier_test.go
git commit -m "$(cat <<'EOF'
test(meeting_notifier): Handle* orchestration — claim dedup, routing, broadcasts, tz fallback

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Full verification

**Files:** none.

- [ ] **Step 1:** `cd apps/backend && env -u GOROOT go build ./...` — clean (main.go + all callers).
- [ ] **Step 2:** `cd apps/backend && env -u GOROOT go test -race ./...` — module-wide green (Docker permitting for the postgres package).
- [ ] **Step 3:** `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...` — `0 issues.`
- [ ] **Step 4:** `git status --short` — clean; confirm `cmd/server/main.go` was never modified.
- [ ] **Step 5 (informational):** after the human pushes, confirm CI is green (`gh run watch`).

---

## Notes on execution order
Task 1 (refactor) must precede Task 2 (the tests reference the new `store`/`sender` interfaces). Task 3 verifies the whole, including that `main.go` still compiles unchanged.
