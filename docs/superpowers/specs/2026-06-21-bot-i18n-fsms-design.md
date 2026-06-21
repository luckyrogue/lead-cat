# Bot i18n — Stateful FSMs C1 (design)

## Context: bot localization, sub-project C1 of A–D

Sub-projects A (catalog + commands/botreg) and B (notifications) are done. C localizes
the stateful FSM handlers. Because `meetingedit` alone is 121 strings in a 716-line
file, C is split:

- **C1 (this spec):** `checker`, `scheduleview`, `botsettings` (~63 strings).
- **C2 (next):** `meetingedit` (its own spec/plan).

Then **D:** the NL scheduling agent. Each extends the same `boti18n` catalog.

## Problem

`checker`, `scheduleview`, and `botsettings` are hardcoded Russian — both reply text
and inline-keyboard button labels. Their `Service` methods take `(ctx, telegramID, …)`
and never receive the user's language. These are registered-user flows (`/checker`,
`/schedule`, `/settings`), so the language is the stored `bot_users.language`,
resolvable by the dispatcher's existing `resolveLang` helper (from A).

## Goal

Localize all user-facing strings (reply text **and** button labels) in these three
FSMs to ru/en/kk via `boti18n`, threading `lang` through each FSM exactly as
sub-project A did for `botreg`. ru values copied verbatim from current code; en/kk new.

## Design

### Language threading (the A/botreg pattern)

- Each FSM entry method gains a trailing `lang string` parameter:
  - `checker`: `Start(ctx, telegramID, lang)`, `OnText(ctx, telegramID, text, lang)`,
    `OnCallback(ctx, telegramID, data, lang)`.
  - `scheduleview`: same three.
  - `botsettings`: `Toggle(ctx, telegramID, minutes int, lang string)`.
- `lang` is threaded down into each FSM's internal helpers
  (`search`/`add`/`done`/`setRange`/`duration`/`formatSlots` in checker; the
  analogous helpers + `periodLabel`/`scheduleText` in scheduleview; `render` in
  botsettings). The FSMs perform no store/Telegram language lookup themselves.
- The dispatcher (`multitenant.go`) resolves the language once per update with the
  existing `h.resolveLang(ctx, from)` (or `h.resolveLang(ctx, cq.From)` for callback
  queries) and passes it into all nine call sites (lines 85/89/93/130/138/146 for
  text/start, 186 for `settings.Toggle`, 198/203/208 for callbacks).

### Localized strings

Every Russian literal — reply `Text`, inline-keyboard `Button.Text`, and the strings
produced by format helpers (`formatSlots`, `scheduleText`, `periodLabel`, the
botsettings `render` text + interval labels) — is replaced by
`boti18n.T(lang, key, args...)`. Parameterized strings use explicit-index verbs
(`%[1]s`/`%[1]d`), e.g. checker's `"Добавлен: %[1]s\nУчастников: %[2]d. …"`.

Button labels are localized too (they are user-facing). Note: checker's
duration-keyboard labels (15/30/45/60/90/120 min) are a **distinct** set from the
reminder offsets and get their own `checker.dur.*` keys — not shared with
`reminder.offset.*` (different values, avoids coupling).

### Catalog

- `boti18n/catalog_checker.go` — `checker.*`
- `boti18n/catalog_scheduleview.go` — `sched.*`
- `boti18n/catalog_botsettings.go` — `botset.*`

ru values verbatim from the current code; en/kk new. The existing
`TestCatalog_AllKeysHaveAllLangs` enforces all three languages on every key.

### Out of scope (C1)

- The hardcoded `Asia/Almaty` date-input parsing in `checker/parse.go` and
  `scheduleview/service.go` — a distinct timezone-correctness concern (priority #4),
  handled separately. C1 does not change parsing behavior.
- `meetingedit` → C2; the NL agent → D.
- Plural/gender rules — plain catalog strings (consistent with A/B).

## Error handling

- Missing key → key returned (loud in dev); missing language → ru. No new failure
  modes; FSM control flow is unchanged — only the string source moves to the catalog.

## Testing

- Per FSM, a test asserts a localizable entry produces different output by language
  for **both** a reply text and at least one button label (ru vs en), and that the ru
  output still matches the original wording. Reuse the existing `*_test.go` fakes in
  each package (`checker`, `scheduleview`, `botsettings` already have tests).
- `botsettings` `render`/`Toggle` localized (text + interval labels + the
  "reminders off" line) asserted across languages.
- The `boti18n` coverage test confirms all new `checker.*`/`sched.*`/`botset.*` keys
  have ru/en/kk.
