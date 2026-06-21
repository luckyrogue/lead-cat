# Bot i18n — meetingedit C2 (design)

## Context: bot localization, sub-project C2 of A–D

Sub-projects A (catalog + commands/botreg), B (notifications), and C1 (checker /
scheduleview / botsettings) are done. C2 localizes the largest remaining bot surface:
the `meetingedit` FSM — ~121 Russian string literals across ~30 functions in a
716-line `service.go`. After C2, only **D** (the NL scheduling agent) remains.

## Problem

`meetingedit` (the `/edit` flow: pick a meeting, edit fields, manage participants,
change recurrence, apply/force-apply on conflict, delete, choose single-vs-series
scope) is hardcoded Russian in both reply text and inline-keyboard button labels. Its
`Service` methods take `(ctx, telegramID, …)` and never receive the user's language.
This is a registered-user flow (`/edit`), so the language is the stored
`bot_users.language`, resolvable by the dispatcher's existing `resolveLang` helper.

## Goal

Localize every user-facing string (reply text **and** button labels, including the
multi-field `menuText` summary and the conflict warning) in `meetingedit` to ru/en/kk
via `boti18n`, threading `lang` through the FSM exactly as A/B/C1 did. ru values copied
verbatim from current code; en/kk new.

## Design

### Language threading (param, the A/B/C1 pattern)

`lang string` is a trailing parameter threaded through every `meetingedit` function
that produces user-facing text — the three entry methods and all helpers:

- Entry: `Start`, `OnCallback`, `OnText` (each gains `lang`).
- Action helpers: `pick`, `field`, `setRec`, `apply`, `applyForce`, `doApply`,
  `parts`, `padd`, `searchParticipant`, `paddPick`, `prem`, `premConfirm`,
  `confirmDelete`, `doDelete`, `deleteErrReply`, `setScope`, `backToMenu`.
- Pure renderers: `menuReply`, `menuText`, `recReply`, `recLabel`, `scopeReply`,
  `menuKeyboard`, `partsReply`, `conflictKeyboard`, `formatConflictWarning`.
  (`summary` is `«name»` + meet link only — language-neutral, not threaded.)

The FSM performs no language lookup itself. The dispatcher (`multitenant.go`) resolves
the language once per update with `h.resolveLang(ctx, from)` (message context) or
`h.resolveLang(ctx, &cq.From)` (callback context — `cq.From` is a value, so its
address is passed, consistent with C1) and passes it into the three `editor` call
sites (lines 85 OnText, 130 Start, 198 OnCallback).

Because the call graph is deeply connected, the threading is split into reviewable
clusters in the plan; each cluster's commit compiles because an as-yet-unthreaded
branch simply keeps its old (Russian) signature until its own task.

### Localized strings

Every Russian literal — reply `Text`, `Button.Text`, the `menuText` field summary
(`Дата/время`, `Отдел`, `Тип`, `Ведущий`, `Описание`, `Частота`, the
`Редактирование встречи (★ — изменено):` / `Редактирование всей серии с %s …`
headers), `formatConflictWarning`, and the `recLabel` `"once"` branch — is replaced by
`boti18n.T(lang, key, args...)` with explicit-index verbs (`%[1]s`/`%[1]d`) where
parameterized. The `•` bullets, `★` change-marker, `«»`, `🔗`, `summary`'s output, and
date/time number formatting are language-neutral and stay as-is.

### Catalog

`boti18n/catalog_meetingedit.go` — `medit.*` keys, ru verbatim, en/kk new. The
existing `TestCatalog_AllKeysHaveAllLangs` enforces all three languages per key.

### Out of scope (C2), explicitly

- **Domain recurrence labels** — `recLabel` delegates non-`once` values to
  `meeting.Recurrence(v).Label()` in `domain/meeting`, which returns Russian labels
  used across the whole app (web, mini-app, bot). Localizing the domain layer is a
  separate, broader-radius concern. C2 localizes only the `meetingedit`-local `"once"`
  branch (`medit.rec.once`); other recurrence values keep the domain label for now.
- **`loadLoc(tz)`** — uses the meeting's stored timezone (defaults to `Asia/Almaty`
  only when empty); it is not the deferred hardcoded-parse timezone item and is not
  touched.
- The NL scheduling agent → D.
- Plural/gender rules — plain catalog strings (consistent with A/B/C1).

## Error handling

- Missing key → key returned (loud in dev); missing language → ru. FSM control flow is
  unchanged — only the string source moves to the catalog.

## Testing

- Reuse the existing `service_test.go` fakes. Add localized assertions covering, at
  minimum: `Start`/menu (text differs ru vs en + a known English phrase), the
  single-vs-series `menuText` header, one participants-flow reply (e.g. the
  participants menu or an add/remove prompt), and one apply/delete-flow reply (e.g. the
  conflict warning or delete confirmation) — each asserting ru≠en and an expected
  English substring, plus at least one localized **button label**.
- Existing tests that call now-localized methods are updated to pass `"ru"` and keep
  their assertions (ru text unchanged).
- The `boti18n` coverage test confirms all new `medit.*` keys have ru/en/kk.
