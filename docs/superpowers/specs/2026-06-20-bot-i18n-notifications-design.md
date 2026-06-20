# Bot i18n — Notifications (design)

## Context: bot localization, sub-project B of A–D

Sub-project A landed the `boti18n` catalog (`T`/`Normalize`/`Resolve`) and migrated
`commands`/help + `botreg`. B localizes the **notification** surfaces and — because
the per-recipient restructure makes it nearly free — also fixes their timezone.
(Remaining: C = stateful FSMs, D = NL agent. Each extends the same catalog.)

## Problem

Two notification surfaces are hardcoded Russian, and the meeting-notifier has a
second defect:

1. **`meeting_notifier`** builds one message with a Russian header
   (`buildMessage`/`buildUpdatedMessage`/`buildRemovedMessage`/`buildCancelledMessage`
   + the `➕ Вас добавили…` participant header) and — for created/updated/cancelled —
   builds that text **once before the recipient loop**, sending the identical text to
   every recipient. It also formats times in the **organization** timezone
   (`w.TZ`), so a recipient in a different timezone sees the wrong time.
2. **`reminder_scheduler`** Telegram reminders (`message` + `offsetLabel` in
   `reminder.go`) are hardcoded Russian.

`meetingrecipients.Recipient` already carries `.Language` and `.Timezone`; the
participant path has the recipient's `postgres.BotUser` (with `.Language`/`.Timezone`).

## Goal

Each recipient receives the notification in **their** language and **their**
timezone (falling back to the organization timezone, then `Asia/Almaty`/UTC as
today). Reminders are localized (they contain no date/time, so no timezone change).
All new strings go through `boti18n` (ru verbatim from current code, + en/kk), and a
catalog-coverage test keeps every key complete.

## Design

### `meeting_notifier` — per-recipient language + timezone

**`message.go`:** every builder takes a `lang string` and resolves its header from
the catalog; the date/time format, `«»`, `🗓`/`🔗` emoji, and `tzLabel` are
language-neutral and stay as-is.

```go
func buildEventMessage(headerKey, lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string
func buildMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string          // header "notif.created"
func buildUpdatedMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string    // "notif.updated"
func buildRemovedMessage(lang, name string, startsAt time.Time, loc *time.Location) string                      // "notif.removed"
func buildCancelledMessage(lang, name string, startsAt time.Time, loc *time.Location) string                    // "notif.cancelled"
// the participant-added header uses "notif.added" (currently the inline "➕ Вас добавили на встречу")
```

**`notifier.go`:**
- A small helper resolves a recipient's location:
  `loc := loadRecipientLoc(recipientTZ, orgTZ)` — `time.LoadLocation(cmp.Or(recipientTZ, orgTZ, "Asia/Almaty"))`, warn + `time.UTC` on error (same fallback behavior as today, just keyed on the recipient first).
- `HandleCreated` / `HandleUpdated` / `HandleCancelled`: **move the `buildX(...)` call
  inside the recipient loop**, building with `r.Language` and `loadRecipientLoc(r.Timezone, w.TZ)`. The org `w.TZ` is now only the fallback.
- `notifyParticipant` (added/removed): already single-recipient — build with
  `u.Language` and `loadRecipientLoc(u.Timezone, w.TZ)`.
- The `TryClaimReminder` gating, send, and error logging are unchanged; only the
  text construction moves into the loop.

### `reminder_scheduler` — localized Telegram reminders

**`reminder.go`:**
- `offsetLabel(min int, lang string) string` — fixed offsets (10/15/30/60/120/1440)
  map to catalog keys (`reminder.offset.10m`, `…15m`, `…30m`, `…1h`, `…2h`, `…1d`);
  the default branch returns `boti18n.T(lang, "reminder.offset.n_min", min)` (a
  `"%[1]d мин"` / `"%[1]d min"` style string — plain, no plural rules per the A-spec
  YAGNI decision).
- `message(name, meetLink string, offset int, lang string) string` — header from
  `reminder.telegram` (`"⏰ Напоминание: встреча через %[1]s!"` → arg = `offsetLabel(offset, lang)`); `«%s»` name and `🔗` link unchanged.

**`scheduler.go`:** the Telegram send site (currently `Text: message(m.Name, m.MeetLink, off)`) passes `t.Language` — the reminder target already carries it.

### Catalog

- `boti18n/catalog_notifier.go` — `notif.created`/`updated`/`removed`/`cancelled`/`added`.
- `boti18n/catalog_reminder.go` — `reminder.telegram`, `reminder.offset.{10m,15m,30m,1h,2h,1d,n_min}`.
- `ru` values copied verbatim from the current code (e.g. `"📅 Новая встреча"`,
  `"⏰ Напоминание: встреча через %[1]s!"`, `"1 час"`); en/kk are new. The existing
  coverage test (`TestCatalog_AllKeysHaveAllLangs`) now also covers these keys.

### Out of scope (B)

- The stateful FSMs (`checker`/`scheduleview`/`meetingedit`/`botsettings`) → C.
- The NL agent → D.
- The **email** reminder path — already localized (uses `t.Language`); untouched.
- Plural/gender rules — `reminder.offset.n_min` accepts the plain-form imperfection
  for arbitrary offsets (the configured set uses fixed phrases anyway).

## Error handling

- A recipient with an unloadable/empty timezone falls back to org TZ, then UTC
  (`time.LoadLocation` error → warn + UTC), exactly as today — no new failure mode.
- Missing catalog key → key returned (loud in dev); missing language → ru. Send/claim
  errors keep their current warn-and-continue behavior.

## Testing

- **`message_test.go`** (meeting_notifier): each builder localizes its header
  (ru/en/kk differ, non-empty) and still contains the name, date, and (where present)
  the meet link; an `en` header is the English string. Pure — no DB.
- **Notifier per-recipient test:** with a fake store/sender and two resolved
  recipients of different languages **and timezones**, assert each receives text in
  its own language and with its own local time (two distinct sent texts). Reuse/extend
  the existing meeting_notifier test fakes.
- **`reminder.go` tests:** `offsetLabel` returns the right localized phrase per
  offset and language (and the `n_min` default for an off-list value); `message`
  localizes the header and includes the meet link.
- The `boti18n` coverage test confirms all new `notif.*`/`reminder.*` keys have
  ru/en/kk.
