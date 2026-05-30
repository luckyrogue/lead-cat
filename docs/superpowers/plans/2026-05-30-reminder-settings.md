# Reminder Settings (§5b-1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a registered bot user configure reminder intervals via `/settings` with an inline keyboard of toggle buttons; persist per-user on `bot_users.reminder_minutes`.

**Architecture:** A `botsettings` package with pure helpers (`parse`/`format`/`toggle`/`render`) + a `Service` over an injected store interface (testable without Telegram/DB). The existing `MultiHandler` gains callback-query handling: `/settings` renders the keyboard; `rem:<minutes>` callbacks toggle and edit the message.

**Tech Stack:** Go 1.26, pgx, go-telegram/bot (inline keyboards + callback queries). Spec: `docs/superpowers/specs/2026-05-30-reminder-settings-design.md`.

**Run from:** `backend/` with `env -u GOROOT go ...`.

---

### Task 1: Migration — reminder_minutes column

**Files:**
- Create: `backend/migrations/20260530150000_reminder_minutes.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS reminder_minutes TEXT NOT NULL DEFAULT '15';

-- +goose Down
ALTER TABLE bot_users DROP COLUMN IF EXISTS reminder_minutes;
```

- [ ] **Step 2: Apply and verify**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && set -a && . ./.env && set +a && cd backend && env -u GOROOT go run ./cmd/migrate up`
Expected: `OK 20260530150000_reminder_minutes.sql` and `successfully migrated database to version: 20260530150000`.

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260530150000_reminder_minutes.sql
git commit -m "feat(bot): reminder_minutes column on bot_users"
```

---

### Task 2: Model + repository — reminder_minutes

**Files:**
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go` (`BotUser` field)
- Modify: `backend/internal/infrastructure/persistence/postgres/bot_user_repo.go` (cols + scans + setter)

- [ ] **Step 1: Add the model field**

In `models.go`, add to the `BotUser` struct (after `Role`):
```go
	ReminderMinutes string `json:"reminder_minutes"`
```

- [ ] **Step 2: Include the column in reads + add the setter**

In `bot_user_repo.go`, replace the whole file with (adds `reminder_minutes` to `botUserCols`, scans it in all three queries, and adds `SetReminderMinutes`):
```go
package postgres

import (
	"context"
)

const botUserCols = `id, telegram_id, full_name, email, role, reminder_minutes`

func (s *Store) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE telegram_id = $1`, telegramID).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes)
	return u, err
}

func (s *Store) GetBotUserByEmail(ctx context.Context, email string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE email = $1`, email).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes)
	return u, err
}

func (s *Store) CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bot_users (telegram_id, full_name, email, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+botUserCols,
		telegramID, fullName, email, role).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes)
	return u, err
}

func (s *Store) SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bot_users SET reminder_minutes = $2 WHERE telegram_id = $1`, telegramID, csv)
	return err
}
```
(Note: the `uuid` import is intentionally gone — the file no longer references it; `BotUser.ID` is declared in models.go.)

- [ ] **Step 3: Build + test (existing botreg tests still pass)**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go test -count=1 ./internal/platform/botreg/...`
Expected: builds; botreg FSM tests still pass (they construct `postgres.BotUser` directly and don't depend on the new field).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/models.go backend/internal/infrastructure/persistence/postgres/bot_user_repo.go
git commit -m "feat(bot): persist reminder_minutes (read + set)"
```

---

### Task 3: botsettings package (pure helpers + service, TDD)

**Files:**
- Create: `backend/internal/platform/botsettings/settings.go`
- Create: `backend/internal/platform/botsettings/service.go`
- Test: `backend/internal/platform/botsettings/settings_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/platform/botsettings/settings_test.go`:
```go
package botsettings

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func TestParse(t *testing.T) {
	got := parse(" 60, 15 ,15, x, ,30 ")
	want := []int{15, 30, 60} // trimmed, numeric-only, deduped, sorted
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if len(parse("")) != 0 {
		t.Fatal("empty csv -> empty slice")
	}
}

func TestFormat(t *testing.T) {
	if format([]int{60, 15, 15}) != "15,60" {
		t.Fatalf("got %q", format([]int{60, 15, 15}))
	}
	if format(nil) != "" {
		t.Fatal("nil -> empty string")
	}
}

func TestToggle(t *testing.T) {
	if got := format(toggle([]int{15, 60}, 30)); got != "15,30,60" {
		t.Fatalf("add: %q", got)
	}
	if got := format(toggle([]int{15, 60}, 15)); got != "60" {
		t.Fatalf("remove: %q", got)
	}
}

func TestRender(t *testing.T) {
	text, kb := render([]int{15})
	if !strings.Contains(text, "Напоминан") {
		t.Fatalf("text: %q", text)
	}
	// 6 intervals laid out in rows of 3 => 2 rows.
	var flat []Button
	for _, row := range kb {
		flat = append(flat, row...)
	}
	if len(flat) != 6 {
		t.Fatalf("want 6 buttons, got %d", len(flat))
	}
	var on15 Button
	for _, b := range flat {
		if b.Data == "rem:15" {
			on15 = b
		}
	}
	if !strings.HasPrefix(on15.Text, "✓") {
		t.Fatalf("15 should be checked: %q", on15.Text)
	}
	emptyText, _ := render(nil)
	if !strings.Contains(emptyText, "выключены") {
		t.Fatalf("empty state text: %q", emptyText)
	}
}

// --- service with a fake store ---

type fakeStore struct {
	csv string
	err error
}

func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	if f.err != nil {
		return postgres.BotUser{}, f.err
	}
	return postgres.BotUser{ReminderMinutes: f.csv}, nil
}
func (f *fakeStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	f.csv = csv
	return nil
}

func TestServiceToggle(t *testing.T) {
	fs := &fakeStore{csv: "15"}
	svc := New(fs)
	if _, _, err := svc.Toggle(context.Background(), 1, 60); err != nil {
		t.Fatal(err)
	}
	if fs.csv != "15,60" {
		t.Fatalf("persisted %q", fs.csv)
	}
}

func TestServiceSettingsError(t *testing.T) {
	svc := New(&fakeStore{err: errors.New("db")})
	if _, _, err := svc.Settings(context.Background(), 1); err == nil {
		t.Fatal("expected error propagated")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/platform/botsettings/ -v`
Expected: FAIL — `parse`/`render`/`New`/`Button` undefined.

- [ ] **Step 3: Write the pure helpers**

`backend/internal/platform/botsettings/settings.go`:
```go
// Package botsettings renders and updates a bot user's reminder intervals.
// The pure helpers (parse/format/toggle/render) have no IO and are unit-tested.
package botsettings

import (
	"slices"
	"strconv"
	"strings"
)

// Button is a transport-agnostic inline-keyboard button (mapped to Telegram
// in the delivery layer).
type Button struct {
	Text string
	Data string
}

// Interval is a selectable reminder offset.
type Interval struct {
	Minutes int
	Label   string
}

// Intervals are the offered reminder offsets (ТЗ §5.2).
var Intervals = []Interval{
	{10, "10м"}, {15, "15м"}, {30, "30м"}, {60, "1ч"}, {120, "2ч"}, {1440, "1день"},
}

// parse turns the stored CSV into a sorted, de-duplicated slice of minutes.
func parse(csv string) []int {
	var out []int
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

// format renders minutes back to the canonical CSV (sorted, comma-joined).
func format(mins []int) string {
	cp := append([]int(nil), mins...)
	slices.Sort(cp)
	parts := make([]string, 0, len(cp))
	for _, m := range cp {
		parts = append(parts, strconv.Itoa(m))
	}
	return strings.Join(parts, ",")
}

// toggle adds v if absent, removes it if present.
func toggle(cur []int, v int) []int {
	if i := slices.Index(cur, v); i >= 0 {
		return slices.Delete(append([]int(nil), cur...), i, i+1)
	}
	return append(append([]int(nil), cur...), v)
}

// render builds the message text + keyboard (rows of 3), checking enabled intervals.
func render(mins []int) (string, [][]Button) {
	text := "⏰ Напоминания о встречах. Выбери, за сколько предупреждать (можно несколько):"
	if len(mins) == 0 {
		text += "\nСейчас напоминания выключены."
	}
	var rows [][]Button
	var row []Button
	for _, iv := range Intervals {
		label := iv.Label
		if slices.Contains(mins, iv.Minutes) {
			label = "✓ " + label
		}
		row = append(row, Button{Text: label, Data: "rem:" + strconv.Itoa(iv.Minutes)})
		if len(row) == 3 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return text, rows
}
```

- [ ] **Step 4: Write the service**

`backend/internal/platform/botsettings/service.go`:
```go
package botsettings

import (
	"context"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type store interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
}

type Service struct{ store store }

func New(s store) *Service { return &Service{store: s} }

// Settings renders the current reminder keyboard for a user.
func (s *Service) Settings(ctx context.Context, telegramID int64) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	text, kb := render(parse(u.ReminderMinutes))
	return text, kb, nil
}

// Toggle flips one interval, persists, and returns the refreshed keyboard.
func (s *Service) Toggle(ctx context.Context, telegramID int64, minutes int) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	next := toggle(parse(u.ReminderMinutes), minutes)
	if err := s.store.SetReminderMinutes(ctx, telegramID, format(next)); err != nil {
		return "", nil, err
	}
	text, kb := render(next)
	return text, kb, nil
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/platform/botsettings/ -v`
Expected: PASS (all tests).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/botsettings/
git commit -m "feat(bot): reminder settings service (parse/toggle/render)"
```

---

### Task 4: Wire /settings + callback handling into the bot

**Files:**
- Modify: `backend/internal/infrastructure/telegram/multitenant.go`

- [ ] **Step 1: Add the settings service to MultiHandler**

In `multitenant.go`, add imports `"strconv"` and `"github.com/Jaryq-Lab/notify-bot/internal/platform/botsettings"`. Add a field + build it in the constructor:
```go
type MultiHandler struct {
	store     *postgres.Store
	executor  *scenario_executor.Executor
	registrar *botreg.Service
	settings  *botsettings.Service
	log       *zap.Logger
}
```
In `NewMultiHandler`, after building `registrar`, add `settings := botsettings.New(store)` and set `settings: settings` in the returned struct literal.

- [ ] **Step 2: Handle callback queries + /settings in Handle**

At the very top of `Handle` (before the `if update.Message == nil` check), add:
```go
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, b, update.CallbackQuery)
		return
	}
```
Then add a `/settings` case to the command `switch` (alongside `/start` etc.):
```go
	case "/settings":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			text, kb, serr := h.settings.Settings(ctx, from.ID)
			if serr == nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      chatID,
					Text:        text,
					ReplyMarkup: toInlineMarkup(kb),
				})
			}
		}
```

- [ ] **Step 3: Add the callback handler + keyboard mapper**

Append these functions to `multitenant.go`:
```go
func (h *MultiHandler) handleCallback(ctx context.Context, b *bot.Bot, cq *models.CallbackQuery) {
	if strings.HasPrefix(cq.Data, "rem:") {
		if v, err := strconv.Atoi(strings.TrimPrefix(cq.Data, "rem:")); err == nil {
			text, kb, serr := h.settings.Toggle(ctx, cq.From.ID, v)
			if serr == nil && cq.Message.Message != nil {
				_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:      cq.Message.Message.Chat.ID,
					MessageID:   cq.Message.Message.ID,
					Text:        text,
					ReplyMarkup: toInlineMarkup(kb),
				})
			}
		}
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID})
}

func toInlineMarkup(rows [][]botsettings.Button) models.InlineKeyboardMarkup {
	out := make([][]models.InlineKeyboardButton, 0, len(rows))
	for _, r := range rows {
		row := make([]models.InlineKeyboardButton, 0, len(r))
		for _, btn := range r {
			row = append(row, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		out = append(out, row)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: out}
}
```

- [ ] **Step 4: Build, vet, test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -count=1 ./...`
Expected: all green. (If the compiler rejects `ReplyMarkup: toInlineMarkup(kb)` because the field wants the value differently, the mapper return type already matches `models.ReplyMarkup`; no change expected.)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/telegram/multitenant.go
git commit -m "feat(bot): /settings inline keyboard + reminder toggle callbacks"
```

---

### Task 5: Docs

**Files:**
- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

In the Backend section, after the bot-registration note, add:
```markdown
> **Reminder settings (done):** `/settings` shows an inline keyboard (10м/15м/30м/1ч/2ч/1день); tapping toggles the interval and saves it to `bot_users.reminder_minutes` (CSV, default `15`, empty = off). The reminder **engine** that sends them (§5b-2) is the next increment.
```

- [ ] **Step 2: Format and commit**

Run `make fmt-check` (run `make fmt` if it reflows docs; stage only this file).
```bash
git add docs/MEETINGS.md
git commit -m "docs(bot): document /settings reminder configuration"
```

---

## Done criteria

- `make lint` → 0 issues; `make test` → all pass (incl. `botsettings` suite + unchanged `botreg`); `make typecheck` → 0; `make fmt-check` → green; `make build`.
- `botsettings` unit tests cover parse (dedupe/sort/garbage), format, toggle (add/remove), render (✓ + empty state), and the service (toggle persists canonical CSV, error propagation).
- Manual (out of CI, real `BOT_TOKEN` polling): `/settings` shows the keyboard; tapping `15м` toggles the ✓ and the message updates; the value persists across `/settings` calls.
