# Design — Meetings: real Google Calendar adapter (increment 2)

**Date:** 2026-05-30
**Status:** Approved (brainstorm)
**Builds on:** [2026-05-30-meetings-backend-design.md](2026-05-30-meetings-backend-design.md) (increment 1 — CRUD + stub). Full product spec (ТЗ): [../../NEW-FEATURES.md](../../NEW-FEATURES.md).

## Goal

Replace the stubbed `CalendarService` with a real Google Calendar adapter that creates events with Google Meet links, using a **per-workspace** Google service account (domain-wide delegation). Keep the stub available for local/CI via a flag.

## Context & decisions

- **Google identity:** a service account alone cannot mint Meet links. The adapter uses **domain-wide delegation** — the SA impersonates a corporate Workspace user (`subject`) and creates the event with `conferenceData.createRequest`, which generates the Meet link.
- **Credential scope: per-workspace.** Each workspace stores its own encrypted SA JSON + subject + calendar id (mirrors the existing encrypted per-workspace VCS-token pattern).
- **No creds → 400.** Creating a meeting in a workspace without Google configured returns `400 "google not configured"`. For local/CI, a global flag `CALENDAR_STUB=true` forces the stub provider for all workspaces.
- **Port stays clean (approach A):** a `CalendarProvider.For(ctx, workspaceID) (CalendarService, error)` resolves the right `CalendarService` per workspace. The `CalendarService` port (`CreateEvent`/`DeleteEvent`) is unchanged.

## Credential storage (new goose migration)

Add to `workspaces`:
- `google_sa_json_enc BYTEA` — service-account JSON, encrypted with the existing `Cipher` (MASTER_ENCRYPTION_KEY)
- `google_subject TEXT NOT NULL DEFAULT ''` — email to impersonate
- `google_calendar_id TEXT NOT NULL DEFAULT 'primary'`

Store methods:
- `SetGoogleConfig(ctx, workspaceID, encJSON []byte, subject, calendarID string) error`
- `GetGoogleConfig(ctx, workspaceID) (encJSON []byte, subject, calendarID string, err error)`

Credentials are **never** serialized into any JSON response.

## CalendarProvider + selection + flag

`application`:
```go
var ErrGoogleNotConfigured = errors.New("google not configured")

type CalendarProvider interface {
    For(ctx context.Context, workspaceID uuid.UUID) (CalendarService, error)
}
```
- `Services.Calendar` changes from `CalendarService` to `CalendarProvider`.
- `CreateMeeting`: `cal, err := s.Calendar.For(ctx, wid)` — if `err` (incl. `ErrGoogleNotConfigured`), return it; the handler maps `ErrGoogleNotConfigured` → HTTP 400. Otherwise `cal.CreateEvent(...)`.
- `CancelMeeting`: `cal, err := s.Calendar.For(ctx, wid)`; if `err == nil && m.GoogleEventID != ""` → best-effort `cal.DeleteEvent(...)`. A provider error does NOT block the DB cancel.

Providers:
- `stub.Provider` (in `infrastructure/calendar/stub`): `For` returns the existing stub `Service` for any workspace.
- `google.Provider` (in `infrastructure/calendar/google`): holds `*postgres.Store` + `*crypto.TokenCipher`. `For` loads the workspace google config; if `encJSON` is empty → `application.ErrGoogleNotConfigured`; else decrypts and returns a google adapter bound to those creds (cache the built `*calendar.Service` per workspace).

Config: new `CALENDAR_STUB bool` in `platform/config`. Wiring in `app.go`:
```go
var calProvider application.CalendarProvider
if cfg.CalendarStub {
    calProvider = stub.NewProvider()
} else {
    calProvider = google.NewProvider(store, cipher)
}
// Services{..., Calendar: calProvider}
```
Add `CALENDAR_STUB=true` to `deploy/.env.example` and `.github/workflows/_smoke.yml` env so local/CI use the stub and `TestSmokeMeetings` stays green.

## Google adapter (`infrastructure/calendar/google`)

New dependency: `google.golang.org/api/calendar/v3` (+ `google.golang.org/api/option`). `golang.org/x/oauth2/google` is already vendored.

Client build (domain-wide delegation):
```go
cfg, err := google.JWTConfigFromJSON(saJSON, calendar.CalendarScope)
cfg.Subject = subject
svc, err := calendar.NewService(ctx, option.WithHTTPClient(cfg.Client(ctx)))
```

- `buildEvent(e application.CalendarEvent) *calendar.Event` — **pure** (unit-tested): `Summary`, `Description`, `Start/End{DateTime: RFC3339}`, `Attendees[]`, `ConferenceData.CreateRequest{RequestId, ConferenceSolutionKey{Type:"hangoutsMeet"}}`.
- `CreateEvent`: `svc.Events.Insert(calendarID, buildEvent(e)).ConferenceDataVersion(1).Do()` → `CalendarResult{EventID: created.Id, MeetLink: created.HangoutLink}`.
- `DeleteEvent`: `svc.Events.Delete(calendarID, eventID).Do()`.

## Config endpoint (extend existing integrations)

Extend `application.PatchIntegrations` + `GetIntegrations` and the `PATCH /api/workspaces/:id/integrations` handler:
- Accept `google_sa_json` (input only; encrypted + stored via `SetGoogleConfig`), `google_subject`, `google_calendar_id`.
- `GetIntegrations` returns `has_google bool`, `google_subject`, `google_calendar_id` — never the JSON.

## Testing

- **Unit:** `buildEvent` (conferenceData createRequest present, attendees mapped, times RFC3339, summary/description). Pure, no network.
- **Unit:** stub provider returns a working `CalendarService`.
- **Smoke:** `_smoke.yml` sets `CALENDAR_STUB=true`; `TestSmokeMeetings` passes unchanged (stub provider).
- **Manual/integration (out of CI):** with real per-workspace creds, create a meeting and confirm a real Meet link + calendar event; delete removes it.

## Out of scope (later)

- Live-Google integration test in CI.
- Frontend UI for entering the SA JSON (the integrations PATCH accepts it; UI is separate).
- Calendar event **updates** (belongs to the edit-meeting increment) and attendee re-sync.
- Retry/backoff on Google API errors (best-effort + log for now).
- Recurrence series expansion, conflict detection, free-slot checker, notifications, bot registration (separate increments).
