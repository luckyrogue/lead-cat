# Bot i18n — Foundation + Pilot (design)

## Context: bot localization, sub-project A of A–D

The Telegram bot layer is hardcoded Russian across ~266 string literals in 8+
packages, while emails and the web/mini-app UIs are trilingual (ru/en/kk). Stored
`bot_users.language` is fetched but consumed only by the email path. Localizing the
whole bot is too large for one spec, so it is decomposed:

- **A (this spec) — foundation + pilot:** a translation-catalog mechanism, a
  language-resolution helper, and migration of two pilot surfaces (`commands`/help +
  `botreg`) that prove both resolution paths.
- **B — notifications:** `meeting_notifier` (per-recipient) + Telegram reminders.
- **C — stateful FSMs:** `checker`, `scheduleview`, `meetingedit`, `botsettings`.
- **D — NL scheduling agent:** prompt + replies in the user's language.

B/C/D each get their own spec → plan; they extend the catalog this sub-project
establishes. This spec covers **A only**.

## Problem

There is no i18n mechanism for bot text (unlike emails) and no single way for a
handler to resolve the acting user's language. Registered users have
`bot_users.language`; unregistered users (mid-registration in `botreg`) are not yet
in the DB, so their only language signal is the Telegram `Message.From.LanguageCode`
(available via `go-telegram/bot` v1.21.0).

## Goal

1. A small dependency-free `boti18n` package: a keyed translation catalog plus
   `T(lang, key, args...)`, `Normalize(lang)`, and `Resolve(stored, telegramCode)`.
2. Migrate two pilot surfaces to it, proving both resolution paths:
   - `commands`/help — **registered** user (stored language).
   - `botreg` — **unregistered** user (Telegram `language_code`), and threading
     `lang` through a stateful FSM (the pattern sub-project C will reuse).

## Design

### Package `internal/platform/boti18n`

```go
// Supported languages; default ru.
func Normalize(lang string) string // "ru" | "en" | "kk"; anything else → "ru"

// T looks up the format string for key in the normalized language and applies args
// via fmt.Sprintf. Missing key → returns key verbatim (visible in dev). Missing
// language for a present key → falls back to ru.
func T(lang, key string, args ...any) string

// Resolve picks the effective language: a non-empty stored preference wins
// (normalized); otherwise the Telegram language_code (normalized); otherwise ru.
func Resolve(stored, telegramCode string) string
```

- **Catalog storage:** `map[string]map[string]string` — `catalog[key][lang] = format`.
  Keys are namespaced by domain (`cmd.*`, `botreg.*`). Catalog entries live in
  per-domain files (`catalog_commands.go`, `catalog_botreg.go`); later sub-projects
  add `catalog_notifier.go`, `catalog_checker.go`, etc. without touching `t.go`.
- **Parameters:** format strings use explicit-index verbs where reordering across
  languages is needed (e.g. `"%[1]s забронировал %[2]s"`), so translators are not
  locked to Go's positional order. `T` passes `args` straight to `fmt.Sprintf`.
- **Fallbacks:** `T` resolves `Normalize(lang)`; if `catalog[key]` exists but lacks
  that language, use the `ru` entry; if `catalog[key]` is absent entirely, return
  `key`. `Resolve` and `Normalize` never error.
- **Why not reuse `emailtemplates.NormalizeLang`:** keeping `boti18n` self-contained
  avoids a platform→platform cross-import and lets the bot catalog evolve
  independently. The one-line normalize duplication is acceptable (KISS over a shared
  abstraction for two trivial functions).

### Pilot 1 — `commands`/help (registered user, stored language)

- The command dispatcher (in `internal/infrastructure/telegram`) resolves the acting
  user's language once per update: load `bot_users.language` by the message's
  Telegram ID, then `lang := boti18n.Resolve(stored, msg.From.LanguageCode)`.
- `helpText()` and any other hardcoded-Russian command responses in `commands.go`
  take a `lang string` parameter and build their text from `boti18n.T(lang, "cmd.…")`.
- All `cmd.*` keys get ru/en/kk entries in `catalog_commands.go`.

### Pilot 2 — `botreg` (unregistered user, Telegram language_code; FSM threading)

- `botreg.Service` methods that currently return Russian strings
  (`Start`, the registration-step handlers, `finishRegistration`, error replies) take
  a `lang string` parameter and return text via `boti18n.T(lang, "botreg.…")`.
- The dispatcher computes `lang` before calling `botreg`: for a returning user, from
  their stored language; for a new user, from `msg.From.LanguageCode` — both via
  `boti18n.Resolve`. `botreg` itself stays free of store/Telegram lookups for
  language (it just receives `lang`), keeping it unit-testable.
- All `botreg.*` keys get ru/en/kk entries in `catalog_botreg.go`. The existing
  Russian strings are the `ru` values verbatim; en/kk are new translations.

### Out of scope (A)

- Localizing notifications, the other FSMs, `meetingedit`, `botsettings`, or the NL
  agent — those are B/C/D and extend this catalog later.
- Changing where/how users set their language (the settings UI already persists
  `bot_users.language`).
- Plural rules / gendered forms — the catalog stores plain format strings; if a
  later surface needs plurals, that sub-project adds a helper then (YAGNI now).

## Error handling

- A missing translation key returns the key string itself (loud in dev, never a
  panic). A missing language for a present key silently falls back to ru.
- Language resolution always yields a valid supported language (default ru); no
  handler needs to error on a bad/empty language.

## Testing

- **`boti18n` unit tests** (`t_test.go`): `Normalize` (ru/en/kk + garbage→ru);
  `Resolve` (stored wins; empty stored → telegramCode; both empty → ru; garbage →
  ru); `T` (returns the right language; missing-language-for-key → ru; missing-key →
  key returned; `args` applied via Sprintf incl. an explicit-index `%[1]s` case).
- **Catalog coverage test:** every key present in the catalog has all three of
  ru/en/kk (so a half-translated key fails CI, not production).
- **Pilot surfaces:** assert `helpText("en")` ≠ `helpText("ru")` and both non-empty;
  a `botreg.Service` test that `Start(..., "en")` returns the English string and
  `Start(..., "ru")` the Russian one (using the existing botreg test fakes). FSM-step
  replies similarly asserted for at least one non-ru language.
