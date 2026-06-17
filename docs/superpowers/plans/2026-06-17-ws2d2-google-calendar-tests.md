# WS2d-2 — Google Calendar Adapter Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unit-test the Google Calendar adapter against an in-process `httptest` Calendar API, plus the pure builders, the probe error classifiers, and the `Provider.For` config-validation branches (via a tiny `configStore` interface).

**Architecture:** White-box `package google` tests construct `&adapter{svc, calendarID}` with a `*calendar.Service` pointed at an `httptest.Server` (verified combo: `option.WithEndpoint(srv.URL)` + `option.WithHTTPClient(srv.Client())` — no `WithoutAuthentication` needed). Pure helpers/classifiers tested directly. `Provider` gets a `configStore` interface so config branches test with a fake; `main.go` unchanged.

**Tech Stack:** Go 1.26, `net/http/httptest`, `google.golang.org/api/calendar/v3` + `option` + `googleapi`, stdlib `testing`.

**Standing constraints (every task):** work on `main`, no branches; commit per task; stage only listed paths (never `git add -A`); `git status` before staging; run Go with `env -u GOROOT`; commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; do NOT touch `.github/workflows/_build.yml`; ignore stale IDE diagnostics, trust `go build`/`test`/`golangci-lint`.

**Verified by spike:** `calendar.NewService(ctx, option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))` + `&adapter{svc, calendarID:"primary"}`; `CreateEvent` issues `POST /calendars/primary/events?conferenceDataVersion=1` and returns `{EventID, MeetLink}` parsed from the JSON response.

---

### Task 1: Adapter via httptest

**Files:** Create `apps/backend/internal/infrastructure/calendar/google/adapter_test.go`

- [ ] **Step 1: Write the tests**

```go
package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type capturedReq struct {
	method string
	path   string
	query  map[string][]string
	body   map[string]any
}

// newTestAdapter spins an httptest Calendar API whose handler returns respJSON
// (status 200) and records the last request into *capturedReq.
func newTestAdapter(t *testing.T, status int, respJSON string) (*adapter, *capturedReq) {
	t.Helper()
	cap := &capturedReq{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method, cap.path, cap.query = r.Method, r.URL.Path, r.URL.Query()
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			cap.body = map[string]any{}
			_ = json.Unmarshal(b, &cap.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if respJSON != "" {
			_, _ = io.WriteString(w, respJSON)
		}
	}))
	t.Cleanup(srv.Close)

	svc, err := calendar.NewService(context.Background(), option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return &adapter{svc: svc, calendarID: "primary"}, cap
}

func sampleEvent() docalendar.CalendarEvent {
	return docalendar.CalendarEvent{
		Title: "Sync", Description: "d",
		Start: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.io", "b@x.io"},
	}
}

func TestCreateEvent_HangoutLink(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1","hangoutLink":"https://meet.google.com/abc"}`)
	res, err := a.CreateEvent(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.EventID != "evt1" || res.MeetLink != "https://meet.google.com/abc" {
		t.Fatalf("result: %+v", res)
	}
	if cap.method != "POST" || cap.path != "/calendars/primary/events" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
	if cap.query.Get("conferenceDataVersion") != "1" {
		t.Fatalf("conferenceDataVersion = %q", cap.query.Get("conferenceDataVersion"))
	}
}

func TestCreateEvent_ConferenceEntryPoint(t *testing.T) {
	a, _ := newTestAdapter(t, 200, `{"id":"evt2","conferenceData":{"entryPoints":[{"entryPointType":"phone","uri":"tel:+1"},{"entryPointType":"video","uri":"https://meet.google.com/xyz"}]}}`)
	res, err := a.CreateEvent(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.MeetLink != "https://meet.google.com/xyz" {
		t.Fatalf("want video entrypoint uri, got %q", res.MeetLink)
	}
}

func TestCreateEvent_APIError(t *testing.T) {
	a, _ := newTestAdapter(t, 500, `{"error":{"code":500,"message":"boom"}}`)
	if _, err := a.CreateEvent(context.Background(), sampleEvent()); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestUpdateEvent_PatchAllUpdates(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1"}`)
	if err := a.UpdateEvent(context.Background(), "evt1", sampleEvent()); err != nil {
		t.Fatalf("update: %v", err)
	}
	if cap.method != "PATCH" || cap.path != "/calendars/primary/events/evt1" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
	if cap.query.Get("sendUpdates") != "all" {
		t.Fatalf("sendUpdates = %q", cap.query.Get("sendUpdates"))
	}
	if cap.body["summary"] != "Sync" {
		t.Fatalf("body summary = %v", cap.body["summary"])
	}
}

func TestUpdateAttendees_PatchWithAttendees(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1"}`)
	if err := a.UpdateAttendees(context.Background(), "evt1", []string{"c@x.io"}); err != nil {
		t.Fatalf("update attendees: %v", err)
	}
	if cap.method != "PATCH" || cap.query.Get("sendUpdates") != "all" {
		t.Fatalf("request: %s sendUpdates=%q", cap.method, cap.query.Get("sendUpdates"))
	}
	att, ok := cap.body["attendees"].([]any)
	if !ok || len(att) != 1 {
		t.Fatalf("attendees body = %v", cap.body["attendees"])
	}
}

func TestDeleteEvent(t *testing.T) {
	a, cap := newTestAdapter(t, 200, ``)
	if err := a.DeleteEvent(context.Background(), "evt1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if cap.method != "DELETE" || cap.path != "/calendars/primary/events/evt1" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
}
```

- [ ] **Step 2: Run + lint**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/infrastructure/calendar/google/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/calendar/google/...` — `0 issues.`

If `DeleteEvent`'s empty 200 body trips the client (it may expect no body or a specific status), adjust the handler to return `204` for DELETE (`if r.Method == "DELETE" { w.WriteHeader(204); return }`) — report the adjustment. Do not weaken the method/path assertions.

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/infrastructure/calendar/google/adapter_test.go
git commit -m "$(cat <<'EOF'
test(calendar/google): adapter CRUD via httptest (MeetLink extraction, request shaping)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Pure builders + probe classifiers

**Files:**
- Create `apps/backend/internal/infrastructure/calendar/google/builders_test.go`
- Create `apps/backend/internal/infrastructure/calendar/google/probe_test.go`

- [ ] **Step 1: builders_test.go**

```go
package google

import (
	"testing"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestBuildEvent(t *testing.T) {
	e := docalendar.CalendarEvent{
		Title: "T", Description: "D",
		Start: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.io", "b@x.io"},
	}
	ev := buildEvent(e, "req-123")
	if ev.Summary != "T" || ev.Description != "D" {
		t.Fatalf("summary/desc: %+v", ev)
	}
	if ev.Start.DateTime != "2026-06-01T10:00:00Z" || ev.End.DateTime != "2026-06-01T11:00:00Z" {
		t.Fatalf("times: %q %q", ev.Start.DateTime, ev.End.DateTime)
	}
	if len(ev.Attendees) != 2 || ev.Attendees[0].Email != "a@x.io" {
		t.Fatalf("attendees: %+v", ev.Attendees)
	}
	if ev.ConferenceData == nil || ev.ConferenceData.CreateRequest == nil ||
		ev.ConferenceData.CreateRequest.RequestId != "req-123" ||
		ev.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
		t.Fatalf("conference data: %+v", ev.ConferenceData)
	}
}

func TestBuildPatch_NoAttendeesOrConference(t *testing.T) {
	ev := buildPatch(docalendar.CalendarEvent{
		Title: "T", Description: "D",
		Start: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"ignored@x.io"},
	})
	if ev.Summary != "T" || ev.Start.DateTime == "" || ev.End.DateTime == "" {
		t.Fatalf("patch core fields: %+v", ev)
	}
	if len(ev.Attendees) != 0 || ev.ConferenceData != nil {
		t.Fatalf("patch must not carry attendees/conference: %+v", ev)
	}
}

func TestAttendeeList(t *testing.T) {
	if attendeeList(nil) != nil {
		t.Fatal("empty -> nil")
	}
	got := attendeeList([]string{"a@x.io", "b@x.io"})
	if len(got) != 2 || got[1].Email != "b@x.io" {
		t.Fatalf("attendeeList: %+v", got)
	}
}
```

- [ ] **Step 2: probe_test.go**

```go
package google

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/api/googleapi"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestIsGoogleAPIDisabled(t *testing.T) {
	disabled := &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "accessNotConfigured"}}}
	if !isGoogleAPIDisabled(disabled) {
		t.Fatal("403 accessNotConfigured should be disabled")
	}
	msg := &googleapi.Error{Code: 403, Message: "Calendar API has not been used in project ... is disabled"}
	if !isGoogleAPIDisabled(msg) {
		t.Fatal("403 'has not been used' should be disabled")
	}
	if isGoogleAPIDisabled(&googleapi.Error{Code: 404}) {
		t.Fatal("404 is not disabled")
	}
	if isGoogleAPIDisabled(errors.New("plain")) {
		t.Fatal("non-googleapi error is not disabled")
	}
}

func TestIsImpersonationFail(t *testing.T) {
	if !isImpersonationFail(errors.New("oauth2: unauthorized_client blah")) {
		t.Fatal("unauthorized_client")
	}
	if !isImpersonationFail(errors.New("...Not Authorized to access this resource...")) {
		t.Fatal("not authorized message")
	}
	if !isImpersonationFail(&googleapi.Error{Code: 401}) {
		t.Fatal("401 googleapi")
	}
	if isImpersonationFail(errors.New("something else")) {
		t.Fatal("unrelated should be false")
	}
}

func TestIsJSONParseErr(t *testing.T) {
	if !isJSONParseErr(errors.New("private key should be a PEM or plain PKCS1 or PKCS8")) {
		t.Fatal("PEM message")
	}
	if isJSONParseErr(errors.New("totally unrelated")) {
		t.Fatal("unrelated should be false")
	}
}

func TestMapProbeError(t *testing.T) {
	cases := map[error]error{
		nil: nil,
		fmt.Errorf("%w: x", ErrJSONParse):   docalendar.ErrProbeSAInvalid,
		fmt.Errorf("%w: x", ErrAPIDisabled): docalendar.ErrProbeAPIDisabled,
		fmt.Errorf("%w: x", ErrSubject):     docalendar.ErrProbeSubject,
		fmt.Errorf("%w: x", ErrCalendar):    docalendar.ErrProbeCalendar,
		errors.New("unknown"):               docalendar.ErrProbeCalendar,
	}
	for in, want := range cases {
		if got := mapProbeError(in); !errors.Is(got, want) && got != want {
			t.Fatalf("mapProbeError(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestProbe_BadSAJSON_NoNetwork(t *testing.T) {
	_, err := Probe(context.Background(), "not json", "s@x.io", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("want ErrJSONParse, got %v", err)
	}
	if _, perr := (Prober{}).Probe(context.Background(), "not json", "s@x.io", "primary"); !errors.Is(perr, docalendar.ErrProbeSAInvalid) {
		t.Fatalf("Prober want ErrProbeSAInvalid, got %v", perr)
	}
}
```

NOTE: `TestProbe_BadSAJSON_NoNetwork` uses `context` — add `"context"` to its import block. Verify the exact "private key should be a PEM" substring against `probe.go`'s `isJSONParseErr` (copy if it differs). The map-iteration in `TestMapProbeError` with a `nil` key is fine in Go.

- [ ] **Step 3: Run + lint + commit**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/infrastructure/calendar/google/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/calendar/google/...` — `0 issues.`

```bash
git add apps/backend/internal/infrastructure/calendar/google/builders_test.go \
  apps/backend/internal/infrastructure/calendar/google/probe_test.go
git commit -m "$(cat <<'EOF'
test(calendar/google): event builders + probe error classifiers + bad-SA path

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Provider configStore refactor + tests

**Files:**
- Modify `apps/backend/internal/infrastructure/calendar/google/provider.go`
- Create `apps/backend/internal/infrastructure/calendar/google/provider_test.go`

- [ ] **Step 1: Refactor provider.go**

Add the interface + assertion and change the struct/constructor. Replace:
```go
type Provider struct {
	store  *postgres.Store
	cipher *crypto.TokenCipher
	cache  sync.Map
}

func NewProvider(store *postgres.Store, cipher *crypto.TokenCipher) *Provider {
	return &Provider{store: store, cipher: cipher}
}
```
with:
```go
type configStore interface {
	GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error)
}

var _ configStore = (*postgres.Store)(nil)

type Provider struct {
	store  configStore
	cipher *crypto.TokenCipher
	cache  sync.Map
}

func NewProvider(store configStore, cipher *crypto.TokenCipher) *Provider {
	return &Provider{store: store, cipher: cipher}
}
```
Leave `For`'s body unchanged (it already calls `p.store.GetGoogleConfig`). `postgres` import stays (used by the assertion + `crypto`/types). Do NOT touch `cmd/server/main.go`.

- [ ] **Step 2: Verify build + main.go untouched**

Run: `cd apps/backend && env -u GOROOT go build ./...` — clean (proves `*postgres.Store` satisfies `configStore` and `main.go`'s `NewProvider(store, cipher)` still compiles).
Run: `git status --short` — `cmd/server/main.go` MUST NOT appear.

- [ ] **Step 3: provider_test.go**

```go
package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type fakeConfigStore struct {
	enc        []byte
	subject    string
	calendarID string
	err        error
}

func (f *fakeConfigStore) GetGoogleConfig(_ context.Context, _ uuid.UUID) ([]byte, string, string, error) {
	return f.enc, f.subject, f.calendarID, f.err
}

var _ configStore = (*fakeConfigStore)(nil)

func TestFor_ErrNotConfigured(t *testing.T) {
	for name, fcs := range map[string]*fakeConfigStore{
		"empty-enc":     {enc: nil, subject: "s@x.io"},
		"empty-subject": {enc: []byte("x"), subject: ""},
	} {
		t.Run(name, func(t *testing.T) {
			p := NewProvider(fcs, nil)
			if _, err := p.For(context.Background(), uuid.New()); err != docalendar.ErrNotConfigured {
				t.Fatalf("want ErrNotConfigured, got %v", err)
			}
		})
	}
}

func TestFor_CacheHit_DefaultsCalendarID(t *testing.T) {
	org := uuid.New()
	enc := []byte("encrypted-bytes")
	subject := "svc@x.io"
	fcs := &fakeConfigStore{enc: enc, subject: subject, calendarID: ""} // blank -> defaults to "primary"
	p := NewProvider(fcs, nil)

	// Pre-seed the cache exactly as For() computes the key, with calendarID defaulted to "primary".
	sum := sha256.Sum256(enc)
	key := org.String() + "|" + subject + "|primary|" + hex.EncodeToString(sum[:])
	want := &adapter{calendarID: "primary"}
	p.cache.Store(key, want)

	got, err := p.For(context.Background(), org)
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	if got != docalendar.Service(want) {
		t.Fatalf("cache hit should return the seeded adapter; got %#v", got)
	}
}
```

NOTE: the cache key formula MUST match `provider.go`'s `For` exactly — read it and replicate (`organizationID.String() + "|" + subject + "|" + calendarID + "|" + hex.EncodeToString(sum[:])`, with `calendarID` already defaulted to `"primary"`). If `For` computes the key with the *pre-default* calendarID, adjust the test's key accordingly (read the order in `For`). The cipher is `nil` in these tests because the cache-hit path returns before any decrypt; if `For` dereferences the cipher before the cache check, pass a real `*crypto.TokenCipher` built via `crypto.NewTokenCipher("0123456789abcdef")` instead — read `For` to confirm ordering.

- [ ] **Step 4: Run + lint + commit**

Run: `cd apps/backend && env -u GOROOT go test -race ./internal/infrastructure/calendar/google/ -v` — all PASS.
Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/calendar/google/...` — `0 issues.`

```bash
git add apps/backend/internal/infrastructure/calendar/google/provider.go \
  apps/backend/internal/infrastructure/calendar/google/provider_test.go
git commit -m "$(cat <<'EOF'
refactor+test(calendar/google): Provider configStore interface + For() config-branch tests

Extracts a configStore interface (main.go unchanged; *postgres.Store satisfies
it) and covers For()'s ErrNotConfigured / calendarID-default / cache-hit paths.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Full verification

**Files:** none.

- [ ] **Step 1:** `cd apps/backend && env -u GOROOT go build ./...` — clean.
- [ ] **Step 2:** `cd apps/backend && env -u GOROOT go test -race ./...` — module-wide green (Docker permitting for postgres).
- [ ] **Step 3:** `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...` — `0 issues.`
- [ ] **Step 4:** `git status --short` — clean; `cmd/server/main.go` never modified.
- [ ] **Step 5 (informational):** after the human pushes, confirm CI green (`gh run watch`).

---

## Notes on execution order
Tasks 1 and 2 are independent (both add only test files). Task 3 modifies `provider.go` (refactor) then adds its test. Run 1 → 2 → 3 → 4. All are in `package google`, so the `newTestAdapter` helper (Task 1) and the fakes (Task 3) coexist in the same test binary — keep helper/type names distinct (`newTestAdapter`, `fakeConfigStore`, `capturedReq`).
