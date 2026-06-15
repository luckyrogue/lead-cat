# Bot Commands Implementation Plan (Phase 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Telegram bot commands `/menu`, `/new`, `/help`, `/admin` and publish the command list via `SetMyCommands`.

**Architecture:** Pure, testable helpers (command list, URL join, WebApp keyboard, copy) live in a new `commands.go` in the telegram package; the existing `MultiHandler.Handle` switch dispatches the four new commands by calling `b.SendMessage` with those helpers. `/menu` and `/new` open the Mini App via a WebApp inline button (`/new` points directly at the `meetings/create` route — no start-param routing needed). `/admin` is gated by the bot-admin set. `SetMyCommands` is called once at startup in `main.go` (only when a real bot token is configured).

**Tech Stack:** Go, `github.com/go-telegram/bot` (+ `/models`), existing `MultiHandler` in `internal/infrastructure/telegram`.

**Scope notes:**
- Backend only. The spec floated a mini-app start-param change; it is NOT needed because a WebApp button carries a full URL (`WebappURL + "/meetings/create"`), and the mini-app SPA already serves that route.
- Bot copy is Russian, matching the existing bot strings (e.g. `"Сначала зарегистрируйся: /start"`).
- `/admin` opens the Mini App home for admins (the mini-app has no dedicated admin route today); non-admins get a denial. A dedicated admin deep-link can come later.

**Prerequisite:** Branch `feat/mini-app-meeting-parity`, on top of Phase 1 (HEAD `e25dd0e`). Run Go from `apps/backend` with `env -u GOROOT go ...`. Stage explicit paths only; never `git add -A`; never touch `.gitignore`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Ignore IDE diagnostics (stale LSP from an in-flight refactor) — trust `go` command output.

---

## File Structure

- Create: `apps/backend/internal/infrastructure/telegram/commands.go` — pure helpers: `PublicCommands`, `webAppMarkup`, `joinURL`, `helpText`.
- Create: `apps/backend/internal/infrastructure/telegram/commands_test.go` — table tests for the pure helpers.
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` — `MultiHandler` gains `webappURL` + `admins`; `NewMultiHandler` takes `webappURL`; four new `switch` cases.
- Modify: `apps/backend/cmd/server/main.go` — pass `cfg.WebappURL` to `NewMultiHandler`; call `SetMyCommands`.

---

## Task 1: Pure command helpers + tests

**Files:**
- Create: `apps/backend/internal/infrastructure/telegram/commands.go`
- Test: `apps/backend/internal/infrastructure/telegram/commands_test.go`

- [ ] **Step 1: Write the failing test**

Create `apps/backend/internal/infrastructure/telegram/commands_test.go`:

```go
package telegram

import (
	"strings"
	"testing"
)

func TestPublicCommands(t *testing.T) {
	cmds := PublicCommands()
	if len(cmds) == 0 {
		t.Fatal("no commands")
	}
	want := map[string]bool{
		"menu": true, "new": true, "schedule": true,
		"checker": true, "settings": true, "edit": true, "help": true,
	}
	for _, c := range cmds {
		if strings.HasPrefix(c.Command, "/") {
			t.Fatalf("command %q must not include leading slash", c.Command)
		}
		if c.Command != strings.ToLower(c.Command) || c.Command == "" {
			t.Fatalf("command %q must be lowercase and non-empty", c.Command)
		}
		if c.Description == "" {
			t.Fatalf("command %q missing description", c.Command)
		}
		delete(want, c.Command)
	}
	if len(want) != 0 {
		t.Fatalf("missing commands: %v", want)
	}
}

func TestJoinURL(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"https://app.example.com", "meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com/", "meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com/", "/meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com", "", "https://app.example.com"},
	}
	for _, c := range cases {
		if got := joinURL(c.base, c.path); got != c.want {
			t.Fatalf("joinURL(%q,%q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

func TestWebAppMarkup(t *testing.T) {
	m := webAppMarkup("Открыть", "https://app.example.com")
	if len(m.InlineKeyboard) != 1 || len(m.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected 1x1 keyboard, got %v", m.InlineKeyboard)
	}
	btn := m.InlineKeyboard[0][0]
	if btn.Text != "Открыть" {
		t.Fatalf("text = %q", btn.Text)
	}
	if btn.WebApp == nil || btn.WebApp.URL != "https://app.example.com" {
		t.Fatalf("web app url = %v", btn.WebApp)
	}
}

func TestHelpText(t *testing.T) {
	h := helpText()
	for _, sub := range []string{"/menu", "/new", "/schedule", "/checker", "/settings", "/edit"} {
		if !strings.Contains(h, sub) {
			t.Fatalf("help text missing %q", sub)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/telegram/ -run 'TestPublicCommands|TestJoinURL|TestWebAppMarkup|TestHelpText'`
Expected: FAIL — `undefined: PublicCommands` (and the others).

- [ ] **Step 3: Implement the helpers**

Create `apps/backend/internal/infrastructure/telegram/commands.go`:

```go
package telegram

import (
	"strings"

	"github.com/go-telegram/bot/models"
)

// PublicCommands is the bot command list published to Telegram via SetMyCommands.
// Command values must be lowercase and without a leading slash.
func PublicCommands() []models.BotCommand {
	return []models.BotCommand{
		{Command: "menu", Description: "Открыть приложение"},
		{Command: "new", Description: "Запланировать встречу"},
		{Command: "schedule", Description: "Расписание коллеги"},
		{Command: "checker", Description: "Найти свободный слот"},
		{Command: "settings", Description: "Напоминания"},
		{Command: "edit", Description: "Редактировать встречи"},
		{Command: "help", Description: "Помощь"},
	}
}

// webAppMarkup builds a one-button inline keyboard that opens the Mini App at url.
func webAppMarkup(text, url string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: text, WebApp: &models.WebAppInfo{URL: url}},
	}}}
}

// joinURL joins a base URL and a path, tolerating slashes on either side.
func joinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	if path == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

// helpText is the /help reply describing the available commands.
func helpText() string {
	return strings.Join([]string{
		"Lead Cat — помощник по встречам 🐾",
		"",
		"/menu — открыть приложение",
		"/new — запланировать встречу",
		"/schedule — расписание коллеги",
		"/checker — найти свободный слот",
		"/settings — настроить напоминания",
		"/edit — редактировать свои встречи",
		"/help — это сообщение",
	}, "\n")
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/telegram/ -run 'TestPublicCommands|TestJoinURL|TestWebAppMarkup|TestHelpText'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/infrastructure/telegram/commands.go \
  apps/backend/internal/infrastructure/telegram/commands_test.go
git commit -m "feat(bot): pure helpers for bot commands (list, webapp button, help)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Wire the dispatcher + startup

**Files:**
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go`
- Modify: `apps/backend/cmd/server/main.go`

- [ ] **Step 1: Add fields to `MultiHandler`**

In `apps/backend/internal/infrastructure/telegram/multitenant.go`, add two fields to the `MultiHandler` struct (after `log *zap.Logger`):

```go
	webappURL string
	admins    map[int64]bool
```

- [ ] **Step 2: Update `NewMultiHandler` to take `webappURL` and build the admins set**

Change the `NewMultiHandler` signature to add a `webappURL string` parameter (right after `adminIDs []int64`), build an `admins` map from `adminIDs`, and set both new fields. The function becomes:

```go
func NewMultiHandler(store *postgres.Store, b *bot.Bot, rdb *redis.Client, adminIDs []int64, webappURL string, backend botBackend, log *zap.Logger) *MultiHandler {
	registrar := botreg.New(store, botreg.NewRedisSessions(rdb), adminIDs)
	settings := botsettings.New(store)
	editor := meetingedit.New(backend, meetingedit.NewRedisSessions(rdb))
	schedule := scheduleview.New(backend, scheduleview.NewRedisSessions(rdb))
	chk := checker.New(backend, checker.NewRedisSessions(rdb))
	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}
	return &MultiHandler{
		store:     store,
		registrar: registrar,
		settings:  settings,
		editor:    editor,
		schedule:  schedule,
		checker:   chk,
		log:       log,
		webappURL: webappURL,
		admins:    admins,
	}
}
```

- [ ] **Step 3: Add the four command cases**

In the `switch cmd {` block in `Handle` (the file already has `chatID := update.Message.Chat.ID`, `from := update.Message.From`, and `isPrivate` in scope), add these cases (e.g. after the `case "/checker":` block, before the closing `}` of the switch):

```go
	case "/menu":
		if isPrivate {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        "Открой Lead Cat 🐾",
				ReplyMarkup: webAppMarkup("Открыть приложение", h.webappURL),
			})
		}
	case "/new":
		if isPrivate {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        "Запланируй встречу 🐾",
				ReplyMarkup: webAppMarkup("Новая встреча", joinURL(h.webappURL, "meetings/create")),
			})
		}
	case "/help":
		if isPrivate {
			h.reply(ctx, b, update.Message, helpText())
		}
	case "/admin":
		if isPrivate {
			if h.admins[from.ID] {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      chatID,
					Text:        "Ты администратор 🐾 Настройки — в приложении.",
					ReplyMarkup: webAppMarkup("Открыть приложение", h.webappURL),
				})
			} else {
				h.reply(ctx, b, update.Message, "Ты не администратор 🐾")
			}
		}
```

- [ ] **Step 4: Update the `NewMultiHandler` call + add `SetMyCommands` in `main.go`**

In `apps/backend/cmd/server/main.go`, inside the `if cfg.RealBotToken() {` block, change the `NewMultiHandler` call to pass `cfg.WebappURL`:

```go
		tgHandler = telegram.NewMultiHandler(store, tg, rdb, cfg.BotAdminTelegramIDs, cfg.WebappURL, services, logger)
```

Immediately after that line (still inside the `if cfg.RealBotToken()` block), publish the command list:

```go
		if _, cerr := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: telegram.PublicCommands()}); cerr != nil {
			logger.Warn("set_my_commands", zap.Error(cerr))
		}
```

(`bot`, `telegram`, `zap`, and `ctx` are already imported/in scope in main.go.)

- [ ] **Step 5: Build + test + lint**

Run: `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./... && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/telegram/ ./cmd/...`
Expected: build clean; all tests `ok`; `0 issues` on the touched packages. (If another caller of `NewMultiHandler` exists, e.g. a test, update it to pass a `webappURL` argument — report which files. If golangci-lint is not installed, report and skip just the lint.)

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/infrastructure/telegram/multitenant.go \
  apps/backend/cmd/server/main.go
git commit -m "feat(bot): /menu /new /help /admin commands + SetMyCommands

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Final verification

- [ ] **Step 1: Full backend gate**

Run: `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
Expected: build clean; all packages `ok`.

- [ ] **Step 2: Confirm command coverage**

Run: `grep -nE '"/menu"|"/new"|"/help"|"/admin"' apps/backend/internal/infrastructure/telegram/multitenant.go && grep -n "SetMyCommands" apps/backend/cmd/server/main.go`
Expected: the four cases present in the dispatcher and the `SetMyCommands` call present in main.go.

- [ ] **Step 3: Confirm clean tree**

Run: `git status --short`
Expected: no unexpected files (only the four files created/modified across the two task commits; `.gitignore`/`bin/` untouched).

---

## Notes & decisions

- **No mini-app change**: WebApp buttons carry a full URL, so `/new` deep-links to `meetings/create` directly; the mini-app start-param routing the spec mentioned is unnecessary.
- **WebApp buttons require HTTPS in production** (Telegram constraint). With a local `WEBAPP_URL=http://localhost:3000`, the buttons may not open in the Telegram client — this is a runtime/config matter, not a code defect.
- **`SetMyCommands` only runs with a real bot token** (it's inside the existing `if cfg.RealBotToken()` block), so dev mode with a fake token is unaffected.
- **`/admin` opens the app home** for admins (no dedicated mini-app admin route exists yet) and denies non-admins; a deep-link to an admin screen is a future nicety.
- **`/start`, `/chatid`** are intentionally not in `PublicCommands` (start is implicit; chatid is an internal setup helper).
