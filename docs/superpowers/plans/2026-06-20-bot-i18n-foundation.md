# Bot i18n Foundation + Pilot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dependency-free `boti18n` translation-catalog package and migrate two pilot bot surfaces (`commands`/help and `botreg`) to it, proving both language-resolution paths (stored vs Telegram `language_code`).

**Architecture:** A new `internal/platform/boti18n` package holds `T(lang,key,args...)`, `Normalize`, `Resolve`, and a `map[key]map[lang]string` catalog assembled from per-domain files via `init()`/`register`. The Telegram dispatcher resolves the acting user's language once per update and passes it into `helpText(lang)`, `PublicCommands(lang)`, and the `botreg.Service` methods.

**Tech Stack:** Go, `github.com/go-telegram/bot` v1.21.0 (`models.User.LanguageCode`), zap (existing).

## Global Constraints

- **Languages:** `ru` | `en` | `kk`, default `ru`. `Normalize` strips a region subtag (`en-US` → `en`) then maps anything outside en/kk to ru.
- **Catalog keys** are namespaced by domain (`cmd.*`, `botreg.*`) and live in per-domain files (`catalog_commands.go`, `catalog_botreg.go`); `t.go` never changes when a domain is added.
- **Format params** use `fmt.Sprintf`; where a translation must reorder args, use explicit-index verbs (`%[1]s`).
- **Fallbacks:** missing key → return the key verbatim; key present but missing the language → fall back to the `ru` entry. Never panic.
- **Every catalog key MUST have all three of ru/en/kk** — enforced by a coverage test that fails CI on a half-translated key.
- **`ru` values are the existing Russian strings verbatim** (copy current text exactly); `en`/`kk` are new translations.
- **Handlers receive `lang string`** — `botreg` does no store/Telegram lookup for language; the dispatcher resolves and passes it (keeps handlers unit-testable).
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: `boti18n` package (mechanism + tests)

**Files:**
- Create: `apps/backend/internal/platform/boti18n/t.go`
- Test: `apps/backend/internal/platform/boti18n/t_test.go`

**Interfaces:**
- Produces (consumed by Tasks 2 & 3):
  ```go
  func Normalize(lang string) string                  // "ru"|"en"|"kk", default "ru"
  func Resolve(stored, telegramCode string) string     // stored wins, else telegramCode, else "ru"
  func T(lang, key string, args ...any) string         // catalog lookup + Sprintf
  func register(entries map[string]map[string]string)  // unexported; domain files call in init()
  ```

- [ ] **Step 1: Write the failing test `t_test.go`**

```go
package boti18n

import "testing"

// fixture keys registered for tests (all three languages present).
func init() {
	register(map[string]map[string]string{
		"test.hi":   {"ru": "Привет", "en": "Hi", "kk": "Сәлем"},
		"test.greet": {"ru": "Привет, %[1]s", "en": "Hi, %[1]s", "kk": "Сәлем, %[1]s"},
		"test.ruonly": {"ru": "ТолькоRU"}, // intentionally missing en/kk for fallback test
	})
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{"ru": "ru", "en": "en", "kk": "kk", "en-US": "en", "kk-KZ": "kk", "": "ru", "fr": "ru"}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve("en", "kk"); got != "en" {
		t.Errorf("stored should win: got %q", got)
	}
	if got := Resolve("", "kk"); got != "kk" {
		t.Errorf("empty stored should use telegram code: got %q", got)
	}
	if got := Resolve("", ""); got != "ru" {
		t.Errorf("both empty should be ru: got %q", got)
	}
	if got := Resolve("garbage", ""); got != "ru" {
		t.Errorf("garbage stored normalizes to ru: got %q", got)
	}
}

func TestT(t *testing.T) {
	if got := T("en", "test.hi"); got != "Hi" {
		t.Errorf("T en = %q", got)
	}
	if got := T("kk", "test.hi"); got != "Сәлем" {
		t.Errorf("T kk = %q", got)
	}
	if got := T("en", "test.greet", "Mia"); got != "Hi, Mia" {
		t.Errorf("T with arg = %q", got)
	}
	// missing language for a present key → ru fallback
	if got := T("en", "test.ruonly"); got != "ТолькоRU" {
		t.Errorf("missing-lang should fall back to ru: %q", got)
	}
	// missing key → key returned verbatim
	if got := T("en", "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key should return key: %q", got)
	}
}

func TestCatalog_AllKeysHaveAllLangs(t *testing.T) {
	for key, langs := range catalog {
		if key == "test.ruonly" {
			continue // intentional fixture gap for the fallback test
		}
		for _, l := range []string{"ru", "en", "kk"} {
			if langs[l] == "" {
				t.Errorf("catalog key %q missing %q translation", key, l)
			}
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/platform/boti18n/ 2>&1 | head`
Expected: FAIL — compile error (`undefined: register`, `Normalize`, `Resolve`, `T`, `catalog`).

- [ ] **Step 3: Implement `t.go`**

```go
// Package boti18n is a dependency-free translation catalog for Telegram bot text.
// Catalog entries are registered by per-domain files (catalog_*.go) via init().
package boti18n

import (
	"fmt"
	"strings"
)

// catalog maps a message key to a map of language → format string.
var catalog = map[string]map[string]string{}

// register merges domain entries into the catalog. Called from init() in
// per-domain catalog_*.go files.
func register(entries map[string]map[string]string) {
	for key, langs := range entries {
		catalog[key] = langs
	}
}

// Normalize returns a supported language code (ru|en|kk), defaulting to ru.
// A region subtag is stripped first ("en-US" → "en").
func Normalize(lang string) string {
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		lang = lang[:i]
	}
	switch lang {
	case "en", "kk":
		return lang
	default:
		return "ru"
	}
}

// Resolve picks the effective language: a non-empty stored preference wins
// (normalized); otherwise the Telegram language_code (normalized); otherwise ru.
func Resolve(stored, telegramCode string) string {
	if strings.TrimSpace(stored) != "" {
		return Normalize(stored)
	}
	return Normalize(telegramCode)
}

// T returns the catalog string for key in the given language, applying args via
// fmt.Sprintf. Missing key → key returned verbatim; key present but missing the
// language → ru fallback.
func T(lang, key string, args ...any) string {
	entry, ok := catalog[key]
	if !ok {
		return key
	}
	s, ok := entry[Normalize(lang)]
	if !ok {
		s = entry["ru"]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/backend && go test ./internal/platform/boti18n/ -v`
Expected: PASS (TestNormalize, TestResolve, TestT, TestCatalog_AllKeysHaveAllLangs).

- [ ] **Step 5: Vet + commit**

```bash
cd apps/backend && go vet ./internal/platform/boti18n/ && cd /Users/temirlan/Workspace/in-house/lead-cat
git add apps/backend/internal/platform/boti18n/t.go apps/backend/internal/platform/boti18n/t_test.go
git commit -m "$(cat <<'EOF'
feat(boti18n): dependency-free bot translation catalog (T/Normalize/Resolve)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: Localize `commands`/help (registered-user path) + dispatcher lang helper

Migrates `helpText` and `PublicCommands` descriptions to the catalog, and adds the reusable dispatcher `resolveLang` helper (consumed again in Task 3). The slash-command menu is registered per-language at startup; the `/help` reply is localized per update.

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_commands.go`
- Modify: `apps/backend/internal/infrastructure/telegram/commands.go`
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (help reply + `resolveLang` helper)
- Modify: `apps/backend/cmd/server/main.go` (register localized command menus)
- Test: `apps/backend/internal/infrastructure/telegram/commands_test.go`

**Interfaces:**
- Consumes (Task 1): `boti18n.T`, `boti18n.Resolve`.
- Produces (consumed by Task 3): `func (h *Handler) resolveLang(ctx context.Context, from *models.User) string` on the telegram dispatcher.

- [ ] **Step 1: Create `catalog_commands.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"cmd.menu":     {"ru": "Открыть приложение", "en": "Open the app", "kk": "Қосымшаны ашу"},
		"cmd.new":      {"ru": "Запланировать встречу", "en": "Schedule a meeting", "kk": "Кездесу жоспарлау"},
		"cmd.schedule": {"ru": "Расписание коллеги", "en": "Colleague's schedule", "kk": "Әріптестің кестесі"},
		"cmd.checker":  {"ru": "Найти свободный слот", "en": "Find a free slot", "kk": "Бос уақыт табу"},
		"cmd.settings": {"ru": "Напоминания", "en": "Reminders", "kk": "Еске салулар"},
		"cmd.edit":     {"ru": "Редактировать встречи", "en": "Edit meetings", "kk": "Кездесулерді өңдеу"},
		"cmd.help":     {"ru": "Помощь", "en": "Help", "kk": "Көмек"},
		"cmd.help.text": {
			"ru": "Lead Cat — помощник по встречам 🐾\n\n/menu — открыть приложение\n/new — запланировать встречу\n/schedule — расписание коллеги\n/checker — найти свободный слот\n/settings — настроить напоминания\n/edit — редактировать свои встречи\n/help — это сообщение",
			"en": "Lead Cat — your meetings assistant 🐾\n\n/menu — open the app\n/new — schedule a meeting\n/schedule — a colleague's schedule\n/checker — find a free slot\n/settings — configure reminders\n/edit — edit your meetings\n/help — this message",
			"kk": "Lead Cat — кездесу көмекшісі 🐾\n\n/menu — қосымшаны ашу\n/new — кездесу жоспарлау\n/schedule — әріптестің кестесі\n/checker — бос уақыт табу\n/settings — еске салуларды баптау\n/edit — кездесулеріңді өңдеу\n/help — осы хабарлама",
		},
	})
}
```

- [ ] **Step 2: Write the failing test `commands_test.go`**

```go
package telegram

import (
	"strings"
	"testing"
)

func TestHelpText_Localized(t *testing.T) {
	ru := helpText("ru")
	en := helpText("en")
	kk := helpText("kk")
	if ru == "" || en == "" || kk == "" {
		t.Fatal("help text must be non-empty in all languages")
	}
	if en == ru || kk == ru {
		t.Fatal("help text must differ by language")
	}
	if !strings.Contains(en, "/menu") || !strings.Contains(en, "Lead Cat") {
		t.Errorf("en help text malformed: %q", en)
	}
}

func TestPublicCommands_Localized(t *testing.T) {
	en := PublicCommands("en")
	ru := PublicCommands("ru")
	if len(en) != len(ru) || len(en) == 0 {
		t.Fatalf("command lists must be same non-zero length: en=%d ru=%d", len(en), len(ru))
	}
	if en[0].Description == ru[0].Description {
		t.Errorf("command descriptions must differ by language")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/infrastructure/telegram/ -run 'TestHelpText_Localized|TestPublicCommands_Localized' 2>&1 | head`
Expected: FAIL — `helpText` / `PublicCommands` do not take a `lang` argument.

- [ ] **Step 4: Update `commands.go` to take `lang`**

Add the import for boti18n at the top of `commands.go`:

```go
import (
	"github.com/go-telegram/bot/models"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)
```

(Drop the now-unused `strings` import if `joinURL`/`webAppMarkup` no longer need it — `joinURL` uses `strings`, so KEEP `strings` and add boti18n alongside it.)

Replace `PublicCommands` and `helpText`:

```go
func PublicCommands(lang string) []models.BotCommand {
	return []models.BotCommand{
		{Command: "menu", Description: boti18n.T(lang, "cmd.menu")},
		{Command: "new", Description: boti18n.T(lang, "cmd.new")},
		{Command: "schedule", Description: boti18n.T(lang, "cmd.schedule")},
		{Command: "checker", Description: boti18n.T(lang, "cmd.checker")},
		{Command: "settings", Description: boti18n.T(lang, "cmd.settings")},
		{Command: "edit", Description: boti18n.T(lang, "cmd.edit")},
		{Command: "help", Description: boti18n.T(lang, "cmd.help")},
	}
}

func helpText(lang string) string {
	return boti18n.T(lang, "cmd.help.text")
}
```

- [ ] **Step 5: Add `resolveLang` helper + localize the `/help` reply in `multitenant.go`**

Confirm `multitenant.go` imports `context`, `github.com/go-telegram/bot/models`, and the boti18n package (add `"github.com/luckyrogue/lead-cat/internal/platform/boti18n"` to its import block). Add this method to the `Handler` (near the other helpers):

```go
// resolveLang returns the acting user's language: their stored bot_users.language
// if registered, else their Telegram client language_code, else ru.
func (h *Handler) resolveLang(ctx context.Context, from *models.User) string {
	var stored string
	if u, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
		stored = u.Language
	}
	return boti18n.Resolve(stored, from.LanguageCode)
}
```

Change the `/help` reply (currently `h.reply(ctx, b, update.Message, helpText())` at multitenant.go:165):

```go
			h.reply(ctx, b, update.Message, helpText(h.resolveLang(ctx, from)))
```

> Note: `postgres.BotUser` has a `Language` field (it is part of `botUserCols`); `models.User` has `LanguageCode`. `from` is `update.Message.From` already in scope in that handler.

- [ ] **Step 6: Register localized command menus in `main.go`**

In `apps/backend/cmd/server/main.go`, the current single registration is:
```go
		if _, cerr := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: telegram.PublicCommands()}); cerr != nil {
```
Replace it with a default (ru) registration plus per-language scopes:

```go
		if _, cerr := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: telegram.PublicCommands("ru")}); cerr != nil {
			// existing error handling unchanged
		}
		for _, lc := range []string{"en", "kk"} {
			if _, cerr := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{
				Commands:     telegram.PublicCommands(lc),
				LanguageCode: lc,
			}); cerr != nil {
				logger.Warn("set_commands_failed", zap.String("lang", lc), zap.Error(cerr))
			}
		}
```

> Keep the existing error handling on the default call exactly as it is; only the argument changed (`telegram.PublicCommands("ru")`). The loop is additive. Use the logger already in scope at that point in `main.go` (match its variable name; if it differs from `logger`, use the in-scope one). `SetMyCommandsParams.LanguageCode` is a field on the go-telegram/bot v1.21.0 params struct.

- [ ] **Step 7: Run the test + build + vet**

Run: `cd apps/backend && go test ./internal/infrastructure/telegram/ -run 'TestHelpText_Localized|TestPublicCommands_Localized' -v && go build ./... && go vet ./internal/infrastructure/telegram/ ./cmd/server/`
Expected: tests PASS; build + vet clean.

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/platform/boti18n/catalog_commands.go apps/backend/internal/infrastructure/telegram/commands.go apps/backend/internal/infrastructure/telegram/multitenant.go apps/backend/internal/infrastructure/telegram/commands_test.go apps/backend/cmd/server/main.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize /help + command menu; add dispatcher resolveLang

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 3: Localize `botreg` (unregistered-user path + FSM threading)

Threads `lang` through `botreg.Service` and pulls all replies from the catalog. The dispatcher resolves `lang` via the Task 2 helper and passes it into `Start`/`OnText`.

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_botreg.go`
- Modify: `apps/backend/internal/platform/botreg/service.go`
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (pass `lang` into `Start`/`OnText`)
- Test: `apps/backend/internal/platform/botreg/service_test.go`

**Interfaces:**
- Consumes (Task 1): `boti18n.T`. (Task 2): `h.resolveLang`.
- Produces: `Start(ctx, telegramID int64, lang string) string`, `OnText(ctx, telegramID int64, text, lang string) (string, bool)`.

- [ ] **Step 1: Create `catalog_botreg.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"botreg.welcome_back": {
			"ru": "С возвращением! 🐾 Открой приложение из меню.",
			"en": "Welcome back! 🐾 Open the app from the menu.",
			"kk": "Қайта келдің! 🐾 Мәзірден қосымшаны аш.",
		},
		"botreg.start": {
			"ru": "Привет! Давай зарегистрируемся.\nВведи ФИО (Фамилия Имя Отчество):",
			"en": "Hi! Let's get you registered.\nEnter your full name:",
			"kk": "Сәлем! Тіркелейік.\nТолық атыңды енгіз:",
		},
		"botreg.ask_name": {
			"ru": "Введи ФИО:", "en": "Enter your full name:", "kk": "Толық атыңды енгіз:",
		},
		"botreg.ask_email": {
			"ru": "Теперь корпоративную почту:", "en": "Now your work email:", "kk": "Енді жұмыс поштаңды енгіз:",
		},
		"botreg.bad_email": {
			"ru": "Не похоже на email. Попробуй ещё раз:", "en": "That doesn't look like an email. Try again:", "kk": "Бұл email емес сияқты. Қайта көр:",
		},
		"botreg.email_taken": {
			"ru": "Эта почта уже привязана к другому аккаунту.", "en": "This email is already linked to another account.", "kk": "Бұл пошта басқа аккаунтқа тіркелген.",
		},
		"botreg.failed": {
			"ru": "Не удалось завершить регистрацию, попробуй позже.", "en": "Couldn't finish registration, please try later.", "kk": "Тіркеуді аяқтау мүмкін болмады, кейінірек көр.",
		},
		"botreg.done": {
			"ru": "Готово, %[1]s! 🐾", "en": "Done, %[1]s! 🐾", "kk": "Дайын, %[1]s! 🐾",
		},
	})
}
```

- [ ] **Step 2: Write the failing test (extend `service_test.go`)**

Add (the file already has `newFakeUsers`, a sessions fake, and constructs a `Service`):

```go
func TestStart_Localized(t *testing.T) {
	svc := New(newFakeUsers(), newFakeSessions(), nil)
	en := svc.Start(context.Background(), 1, "en")
	ru := svc.Start(context.Background(), 2, "ru")
	if en == ru {
		t.Fatalf("Start must differ by language; both = %q", en)
	}
	if !strings.Contains(en, "register") {
		t.Errorf("en Start = %q", en)
	}
}

func TestOnText_NameStep_Localized(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	svc := New(users, sess, nil)
	_ = svc.Start(context.Background(), 1, "en") // sets awaiting_name
	reply, handled := svc.OnText(context.Background(), 1, "John Smith", "en")
	if !handled {
		t.Fatal("expected handled")
	}
	if !strings.Contains(reply, "email") {
		t.Errorf("en ask_email = %q", reply)
	}
}
```

> Use the existing sessions fake constructor in `service_test.go`. If the file's sessions fake has a different constructor name than `newFakeSessions`, use that name; add `"strings"` to the test imports if absent (`context` is already imported).

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/platform/botreg/ -run 'Localized' 2>&1 | head`
Expected: FAIL — `Start`/`OnText` don't accept a `lang` argument.

- [ ] **Step 4: Thread `lang` through `service.go`**

Add the boti18n import:

```go
import (
	"context"
	"net/mail"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)
```

Replace the three methods' signatures and bodies (every Russian literal → `boti18n.T`):

```go
func (s *Service) Start(ctx context.Context, telegramID int64, lang string) string {
	if _, err := s.users.GetBotUserByTelegramID(ctx, telegramID); err == nil {
		return boti18n.T(lang, "botreg.welcome_back")
	}
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepName})
	return boti18n.T(lang, "botreg.start")
}

func (s *Service) finishRegistration(ctx context.Context, telegramID int64, st State, lang string) (string, bool) {
	role := "user"
	if s.admins[telegramID] {
		role = "admin"
	}
	if _, err := s.users.CreateBotUser(ctx, telegramID, st.FullName, st.Email, role); err != nil {
		return boti18n.T(lang, "botreg.failed"), true
	}
	_ = s.sessions.Del(ctx, telegramID)
	return boti18n.T(lang, "botreg.done", st.FullName), true
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string) (string, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepName:
		if text == "" {
			return boti18n.T(lang, "botreg.ask_name"), true
		}
		st.FullName = text
		st.Step = stepEmail
		_ = s.sessions.Set(ctx, telegramID, *st)
		return boti18n.T(lang, "botreg.ask_email"), true

	case stepEmail:
		addr, perr := mail.ParseAddress(text)
		if perr != nil {
			return boti18n.T(lang, "botreg.bad_email"), true
		}
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, gerr := s.users.GetBotUserByEmail(ctx, email); gerr == nil {
			return boti18n.T(lang, "botreg.email_taken"), true
		}
		st.Email = email
		return s.finishRegistration(ctx, telegramID, *st, lang)
	}
	return "", false
}
```

- [ ] **Step 5: Update the dispatcher calls in `multitenant.go`**

The two `botreg` call sites must pass the resolved language. Change:
```go
			if reply, handled := h.registrar.OnText(ctx, from.ID, text); handled {
```
to:
```go
			if reply, handled := h.registrar.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {
```
and:
```go
			h.reply(ctx, b, update.Message, h.registrar.Start(ctx, from.ID))
```
to:
```go
			h.reply(ctx, b, update.Message, h.registrar.Start(ctx, from.ID, h.resolveLang(ctx, from)))
```

> `h.resolveLang` is the helper added in Task 2; `from` is in scope at both call sites.

- [ ] **Step 6: Run the test + build + vet**

Run: `cd apps/backend && go test ./internal/platform/botreg/ -v && go build ./... && go vet ./internal/platform/botreg/ ./internal/infrastructure/telegram/`
Expected: botreg tests PASS (existing + the 2 new localized ones); build + vet clean.

- [ ] **Step 7: Run the boti18n coverage test (confirms all new keys have ru/en/kk)**

Run: `cd apps/backend && go test ./internal/platform/boti18n/ -run TestCatalog_AllKeysHaveAllLangs -v`
Expected: PASS (cmd.* and botreg.* keys all have three languages).

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/platform/boti18n/catalog_botreg.go apps/backend/internal/platform/botreg/service.go apps/backend/internal/infrastructure/telegram/multitenant.go apps/backend/internal/platform/botreg/service_test.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize botreg registration FSM (ru/en/kk)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- `boti18n` package: `T`/`Normalize`/`Resolve` + keyed catalog + per-domain files via `register` → Task 1. ✓
- Explicit-index `%[1]s` params → `botreg.done` + `test.greet`, exercised in tests. ✓
- Missing-key → key; missing-lang → ru fallback → Task 1 `T` + tests. ✓
- Catalog coverage test (all keys have ru/en/kk) → Task 1, re-run in Task 3 Step 7. ✓
- Pilot 1 commands/help, registered-user stored language → Task 2 (`helpText(lang)`, `PublicCommands(lang)`, `resolveLang` loads `bot_users.language`). ✓
- Pilot 2 botreg, unregistered Telegram `language_code` + FSM threading → Task 3 (`resolveLang` falls back to `from.LanguageCode`; `lang` threaded through `Start`/`OnText`/`finishRegistration`). ✓
- ru values verbatim from current code → Task 2/3 catalogs copy existing strings. ✓
- Tests: boti18n units, help differs by lang, botreg replies differ by lang → Tasks 1–3. ✓
- Out of scope (B/C/D, settings UI, plurals) → not touched. ✓

**Placeholder scan:** No TBD/TODO. Every code step shows complete code. The `> Note` blocks flag real integration facts (BotUser.Language field, the sessions-fake constructor name, keeping the existing error handling) — not deferred work.

**Type consistency:** `Start(ctx, id, lang)`, `OnText(ctx, id, text, lang)`, `finishRegistration(ctx, id, st, lang)` signatures are consistent between Task 3's service code, its tests, and the Task 3 Step 5 dispatcher calls. `resolveLang(ctx, from *models.User) string` is defined in Task 2 and consumed in Tasks 2 & 3. `PublicCommands(lang)`/`helpText(lang)` consistent between commands.go, commands_test.go, and main.go. Catalog keys referenced in code (`cmd.help.text`, `botreg.*`) all exist in the catalog files.

**Known build-verified-only items (repo convention for thin layers):** the `main.go` command-menu registration and the `multitenant.go` dispatcher wiring are verified by `go build`/`go vet`; their behavior (lang resolution, localized text) is covered by the `boti18n`, `commands`, and `botreg` unit tests.
