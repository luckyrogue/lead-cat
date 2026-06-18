# Slice 1b — Microsoft Graph Calendar Adapter (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 1 (cross-calendar wedge)**, slice **1b**. Follows 1a.

## Epic context

1a shipped per-user Google calendar OAuth (token vault keyed by email, organizer-aware
`CalendarProvider.For`, connect/callback/status/disconnect on web + mini-app, SA fallback).
1b makes **Microsoft** a real calendar source (today it is only a web *login* provider).
After 1b the product is genuinely multi-platform: **Google Meet** for Google-connected
organizers, **Microsoft Teams** for Microsoft-connected organizers.

**Sequence:** 1a [done] → 1b [this] → 1c (unified availability) → 2a (onboarding) → 3 (booking links).

## Goal

Let a user connect their own **Microsoft** calendar via OAuth; create/update/cancel the
meetings they organize on their MS calendar as **Teams** online meetings (join URL surfaced
as the meeting link); resolve the correct provider per organizer (MS / Google / SA fallback);
and read MS free/busy (`getSchedule`) for later consumption by the availability engine.

## Decisions (from brainstorming)

- **MS-organized meetings get a native Teams meeting** (`isOnlineMeeting:true`,
  `onlineMeetingProvider:"teamsForBusiness"`); `MeetLink` = `onlineMeeting.joinUrl`. The
  product is no longer Meet-only.
- **Raw HTTP against Graph REST** (`https://graph.microsoft.com/v1.0`) with typed structs over
  an `oauth2`-authed `*http.Client` — NOT the `msgraph-sdk-go` (too heavy, awkward for
  httptest). Base URL injectable so tests point at an `httptest.Server`.
- **Free/busy (`getSchedule`) built in 1b** even though nothing consumes it until 1c (defined
  as a thin interface + MS impl, httptest-tested).
- **Resolution precedence = most-recently-updated connection wins** (tiebreak when a user
  connected both Google and Microsoft).
- **Keep the `GoogleEventID` column** as a generic external-event-id holder (no rename — KISS).
- The `BusyReader` interface shape may be **refined in 1c** when the engine consumes it.

## Background — verified current state

- `docalendar.Service` is provider-neutral: `CreateEvent(ctx, CalendarEvent) (CalendarResult{EventID, MeetLink}, error)`,
  `UpdateEvent(ctx, id, CalendarEvent) error`, `UpdateAttendees(ctx, id, []string) error`,
  `DeleteEvent(ctx, id) error`. The MS adapter implements the same interface unchanged.
- `application.CalendarProvider.For(ctx, organizationID uuid.UUID, organizerEmail string)`
  (1a) currently dispatches to the Google `Provider` (Google-connection-or-SA).
- `application.CalendarConnector` port + `Services.ConfigureCalendarConnectors`/`CalendarConnectorByName`
  exist; the connect flow `/api/calendar/connect/:provider/{start,callback}` +
  `/api/calendar/connections[/:provider]` (and `/api/miniapp/calendar/...`) is already
  `:provider`-parameterized and surface-agnostic (email-keyed).
- `calendar_connections` PK is `(email, provider)`; `provider` already holds `'google'` |
  `'microsoft'`. Repo: `GetCalendarConnection`, `UpsertCalendarConnection`,
  `ListCalendarConnections(email)`, `DeleteCalendarConnection`.
- `oauth/microsoft/provider.go` is the **login-only** SSO provider (scopes `openid email profile`,
  discards tokens). The MS calendar connector is new and separate, mirroring 1a's
  `oauth/google/calendar_connector.go`.
- 1a's self-persisting `savingSource` token source lives in `infrastructure/calendar/google`;
  the pattern is reused for MS (a small equivalent in the microsoft package).
- Meeting model stores `GoogleEventID` + `MeetLink` (both already populated from
  `CalendarResult`); no schema change needed.

## Design

### A. MS calendar OAuth connector (`oauth/microsoft/calendar_connector.go`)

Mirror `oauth/google/calendar_connector.go`:
- `NewCalendarConnector(clientID, clientSecret string) *CalendarConnector`; `endpoint oauth2.Endpoint`
  default = Microsoft v2.0 (`login.microsoftonline.com/common/oauth2/v2.0/{authorize,token}`),
  overridable for httptest.
- Scopes: `https://graph.microsoft.com/Calendars.ReadWrite`,
  `https://graph.microsoft.com/OnlineMeetings.ReadWrite`, `offline_access`, `openid`, `email`,
  `profile`.
- `AuthURL(state, challenge, redirectURL)` — `access_type=offline` equivalent + `prompt=consent`
  + PKCE S256.
- `Exchange(ctx, code, verifier, redirectURL) (application.CalendarToken, error)` — keeps access
  + refresh + expiry + granted scopes.
- `OAuthConfig(redirectURL string) *oauth2.Config` (for the refreshing token source, like Google).
- `Name() string { return "microsoft" }`; `var _ application.CalendarConnector = (*CalendarConnector)(nil)`.
- Registered in `cmd/server/main.go` under `"microsoft"` when MS client id/secret are set
  (reuse the existing Microsoft SSO credentials).

### B. MS event adapter (`infrastructure/calendar/microsoft/`)

`adapter{httpClient *http.Client, baseURL string}` (baseURL default `https://graph.microsoft.com/v1.0`,
injectable). Implements `docalendar.Service`:

- **CreateEvent:** `POST {base}/me/events`, body:
  ```json
  { "subject": "...", "body": {"contentType":"text","content":"..."},
    "start": {"dateTime":"<RFC3339 no zone>","timeZone":"UTC"},
    "end":   {"dateTime":"...","timeZone":"UTC"},
    "attendees": [{"emailAddress":{"address":"..."},"type":"required"}],
    "isOnlineMeeting": true, "onlineMeetingProvider": "teamsForBusiness" }
  ```
  Response → `CalendarResult{EventID: resp.id, MeetLink: resp.onlineMeeting.joinUrl}`. (Times sent
  as Graph `dateTimeTimeZone`; convert the domain event's UTC instants.)
- **UpdateEvent:** `PATCH {base}/me/events/{id}` with subject/body/start/end.
- **UpdateAttendees:** `PATCH {base}/me/events/{id}` with the attendees array.
- **DeleteEvent:** `DELETE {base}/me/events/{id}` (204/404 tolerated per existing semantics).
- Non-2xx Graph response → a wrapped error including the Graph `error.code` (no tokens logged).
- Typed structs in `events.go`; the adapter has no knowledge of OAuth (it gets a ready
  `*http.Client`).

A builder (in the microsoft package) constructs the adapter from a `model.CalendarConnection`'s
tokens via `connector.OAuthConfig("").TokenSource(ctx, tok)` wrapped in a self-persisting
source (writes refreshed tokens back via `UpsertCalendarConnection`), then `oauth2.NewClient`.

### C. Composite multi-provider resolver

New composite `CalendarProvider` (e.g. `infrastructure/calendar/resolver` or a small type wired
in `main.go`) holding: the connection store (`ListCalendarConnections`), the 1a Google
`*google.Provider`, and an MS adapter factory (connector + connection store):
```
For(ctx, orgID, organizerEmail):
  if organizerEmail != "":
    conns := store.ListCalendarConnections(organizerEmail)   // ignore error → fall through
    best := most-recently-updated(conns)                     // by UpdatedAt
    if best.Provider == "microsoft": 
        svc, ok := msFactory.For(ctx, best); if ok: return svc, nil   // failure → fall through
  return googleProvider.For(ctx, orgID, organizerEmail)        // Google-conn-or-SA (1a, unchanged)
```
- Most-recent-wins tiebreak. Any MS build/refresh error → fall through to the Google/SA path
  (meeting creation never hard-fails — same invariant as 1a).
- `main.go` wires `calProvider = resolver.New(store, googleProvider, msFactory)` and passes it as
  `Services.Calendar`. The Google provider is constructed as today.
- Update/delete re-resolve by organizer (documented limitation: disconnecting strands existing
  events to the fallback path).

### D. Free/busy — `BusyReader` (built now, consumed in 1c)

Define a thin interface (in `domain/calendar` or `application`, leaf-safe):
```go
type Interval struct { Start, End time.Time }
type BusyReader interface {
    BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]Interval, error)
}
```
MS impl on the adapter via `POST {base}/me/calendar/getSchedule`
(`{"schedules":[emails],"startTime":{...},"endTime":{...},"availabilityViewInterval":30}`) →
parse `scheduleItems` busy blocks per email. Queried as the connected MS user. httptest-tested.
**Not wired** into the conflict/free-slot engine in 1b. (1c adds the Google impl + engine wiring
and may refine the interface.)

### E. Frontend (both surfaces)

The 1a entity (`entities/calendar-connection`) already takes a `provider` param in
`startConnect`/`disconnect`, and `connections` returns a list. Additions:
- **Admin** `calendar-connections-card`: render a Microsoft row/button alongside Google
  (Connect Microsoft / Connected as {email} / Disconnect), driven by the connections list.
- **Mini-app** `calendar-connection-row`: same, with the `openLink` external-browser flow.
- i18n: add `...microsoft`/`connectMicrosoft` keys to en/ru/kk in both apps (admin formal,
  mini-app informal). Key parity is compile-enforced.

## Testing / verification

- MS adapter (`httptest.Server`, baseURL override): CreateEvent body shape (`isOnlineMeeting`,
  `onlineMeetingProvider`, attendees, UTC times) + `joinUrl` extraction; UpdateEvent/UpdateAttendees
  PATCH bodies; DeleteEvent; Graph 4xx/5xx → error; `getSchedule` busy parsing.
- MS connector: `AuthURL` (scopes incl. `Calendars.ReadWrite` + `OnlineMeetings.ReadWrite`,
  offline, consent, PKCE) + `Exchange` (httptest token endpoint, tokens+scopes returned).
- Composite resolver (fakes): MS-connection→MS adapter; Google-connection→Google delegate;
  none→SA; most-recent tiebreak when both present; MS build error→fallback.
- `env -u GOROOT go build/vet`, `go test -race ./...`, `golangci-lint run ./...` clean.
- Admin + mini-app `typecheck`/`lint`/`build` green; new i18n keys in en/ru/kk.

## Risks & mitigations

- **MS consent / admin-approval for `OnlineMeetings.ReadWrite`.** Some tenants require admin
  consent for online-meeting scopes. *Mitigation:* ops/config note in the plan; tests use
  httptest, not real Graph. Reuse the existing Microsoft SSO client where possible.
- **Graph time formatting.** Graph wants `dateTimeTimeZone` (local datetime + zone), not a Z
  instant. *Mitigation:* send UTC datetimes with `timeZone:"UTC"`; unit-test the marshaled body.
- **Provider re-resolution on update/delete** strands events if the user disconnects. *Mitigation:*
  documented limitation, consistent with 1a; revisit if it bites.
- **Premature `BusyReader` contract** (no 1b consumer). *Mitigation:* keep it minimal; explicitly
  allow 1c to refine it.
- **Two online-meeting platforms in one product.** Naming/UX: a meeting's link may be Teams or
  Meet. *Mitigation:* UI shows whatever `MeetLink` is; no Meet-specific copy on the link itself.

## Done criteria

- MS calendar OAuth connector registered under `"microsoft"`; `/api/calendar/connect/microsoft/...`
  works on web + mini-app; tokens persisted encrypted (reuses 1a's vault + flow).
- MS event adapter implements `docalendar.Service` over raw Graph HTTP; MS-organized meetings
  carry a Teams `joinUrl` as `MeetLink`.
- Composite resolver selects MS / Google / SA by most-recent connection; meeting creation never
  hard-fails on a per-user error.
- MS `BusyReader.BusyTimes` implemented + httptest-tested (unwired; 1c consumes).
- Both frontends expose Microsoft connect/disconnect; ru/en/kk strings present.
- `go test -race ./...` + `golangci-lint` + FE typecheck/lint/build all green.
- Free/busy engine wiring, onboarding, booking links explicitly deferred (1c/2a/3).
