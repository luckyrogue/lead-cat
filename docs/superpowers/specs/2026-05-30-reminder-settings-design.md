# Design — Reminder settings (§5b-1)

**Date:** 2026-05-30
**Status:** Approved (brainstorm)
**Part of:** meetings notifications ([NEW-FEATURES.md](../../NEW-FEATURES.md) §5.2). First half of §5b; the reminder **engine** (§5b-2) is a separate increment that consumes these settings. Builds on bot registration (`bot_users`).

## Goal

Let a registered bot user configure their reminder intervals via `/settings` with an inline keyboard of toggle buttons (10м / 15м / 30м / 1ч / 2ч / 1день; multiple allowed; can disable all). Persist per-user; no reminder engine yet.

## Decisions (from brainstorm)

- **UI: inline keyboard** (`/settings` + callback-query toggles). Requires adding callback-query handling to the bot (currently messages-only).
- **Storage: TEXT CSV** on `bot_users` (`reminder_minutes`, e.g. `"15,60"`; empty = disabled). Chosen over `INT[]` to avoid pgx array-scan friction.
- **Default `'15'`** — one 15-minute reminder out of the box (configurable / disable-able).
- **Testability:** a `botsettings` service with injected store + **pure** `parse`/`toggle`/`render` functions (no bot/DB).

## Data (new goose migration)

```sql
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS reminder_minutes TEXT NOT NULL DEFAULT '15';
```
- Model: `BotUser` gains `ReminderMinutes string` (raw CSV). `GetBotUserByTelegramID` adds the column to its SELECT/Scan.
- Store: `SetReminderMinutes(ctx, telegramID int64, csv string) error`.

## Settings service (`internal/platform/botsettings`)

Intervals (minutes ↔ label): `10→10м, 15→15м, 30→30м, 60→1ч, 120→2ч, 1440→1день`. Callback data: `rem:<minutes>` (e.g. `rem:15`).

Pure helpers (unit-tested, no IO):
- `parse(csv string) []int` — split, trim, drop empties/non-numeric, dedupe, sort ascending.
- `format(mins []int) string` — back to CSV (canonical: sorted, comma-joined).
- `toggle(cur []int, v int) []int` — add v if absent, remove if present.
- `render(mins []int) (text string, keyboard [][]Button)` — one button per interval, prefixed with `✓ ` when enabled; `Button{Text, Data}`.

Service (injected `store` interface with `GetBotUserByTelegramID` + `SetReminderMinutes`):
- `Settings(ctx, telegramID) (text, keyboard, error)` — load `reminder_minutes`, `render(parse(...))`.
- `Toggle(ctx, telegramID, v int) (text, keyboard, error)` — load → `toggle` → `SetReminderMinutes(format(...))` → `render`.

`Button` is a transport-agnostic struct (`Text`, `Data`); the telegram layer maps it to `models.InlineKeyboardButton`.

## Telegram wiring (`MultiHandler.Handle`)

- At the top, branch on `update.CallbackQuery != nil`: if `Data` has prefix `rem:`, parse the int, call `botsettings.Toggle(ctx, from.ID, v)`, then `b.EditMessageText` (new text + inline keyboard on the originating message) and `b.AnswerCallbackQuery`. Ignore unknown callback data.
- Add a `/settings` command case (private chats, registered users only): `botsettings.Settings(...)` → `b.SendMessage` with `ReplyMarkup: &models.InlineKeyboardMarkup{...}` built from the returned `[][]Button`.
- A small helper converts `[][]botsettings.Button` → `models.InlineKeyboardMarkup`.

The settings service + store are wired into `MultiHandler` (it already gets `store`; add the `botsettings.Service`).

## Message

Text: `⏰ Напоминания о встречах. Выбери, за сколько предупреждать (можно несколько):`. When the set is empty, append a line `Сейчас напоминания выключены.` Keyboard: the interval buttons (rows of 3), each `✓ 15м` when on, `15м` when off.

## Testing

- **Unit (pure):** `parse` (dedupe/sort/garbage), `format`, `toggle` (add/remove), `render` (✓ marks, empty state). No IO.
- **Service:** `Toggle`/`Settings` with a fake store — toggling persists canonical CSV; render reflects it.
- **Manual/integration (out of CI):** real bot polling — `/settings` shows the keyboard; tapping toggles and the message updates. Not in CI (no Telegram).

## Out of scope (later)

- The reminder **engine** (scheduler + dedup + DM sending) — §5b-2.
- Meeting-created notification (§5a).
- Per-meeting reminder overrides (settings are global per user, per ТЗ §5.2).
- Localization of labels; an explicit "disable all" button (toggling everything off = disabled).
