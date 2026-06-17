# WS2d-2 — Google Calendar Adapter Tests (httptest) (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, WS2 / WS2d sub-phase **2** (external adapters). Follows WS2d-1.

## Goal

Unit-test the Google Calendar adapter (`CreateEvent`/`UpdateEvent`/`UpdateAttendees`/`DeleteEvent`) against an in-process `httptest` Calendar API, plus the pure event builders and the probe error classifiers, plus the `Provider.For` config-validation branches (via a tiny `configStore` interface). Defer the live SA-credential → JWT → real-Google paths to WS4 E2E.

## Background — verified current state

- `adapter{svc *calendar.Service, calendarID string}` (unexported; a white-box `package google` test can construct it directly). Methods call `a.svc.Events.Insert/Patch/Delete(...).Context(ctx).Do()`.
  - `CreateEvent`: `Events.Insert(calID, buildEvent(e, uuid)).ConferenceDataVersion(1).Do()`; MeetLink = `created.HangoutLink`, else first `ConferenceData.EntryPoints` with `EntryPointType=="video"`.
  - `UpdateEvent`: `Events.Patch(calID, id, buildPatch(e)).SendUpdates("all").Do()`.
  - `UpdateAttendees`: `Events.Patch(calID, id, {Attendees, ForceSendFields:["Attendees"]}).SendUpdates("all").Do()`.
  - `DeleteEvent`: `Events.Delete(calID, id).Do()`.
  - Pure helpers: `buildEvent(e, requestID)`, `buildPatch(e)`, `attendeeList(emails)`.
- `probe.go`: `Probe(ctx, saJSON, subject, calendarID)` — `JWTConfigFromJSON` (fails fast on bad JSON → `ErrJSONParse`, no network), then `calendar.NewService` + `Calendars.Get(...).Do()`, mapping errors via pure classifiers `isJSONParseErr`/`isGoogleAPIDisabled`/`isImpersonationFail` (over `error`/`*googleapi.Error`). `prober.go`: `Prober.Probe` → `mapProbeError` maps `ErrJSONParse/ErrAPIDisabled/ErrSubject/ErrCalendar` → `docalendar.ErrProbe*`.
- `provider.go`: `Provider{store *postgres.Store, cipher, cache sync.Map}`; `For` calls `store.GetGoogleConfig(ctx, orgID) (enc []byte, subject, calendarID string, err error)`, returns `docalendar.ErrNotConfigured` when `len(enc)==0 || subject==""`, defaults `calendarID="primary"`, checks `cache` (key `orgID|subject|calendarID|sha256(enc)`), else decrypt→JWT→`NewService`. Only `GetGoogleConfig` is used from the store.
- Google API client honors `option.WithEndpoint(url)` + `option.WithHTTPClient(client)` — lets a test point the service at an `httptest.Server`.

## Decisions (from brainstorming)

- **Adapter + helpers + probe classifiers + bad-SA path: no production refactor** — testable as-is.
- **Include the small `Provider` refactor**: extract a `configStore` interface so `For()`'s config-validation/default/cache branches are unit-testable with a fake. `main.go` stays unchanged (`*postgres.Store` satisfies `configStore`).
- **Defer to WS4 E2E**: `Provider.For`'s SA-decrypt→JWT→`NewService` and `Probe`'s valid-SA `Calendars.Get` (need real Google credentials/token exchange).

## Design

### A. Adapter via httptest (`adapter_test.go`, `package google`)

A test helper builds a `*calendar.Service` against an `httptest.Server`:
```go
svc, _ := calendar.NewService(ctx, option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))
a := &adapter{svc: svc, calendarID: "primary"}
```
(If `NewService` rejects the `WithHTTPClient`+endpoint combination, add `option.WithoutAuthentication()` — the plan notes this fallback.)

The httptest handler records the last request (method, path, query, decoded body) and returns canned JSON. Tests:
- **CreateEvent_HangoutLink:** handler returns `{"id":"evt1","hangoutLink":"https://meet.google.com/abc"}` → result `{EventID:"evt1", MeetLink:"https://meet.google.com/abc"}`; assert request was `POST .../calendars/primary/events` with `conferenceDataVersion=1`.
- **CreateEvent_ConferenceEntryPoint:** no `hangoutLink`, but `conferenceData.entryPoints=[{entryPointType:"video", uri:"…/xyz"}]` → MeetLink = the video URI.
- **CreateEvent_APIError:** handler returns 500/403 → adapter returns a non-nil error.
- **UpdateEvent:** `PATCH .../events/evt1?...sendUpdates=all`; body has summary/description/start/end; returns nil.
- **UpdateAttendees:** `PATCH` body has the attendee list (+ ForceSendFields semantics — attendees present); `sendUpdates=all`.
- **DeleteEvent:** `DELETE .../events/evt1` → nil; error status → error.

### B. Pure helpers + probe classifiers (`builders_test.go` / `probe_test.go`)

- `buildEvent`: summary/description, RFC3339 start/end, attendees mapped, `ConferenceData.CreateRequest` set with the passed requestID + `hangoutsMeet`.
- `buildPatch`: summary/description/start/end only (no attendees/conference).
- `attendeeList`: emails → `[]*calendar.EventAttendee`; empty → nil.
- `isJSONParseErr`: true for "private key should be a PEM" and the asn1 parse message; false otherwise.
- `isGoogleAPIDisabled`: true for a `*googleapi.Error{Code:403, Errors:[{Reason:"accessNotConfigured"}]}` and for messages "has not been used"/"is disabled"; false for a 404.
- `isImpersonationFail`: true for "unauthorized_client", "Not Authorized to access this resource", and a `*googleapi.Error{Code:401}`; false otherwise.
- `mapProbeError`: each sentinel → the right `docalendar.ErrProbe*`; nil → nil; unknown → `ErrProbeCalendar`.
- **bad-SA-JSON:** `Probe(ctx, "not json", "s", "primary")` → `errors.Is(err, ErrJSONParse)`; `Prober{}.Probe(ctx, "not json", …)` → `docalendar.ErrProbeSAInvalid`. (No network.)

### C. Provider `configStore` refactor + tests (`provider.go`, `provider_test.go`)

Refactor (behavior-preserving):
```go
type configStore interface {
	GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error)
}
type Provider struct { store configStore; cipher *crypto.TokenCipher; cache sync.Map }
func NewProvider(store configStore, cipher *crypto.TokenCipher) *Provider { ... }
var _ configStore = (*postgres.Store)(nil)
```
`main.go`'s `calendargoogle.NewProvider(store, cipher)` is unchanged (`*postgres.Store` satisfies `configStore`).

Tests with a `fakeConfigStore`:
- **For_ErrNotConfigured:** empty `enc` → `docalendar.ErrNotConfigured`; empty `subject` → same.
- **For_CacheHit:** pre-store an `*adapter` in `p.cache` under the computed key (`orgID|subject|calendarID|hex(sha256(enc))`, `calendarID` defaulted to `"primary"` when blank); `For` returns that cached adapter without decrypting — this also exercises the `calendarID="primary"` default and the cache path. (Use a small exported-for-test helper or replicate the key formula in the test; the plan picks one.)

The decrypt→JWT→`NewService` path is **not** exercised here (needs real SA JSON) — deferred to E2E.

## Testing / verification

- `go test -race ./internal/infrastructure/calendar/google/...` green (httptest is in-process; no real network).
- `go build ./...` + `go vet ./...` confirm `main.go`/callers still compile after the `configStore` change.
- `golangci-lint run ./...` clean.

## Risks & mitigations

- **google-api option combo for httptest.** Mitigation: plan specifies `WithEndpoint`+`WithHTTPClient`, with `WithoutAuthentication()` as a documented fallback; the implementer confirms `NewService` succeeds.
- **Request-path/query assertions brittle across client versions.** Mitigation: assert on method + path suffix + key query params (`sendUpdates`, `conferenceDataVersion`) and decoded body fields, not exact full URLs.
- **Cache-key replication in the test.** Mitigation: compute the key with the exact formula from `For` (documented in the plan); or add a tiny unexported test-only key helper used by both.
- **Provider refactor behavior drift.** Mitigation: only the store field/param type changes; `main.go` unchanged; gated by build/vet/test/lint + CI.

## Done criteria

- `adapter_test.go` (httptest), `builders_test.go`, `probe_test.go`, `provider_test.go` cover §A–§C.
- `configStore` interface added; `Provider`/`NewProvider` use it; `var _ configStore = (*postgres.Store)(nil)`; `main.go` unchanged; module builds.
- `go test -race ./...` + `golangci-lint run ./...` pass; SA/JWT/real-Google paths explicitly left to WS4 E2E.
