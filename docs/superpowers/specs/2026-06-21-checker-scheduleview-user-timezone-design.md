# checker / scheduleview — user timezone (design)

## Context

A deferred correctness item from the bot-localization work: the `checker` (`/checker`)
and `scheduleview` (`/schedule`) FSMs hardcode `Asia/Almaty` (via a local `almaty()`
helper) for both **parsing** a user's date input and **displaying** times. A user in
another timezone who types `2026-06-15`, a range, or picks "today/tomorrow/upcoming"
gets a window computed in Almaty, and sees slot/schedule times in Almaty — wrong for
their locale. (Notification display tz was already fixed per-recipient in sub-project
B; meeting times in the mini-app are server-formatted in the user's tz; this item
closes the remaining checker/scheduleview gap.)

The user's timezone is the stored `bot_users.timezone` — the same record the
dispatcher's `resolveLang` already reads for language.

## Problem

In `checker` and `scheduleview`, `almaty()` is passed as the `*time.Location` to:
- **Parsing:** `parseRange`/`parseDate` (interpret `YYYY-MM-DD` input) and `dayWindow`
  (compute today/tomorrow/upcoming boundaries relative to `time.Now()`).
- **Display:** `formatSlots` (checker free-slot times) and `scheduleText`
  (scheduleview meeting times).

The parse/display helpers already take a `*time.Location` parameter — only the
`almaty()` call sites are hardcoded. Nothing reads the user's stored timezone here.

## Goal

In both FSMs, interpret date input and render times in the **user's stored timezone**
(`bot_users.timezone`), falling back to `Asia/Almaty` (then UTC) when unset/invalid —
the same fallback `almaty()` provides today. The dispatcher resolves the location once
and threads it in, parallel to `lang`.

## Design

### 1. Dispatcher — resolve language + location together

Add `func (h *MultiHandler) resolveLangLoc(ctx, from *models.User) (string, *time.Location)`:
one `GetBotUserByTelegramID` lookup returning both the resolved language (as
`resolveLang` does today) and a `*time.Location` from `bot_users.timezone`
(`time.LoadLocation`, fallback `Asia/Almaty`, then `time.UTC` on load error). The
existing `resolveLang` stays for the other FSMs; `resolveLangLoc` is used only at the
checker/scheduleview `OnText`/`OnCallback` dispatch sites (which need both). The
`Start` sites of those FSMs only emit a prompt (no date parse/display), so they keep
`resolveLang` (lang only).

### 2. Thread `loc` into the date paths

Add a trailing `loc *time.Location` parameter to the entry methods and helpers that
parse or display dates, alongside the `lang` added in sub-project C1:
- **checker:** `OnText(...,lang,loc)` → `setRange(...,lang,loc)` (uses `loc` for
  `parseRange`); `OnCallback(...,lang,loc)` → `duration(...,lang,loc)` (uses `loc` for
  `time.ParseInLocation` of the stored range and for `formatSlots`).
- **scheduleview:** `OnText(...,lang,loc)` (uses `loc` for `parseDate`/`parseRange`,
  and passes it to `list`); `OnCallback(...,lang,loc)` → `period(...,lang,loc)` (uses
  `loc` for `dayWindow`) and `list(...,loc)`; `list(...,lang,loc)` passes `loc` to
  `scheduleText`.
- Helpers (`parseRange`, `parseDate`, `dayWindow`, `formatSlots`, `scheduleText`) are
  **unchanged** — they already take `*time.Location`; only the `almaty()` arguments at
  the call sites become the threaded `loc`.

`time.Now()` stays as the absolute "now"; the `loc` it is combined with in `dayWindow`
determines the day boundaries — so "today/tomorrow" become the user's.

### 3. `almaty()` retained as the fallback

`almaty()` stays as the default the dispatcher's resolver falls back to (so the
behavior for a user with no stored timezone is unchanged). It is no longer called
directly from the parse/display flow.

### Out of scope

- **Weekday name localization** (`checker`'s `ruWeekday`/`dayLabel`): the weekday
  abbreviations stay Russian — that is the separate, still-deferred bot-i18n tail, not
  a timezone concern. This change makes the weekday *correct* for the user's tz; the
  *names* remain Russian.
- `meetingedit` (its `loadLoc` already uses the meeting's stored tz), the NL agent, and
  all other surfaces.
- Changing where users set their timezone (already persisted via settings).

## Error handling

- Unset/invalid `bot_users.timezone` → `Asia/Almaty` → UTC (existing `almaty()`
  fallback chain), so no new failure mode. Parse errors are unchanged (still surfaced
  via the C1-localized `checker.bad_range` / `sched.bad_date` / `sched.bad_range`
  keys).

## Testing

- **checker:** a `setRange`/`duration`-path test asserting a date range parsed under a
  non-Almaty `loc` yields the expected UTC instants (e.g. `2026-06-15` in
  `Europe/London` ≠ the Almaty interpretation), and `formatSlots` renders in the
  passed `loc`. Reuse the existing checker test fakes; pass an explicit `loc`.
- **scheduleview:** `dayWindow("today", loc)` boundaries differ by `loc` (a London vs
  Almaty "today" start differ); `parseDate`/`parseRange` interpret under the passed
  `loc`; `scheduleText` renders in `loc`. The parse/`dayWindow` helpers are pure and
  already loc-parameterized, so these are direct unit tests.
- **Dispatcher:** `resolveLangLoc` returns the stored-tz location when set and the
  Almaty fallback when unset — a small unit test against the store seam (or a focused
  test of the fallback chain).
- Backend `go build ./... && go vet && go test ./internal/platform/checker/ ./internal/platform/scheduleview/` green.
