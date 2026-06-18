# Slice 1a — Per-user Calendar OAuth + Token Vault (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 1 (cross-calendar wedge)**, slice **1a**.

## Epic context

The original meetings ТЗ (§4–§8) is complete; the SaaS pivot is structurally done
(web SSO + magic-link, multi-tenant orgs/members/invites, web dashboard + landing +
mini-app). The next feature push targets three areas chosen by the user:

1. **Complete the product wedge** — unified cross-calendar availability (Google + Microsoft).
2. **Activation & onboarding.**
3. **New end-user capability** — scheduling/booking links (specced later).

**Calendar model (locked):** *hybrid — per-user OAuth, corporate SA optional.* Each member
connects their own Google/Microsoft calendar via OAuth; the existing one-corporate-SA path
remains as a fallback for orgs that want it.

**Decomposition + sequence (locked):** 1a → 1b → 1c → 2a → 3.

- **1a (this spec)** — per-user calendar OAuth + token vault; connecting routes the
  organizer's new meetings onto their own Google calendar (SA fallback). Both web + mini-app.
- **1b** — Microsoft Graph calendar adapter (connect + event CRUD + free/busy).
- **1c** — unified availability: conflict checker + free-slots query each participant's
  connected calendar across providers.
- **2a** — first-run onboarding (signup → connect calendar → invite → first meeting).
- **3** — scheduling/booking links.

## Goal

Let an authenticated user connect their **own Google calendar** via OAuth, persist
refreshable per-user tokens encrypted in a vault, and route the **organizer's newly
created/edited meetings onto their own calendar** when connected — falling back to the
existing corporate service-account path when not. Expose connect/disconnect/status on
**both** the web dashboard and the Telegram mini-app.

## Background — verified current state

- **OAuth is login-only.** `internal/infrastructure/oauth/{google,microsoft}/provider.go`
  request scopes `openid email profile` and PKCE; `Exchange` extracts the `id_token` and
  **discards the access/refresh token**. No calendar scope, no token persistence.
- **Calendar resolution is org-wide.** `application.CalendarProvider.For(ctx, orgID uuid.UUID)`
  returns one `docalendar.Service`. The Google implementation
  (`internal/infrastructure/calendar/google/provider.go`) decrypts the org SA JSON, builds a
  JWT client (`JWTConfigFromJSON` + `Subject`), and constructs a `*calendar.Service`; the
  `adapter{svc, calendarID}` wraps it. `.For(` is called at ~10 event-mutating sites:
  `application/command/meetings.go` (create/update/cancel), `application/series_edit.go`
  (6 sites), `application/participants.go` (1 site).
- **The adapter only needs a `*calendar.Service`.** A per-user OAuth token source builds the
  same service, so the existing (and already unit-tested) `adapter` CRUD is reused unchanged.
- **`TokenCipher`** lives at `internal/infrastructure/crypto` with `Encrypt`/`Decrypt`
  (currently encrypts the org SA JSON). Reused for per-user tokens.
- **Web auth** handlers + state/PKCE handling live in
  `internal/delivery/http/handlers/web_auth.go`; web sessions are server-side cookies
  (`web_session`). TMA JWT auth gates `/api/miniapp/*`.
- **Users bound by email.** `platform_users` (web) and `bot_users` (Telegram) are distinct
  tables bound by corporate email.

## Decisions (from brainstorming)

- **Connection keyed by email**, not surface-specific user id — `(email, provider)` composite
  PK. A connection made on web or Telegram is shared automatically (both are bound by email,
  and the meetings domain identifies the organizer by email).
- **Connect + route creation** (not vault-only): connecting changes behavior — the organizer's
  new meetings are created on their own calendar, with SA fallback.
- **Change the `For` signature** to be organizer-aware (single resolution point) rather than
  adding a parallel `ForOrganizer` method.
- **Both surfaces** in 1a; the OAuth round-trip always happens in a real browser. The mini-app
  launches the connect URL via `Telegram.WebApp.openLink` and polls status.
- **Request read scope now** (`calendar.events` + `calendar.readonly`) so slice 1c needs no
  re-consent.

## Design

### A. Data model — `calendar_connections`

New migration (per `migrations.mdc`):

```sql
CREATE TABLE calendar_connections (
    email             CITEXT      NOT NULL,
    provider          TEXT        NOT NULL,           -- 'google' | 'microsoft'
    access_token_enc  BYTEA       NOT NULL,
    refresh_token_enc BYTEA       NOT NULL,
    expiry            TIMESTAMPTZ NOT NULL,
    scopes            TEXT        NOT NULL DEFAULT '', -- space-joined granted scopes
    connected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (email, provider)
);
```

(`citext` already used elsewhere; if the extension is not present, the migration enables it,
matching existing migration conventions.)

Repo methods on `postgres.Store` (and the application port the resolver depends on):

- `UpsertCalendarConnection(ctx, conn model.CalendarConnection) error` — encrypts tokens.
- `GetCalendarConnection(ctx, email, provider string) (model.CalendarConnection, error)` —
  `model.IsNotFound` on absence.
- `DeleteCalendarConnection(ctx, email, provider string) error`.
- `ListCalendarConnections(ctx, email string) ([]model.CalendarConnection, error)` — for status.

`model.CalendarConnection` holds decrypted `AccessToken`/`RefreshToken` in memory; the repo
encrypts on write and decrypts on read.

### B. Connection flow (surface-agnostic, email-bound)

A new `calendar_connect` handler group. Each endpoint accepts **either** a web session **or**
a TMA JWT (reuse both auth middlewares; resolve the caller's email from whichever succeeds).

1. **Initiate** — `POST /api/calendar/connect/google/start`
   - Build an auth URL via a calendar-OAuth config (new method on the Google provider, or a
     small dedicated calendar-oauth helper): scopes
     `https://www.googleapis.com/auth/calendar.events` +
     `https://www.googleapis.com/auth/calendar.readonly`, `access_type=offline`,
     `prompt=consent`, PKCE (`code_challenge`/`S256`), and a **signed state** that carries the
     authenticated caller's **email** + PKCE verifier + a short expiry (mirror the existing
     `web_auth.go` state/PKCE persistence; store verifier server-side keyed by state).
   - Response: `{ "auth_url": "..." }`.

2. **Callback** — `GET /api/calendar/connect/google/callback?code=&state=`
   - Verify + consume state (email + verifier). Exchange the code (PKCE) **keeping the full
     `*oauth2.Token`** (access + refresh + expiry), unlike login.
   - Upsert `calendar_connections` by `(email, google)` with encrypted tokens + granted scopes.
   - Render a minimal self-contained HTML page: "Calendar connected — you can close this tab"
     (the mini-app polls; the web app detects via redirect/opener).

3. **Status** — `GET /api/calendar/connections`
   - `[{ "provider": "google", "connected": true, "email": "...", "scopes": "...", "stale": false }]`.

4. **Disconnect** — `DELETE /api/calendar/connections/google`
   - Delete the row; best-effort token revoke (Google revoke endpoint), ignore revoke errors.

### C. Provider resolution (organizer-aware)

Change the port:

```go
type CalendarProvider interface {
    For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (CalendarService, error)
}
```

Google `Provider.For(ctx, orgID, organizerEmail)`:

1. Look up `GetCalendarConnection(ctx, organizerEmail, "google")`.
   - **Found:** build an `oauth2.Config` (calendar scopes) and a **self-persisting
     `TokenSource`** — wrap `cfg.TokenSource(ctx, tok)` so that when it refreshes, the new token
     is written back (`UpsertCalendarConnection`, encrypted). Build `*calendar.Service` via
     `option.WithHTTPClient(oauth2.NewClient(ctx, persistingSource))` (or
     `option.WithTokenSource`). Wrap in the existing `adapter{svc, calendarID:"primary"}`.
   - **Not found (`IsNotFound`):** fall through to the existing org-SA path
     (`GetGoogleConfig` → JWT → service). Behavior unchanged.
2. Cache per-user adapters by `(organizerEmail|tokenfingerprint)` similarly to the SA cache
   (invalidate is not required for 1a; the persisting source keeps the access token fresh, and
   the cache key changes when the refresh token rotates — document the key formula in the plan).

Update all ~10 `.For(` call sites to pass the organizer's email:

- `command/meetings.go` create/update/cancel — the organizer is the meeting's organizer
  (caller on create; `meeting.Organizer` on update/cancel).
- `series_edit.go` (6 sites) — `meeting.Organizer` / series organizer.
- `participants.go` (1 site) — `meeting.Organizer`.

Update the **stub** provider (`calendar/stub`) and all tests/fakes to the new signature.

### D. Both surfaces (UI)

- **Web (admin, `apps/admin`):** a "Calendars" card on the Settings (or Profile) page —
  Connect Google button (POST `/start` → `window.location = auth_url`), Connected/Disconnect
  state from `GET /connections`. FSD: `entities/calendar-connection/{types,api,queries}` +
  a `features/calendar-connections` card. i18n keys ru/en/kk.
- **Mini-app (`apps/mini-app`):** a "Connected calendar" row in Profile — "Connect" calls
  `Telegram.WebApp.openLink(auth_url)` (external browser performs OAuth); on window focus /
  pull-to-refresh, re-query `GET /connections` to flip to "Connected." Disconnect inline.
  Lightweight, no new heavy deps. i18n keys ru/en/kk.

### E. Error handling

- **Refresh failure** (revoked/expired refresh token): resolver marks the connection `stale`
  (a flag derivable from a failed refresh — surfaced via status) and **falls back to the org
  SA** so meeting creation never hard-fails. Status shows "reconnect needed."
- **Bad/expired/replayed state** on callback → `400`.
- **Token revoked on disconnect** → delete row regardless of revoke-endpoint result.
- No tokens, scopes, or `code`/`state` values in logs (per AGENTS.md logging rules); log
  `calendar_connected` / `calendar_refresh_failed` with `provider` + email-hash only.

## Testing / verification

- **Token vault repo** (testcontainers, `package postgres_test`): encrypt round-trip,
  upsert-by-email overwrites, get/list/delete, `IsNotFound` semantics.
- **Connect flow** (httptest OAuth server, reusing the WS2d httptest/OIDC pattern): `/start`
  URL shape (scopes, `access_type=offline`, `prompt=consent`, PKCE present, state carries
  email); `/callback` exchanges code, persists tokens, binds to state email; replayed/expired
  state → 400.
- **Provider resolution** (fake connection store): connected organizer → per-user adapter;
  no connection → SA fallback; refresh writes the rotated token back to the store.
- **Call-site migration:** `go build ./...` + `go vet ./...` green after the `For` signature
  change; stub + all fakes updated.
- `go test -race ./...` + `golangci-lint run ./...` clean.
- Frontend: admin + mini-app `typecheck` + `lint` + `build` green; new i18n keys present in
  en/ru/kk (key-parity is compile-enforced per the existing i18n setup).

## Risks & mitigations

- **Google OAuth consent screen / scope verification.** Calendar scopes are sensitive; the
  Google Cloud project's consent screen must list them (and may need verification for external
  users). *Mitigation:* ops/config note in the plan; testing uses an httptest OAuth server, not
  real Google. Reuse the existing login OAuth client id/secret (add scopes) where possible.
- **`For` signature change blast radius (~10 sites + stub + tests).** *Mitigation:* mechanical;
  gated by build/vet/test/lint; each call site already has the organizer email.
- **Telegram WebView OAuth.** Redirect-OAuth can't complete inside the WebView. *Mitigation:*
  always perform OAuth in an external browser via `openLink`; connection is email-keyed so the
  mini-app only launches + polls.
- **Token-source write-back correctness.** A naive `TokenSource` won't persist refreshes.
  *Mitigation:* explicit self-persisting wrapper; unit-tested via the fake store.
- **Per-user adapter cache staleness on refresh.** *Mitigation:* the persisting source keeps
  the access token fresh in place; document the cache key + that 1a does not require explicit
  invalidation.

## Done criteria

- `calendar_connections` migration + repo methods + `model.CalendarConnection`.
- Connect/callback/status/disconnect endpoints accept web session **or** TMA JWT; tokens
  persisted encrypted; OAuth round-trip works against the httptest server in tests.
- `CalendarProvider.For` is organizer-aware; connected organizer's new/edited meetings land on
  their **own** Google calendar; SA fallback preserved; all call sites + stub + tests updated.
- Web + mini-app expose connect/disconnect/status with ru/en/kk strings.
- `go test -race ./...` + `golangci-lint run ./...` + FE typecheck/lint/build all green.
- MS Graph, participant-availability reads, and onboarding are explicitly deferred (1b/1c/2a).
