# Reminder Engine (§5b-2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A 1-minute scheduler that DMs upcoming-meeting reminders over Telegram — to registered participants by their configured intervals and to the organizer by a 15-minute default — each sent at most once via durable dedup.

**Architecture:** A `reminder_scheduler` package mirroring `scenario_scheduler` (ticker + Redis leader lock). Pure helpers (`dueOffsets`/`offsetLabel`/`message`) are unit-tested; the tick reads meetings/participants/bot_users, claims each reminder atomically (`INSERT … ON CONFLICT DO NOTHING`), and sends via `bot.SendMessage`.

**Tech Stack:** Go 1.26, pgx, go-redis, go-telegram/bot. Spec: `docs/superpowers/specs/2026-05-30-reminder-engine-design.md`.

**Run from:** `backend/` with `env -u GOROOT go ...`.

---

### Task 1: Migration — meeting_reminders dedup table

**Files:**

- Create: `backend/migrations/20260530160000_meeting_reminders.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE meeting_reminders (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    telegram_id BIGINT NOT NULL,
    offset_minutes INT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (meeting_id, telegram_id, offset_minutes)
);

-- +goose Down
DROP TABLE IF EXISTS meeting_reminders;
```

- [ ] **Step 2: Apply and verify**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && set -a && . ./.env && set +a && cd backend && env -u GOROOT go run ./cmd/migrate up`
Expected: `OK 20260530160000_meeting_reminders.sql` and `successfully migrated database to version: 20260530160000`.

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260530160000_meeting_reminders.sql
git commit -m "feat(reminders): meeting_reminders dedup table"
```

---

### Task 2: Store — upcoming meetings, claim, organizer TG

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go` (add 2 methods + `time` import)
- Modify: `backend/internal/infrastructure/persistence/postgres/user_repo.go` (add `GetUserTelegramID`)

(No DB unit test — package has no DB harness; build-verified, exercised manually.)

- [ ] **Step 1: Add upcoming-meetings + claim to meeting_repo.go**

In `meeting_repo.go`, change the import block from `import ( "context"; "github.com/google/uuid" )` to also import `"time"`:

```go
import (
	"context"
	"time"

	"github.com/google/uuid"
)
```

Then append these two methods (they reuse the existing `meetingCols` const and `queryMeetings` helper):

```go
func (s *Store) ListUpcomingMeetings(ctx context.Context, until time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE status = 'scheduled' AND starts_at > now() AND starts_at <= $1
		ORDER BY starts_at`, until)
}

// TryClaimReminder atomically records that (meeting, telegram, offset) is being
// reminded. Returns true if this call claimed it (caller should send), false if
// it was already claimed.
func (s *Store) TryClaimReminder(ctx context.Context, meetingID uuid.UUID, telegramID int64, offset int) (bool, error) {
	ct, err := s.pool.Exec(ctx, `
		INSERT INTO meeting_reminders (meeting_id, telegram_id, offset_minutes)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, meetingID, telegramID, offset)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}
```

- [ ] **Step 2: Add GetUserTelegramID to user_repo.go**

Append to `backend/internal/infrastructure/persistence/postgres/user_repo.go` (it already imports `context` and `github.com/google/uuid`):

```go
// GetUserTelegramID returns the platform user's linked Telegram id. ok is false
// when the user exists but has not linked Telegram.
func (s *Store) GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error) {
	var tg *int64
	err := s.pool.QueryRow(ctx, `SELECT telegram_id FROM platform_users WHERE id = $1`, userID).Scan(&tg)
	if err != nil {
		return 0, false, err
	}
	if tg == nil {
		return 0, false, nil
	}
	return *tg, true, nil
}
```

- [ ] **Step 3: Build**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./...`
Expected: builds. (If `user_repo.go` does not already import `uuid`, add it — but it does, as existing methods take `uuid.UUID`.)

- [ ] **Step 4: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go backend/internal/infrastructure/persistence/postgres/user_repo.go
git commit -m "feat(reminders): store upcoming meetings, claim, organizer TG id"
```

---

### Task 3: Pure helpers + export botsettings.Parse (TDD)

**Files:**

- Modify: `backend/internal/platform/botsettings/settings.go` (export `Parse`)
- Create: `backend/internal/platform/reminder_scheduler/reminder.go`
- Test: `backend/internal/platform/reminder_scheduler/reminder_test.go`

- [ ] **Step 1: Export Parse from botsettings**

In `backend/internal/platform/botsettings/settings.go`, add (below the unexported `parse`):

```go
// Parse exposes the reminder-minutes CSV parser for other packages.
func Parse(csv string) []int { return parse(csv) }
```

- [ ] **Step 2: Write the failing test**

`backend/internal/platform/reminder_scheduler/reminder_test.go`:

```go
package reminder_scheduler

import (
	"strings"
	"testing"
	"time"
)

func TestDueOffsets(t *testing.T) {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)

	// 16 minutes before start: only the 15-min offset is due (10 not yet).
	now := start.Add(-16 * time.Minute)
	got := dueOffsets(now, start, []int{10, 15, 30})
	if len(got) != 1 || got[0] != 15 {
		t.Fatalf("at -16m want [15], got %v", got)
	}

	// 9 minutes before: 10 and 15 and 30 all crossed.
	now = start.Add(-9 * time.Minute)
	got = dueOffsets(now, start, []int{10, 15, 30})
	if len(got) != 3 {
		t.Fatalf("at -9m want 3 offsets, got %v", got)
	}

	// 40 minutes before: none crossed.
	now = start.Add(-40 * time.Minute)
	if got := dueOffsets(now, start, []int{10, 15, 30}); len(got) != 0 {
		t.Fatalf("at -40m want none, got %v", got)
	}

	// after start: nothing fires.
	now = start.Add(1 * time.Minute)
	if got := dueOffsets(now, start, []int{10, 15, 30}); len(got) != 0 {
		t.Fatalf("after start want none, got %v", got)
	}
}

func TestOffsetLabel(t *testing.T) {
	cases := map[int]string{10: "10 минут", 60: "1 час", 120: "2 часа", 1440: "1 день", 45: "45 минут"}
	for min, want := range cases {
		if got := offsetLabel(min); got != want {
			t.Fatalf("offsetLabel(%d)=%q want %q", min, got, want)
		}
	}
}

func TestMessage(t *testing.T) {
	m := message("Разработка | Планёрка", "https://meet.google.com/abc", 15)
	if !strings.Contains(m, "через 15 минут") || !strings.Contains(m, "Разработка | Планёрка") || !strings.Contains(m, "https://meet.google.com/abc") {
		t.Fatalf("bad message: %q", m)
	}
	if strings.Contains(message("X", "", 15), "🔗") {
		t.Fatal("no link line when meet link empty")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/platform/reminder_scheduler/ -v`
Expected: FAIL — `dueOffsets`/`offsetLabel`/`message` undefined.

- [ ] **Step 4: Write the helpers**

`backend/internal/platform/reminder_scheduler/reminder.go`:

```go
// Package reminder_scheduler sends Telegram reminders for upcoming meetings.
package reminder_scheduler

import (
	"fmt"
	"strconv"
	"time"
)

// dueOffsets returns the offsets (minutes) whose reminder time has arrived:
// now is at or past (startsAt - offset) and the meeting has not started yet.
func dueOffsets(now, startsAt time.Time, offsets []int) []int {
	var due []int
	for _, off := range offsets {
		threshold := startsAt.Add(-time.Duration(off) * time.Minute)
		if !now.Before(threshold) && now.Before(startsAt) {
			due = append(due, off)
		}
	}
	return due
}

func offsetLabel(min int) string {
	switch min {
	case 10:
		return "10 минут"
	case 15:
		return "15 минут"
	case 30:
		return "30 минут"
	case 60:
		return "1 час"
	case 120:
		return "2 часа"
	case 1440:
		return "1 день"
	default:
		return strconv.Itoa(min) + " минут"
	}
}

func message(name, meetLink string, offset int) string {
	m := fmt.Sprintf("⏰ Напоминание: встреча через %s!\n«%s»", offsetLabel(offset), name)
	if meetLink != "" {
		m += "\n🔗 " + meetLink
	}
	return m
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/platform/reminder_scheduler/ -v && env -u GOROOT go build ./...`
Expected: PASS; builds.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/botsettings/settings.go backend/internal/platform/reminder_scheduler/reminder.go backend/internal/platform/reminder_scheduler/reminder_test.go
git commit -m "feat(reminders): pure due/label/message helpers + botsettings.Parse"
```

---

### Task 4: Scheduler + wiring

**Files:**

- Create: `backend/internal/platform/reminder_scheduler/scheduler.go`
- Modify: `backend/cmd/server/main.go` (start the scheduler)

- [ ] **Step 1: Write the scheduler**

`backend/internal/platform/reminder_scheduler/scheduler.go`:

```go
package reminder_scheduler

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/botsettings"
)

const lockKey = "leadcat:reminders:leader"

// defaultOrganizerOffsets is used for the organizer (who has no bot_users
// reminder settings).
var defaultOrganizerOffsets = []int{15}

type Scheduler struct {
	store *postgres.Store
	bot   *bot.Bot
	rdb   *redis.Client
	log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, rdb *redis.Client, log *zap.Logger) *Scheduler {
	return &Scheduler{store: store, bot: b, rdb: rdb, log: log}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	ok, err := s.rdb.SetNX(ctx, lockKey, "1", 90*time.Second).Result()
	if err != nil || !ok {
		return
	}
	now := time.Now()
	meetings, err := s.store.ListUpcomingMeetings(ctx, now.Add(24*time.Hour))
	if err != nil {
		s.log.Warn("list upcoming meetings", zap.Error(err))
		return
	}
	for _, m := range meetings {
		recipients := s.recipients(ctx, m)
		for tg, offsets := range recipients {
			for _, off := range dueOffsets(now, m.StartsAt, offsets) {
				claimed, err := s.store.TryClaimReminder(ctx, m.ID, tg, off)
				if err != nil || !claimed {
					continue
				}
				_, _ = s.bot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: tg,
					Text:   message(m.Name, m.MeetLink, off),
				})
			}
		}
	}
}

// recipients maps telegram_id -> reminder offsets: registered participants use
// their own settings; the organizer (if linked and not already a participant)
// uses the default.
func (s *Scheduler) recipients(ctx context.Context, m postgres.Meeting) map[int64][]int {
	out := map[int64][]int{}
	parts, err := s.store.ListParticipants(ctx, m.ID)
	if err != nil {
		return out
	}
	for _, p := range parts {
		if p.Email == "" {
			continue
		}
		u, err := s.store.GetBotUserByEmail(ctx, p.Email)
		if err != nil {
			continue
		}
		out[u.TelegramID] = botsettings.Parse(u.ReminderMinutes)
	}
	if m.OrganizerUserID != nil {
		if tg, linked, err := s.store.GetUserTelegramID(ctx, *m.OrganizerUserID); err == nil && linked {
			if _, exists := out[tg]; !exists {
				out[tg] = defaultOrganizerOffsets
			}
		}
	}
	return out
}
```

- [ ] **Step 2: Start it in cmd/server**

In `backend/cmd/server/main.go`, find:

```go
	sched := scenario_scheduler.New(store, queueClient, rdb, logger)
	go sched.Run(ctx)
```

and add immediately after:

```go
	remSched := reminder_scheduler.New(store, tg, rdb, logger)
	go remSched.Run(ctx)
```

Add the import to main.go's import block:

```go
	"github.com/Jaryq-Lab/notify-bot/internal/platform/reminder_scheduler"
```

- [ ] **Step 3: Build, vet, test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -count=1 ./...`
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/platform/reminder_scheduler/scheduler.go backend/cmd/server/main.go
git commit -m "feat(reminders): meeting reminder scheduler (participants + organizer)"
```

---

### Task 5: Docs

**Files:**

- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

In the Backend section, after the "Reminder settings (done)" note, add:

```markdown
> **Reminder engine (done):** a 1-minute scheduler (Redis leader lock) DMs upcoming-meeting reminders — registered participants by their `/settings` intervals, plus the organizer (linked Telegram) at a 15-minute default. Durable dedup via `meeting_reminders` (one DM per meeting/user/offset). Best-effort send; needs the bot polling.
```

- [ ] **Step 2: Format and commit**

Run `make fmt-check` (run `make fmt` if it reflows docs; stage only this file).

```bash
git add docs/MEETINGS.md
git commit -m "docs(reminders): document the reminder engine"
```

---

## Done criteria

- `make lint` → 0 issues; `make test` → all pass (incl. the `reminder_scheduler` pure-helper suite); `make typecheck` → 0; `make fmt-check` → green; `make build`.
- Pure-helper unit tests cover `dueOffsets` (before/at/after threshold, multiple offsets, past meeting), `offsetLabel` (each interval + fallback), `message` (with/without Meet link).
- Manual (out of CI, real `BOT_TOKEN`): a meeting starting within a configured interval DMs each registered participant once at each due offset, plus the organizer at 15m if not a participant; a second tick sends nothing (dedup row present).
