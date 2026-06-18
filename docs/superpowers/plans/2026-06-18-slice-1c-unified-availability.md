# Slice 1c — Unified Cross-Calendar Availability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge each participant's real external-calendar busy times (Google FreeBusy + MS getSchedule) into `MeetingConflicts` and `FreeSlots`, best-effort, via a hybrid union (requester's reader over all emails ∪ each connected participant's own reader).

**Architecture:** A new Google `BusyReader` (FreeBusy) mirrors 1b's MS one. A `BusyResolver` port (email→reader) resolves a per-person reader by most-recent connection. A best-effort, parallel `gatherExternalBusy` orchestrator (in `application`) is merged into both engine functions; `series_conflicts` inherits it. Backend-only.

**Tech Stack:** Go 1.26, `google.golang.org/api/calendar/v3` (FreeBusy), `golang.org/x/oauth2`, `internal/platform/fanio` (bounded parallel), httptest.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-18-slice-1c-unified-availability-design.md`.
- depguard: `application` imports zero `internal/infrastructure` (the `BusyResolver` is a port defined in `application`, impl in infrastructure, wired in `main.go`); the resolver/google/ms calendar packages import `docalendar` + `model`, not `application`.
- **Best-effort:** external calendar access must NEVER fail a conflict/free-slot query — readers that error are logged (`external_busy_fetch_failed`, email-hashed, NO tokens/raw-email) and skipped.
- **Privacy:** external-busy conflicts carry an empty `MeetingName` (free/busy never exposes titles).
- **Union model:** external busy = requester-reader(all emails) ∪ each-connected-participant-own-reader(self), merged/deduped. A participant with no connection contributes only app-DB meetings (unchanged).
- No code comments in new Go files. gofmt all; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green; existing conflict/free-slot/series tests must still pass (regression guard).
- Work on `main`; never `git add -A` (stage explicit paths); **verify HEAD before each commit** (the user commits in parallel — commit your staged files on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference — domain (`internal/domain/calendar/calendar.go`, from 1b):**
```go
type Interval struct { Start, End time.Time }
type BusyReader interface { BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]Interval, error) }
```
**Reference — MS factory (1b, `internal/infrastructure/calendar/microsoft/factory.go`):** `(*Factory).For(ctx, conn model.CalendarConnection) (docalendar.Service, bool)` — the returned concrete `*adapter` ALSO implements `docalendar.BusyReader`.
**Reference — `Conflict` (`internal/application/conflict.go`):** `{ Email, PersonName, MeetingName string; Start, End time.Time }`.
**Reference — `fanio.AllBestEffort(ctx, limit, n int, fn func(ctx, int))`** — bounded parallel, no error propagation.

---

### Task 1: Google `BusyReader` (FreeBusy) + reader factory

**Files:**
- Create: `apps/backend/internal/infrastructure/calendar/google/reader.go`
- Test: `apps/backend/internal/infrastructure/calendar/google/reader_test.go`

**Interfaces:**
- Produces:
  - `google.newGoogleReader(svc *calendar.Service) *googleReader` implementing `docalendar.BusyReader`.
  - `google.NewReaderFactory(connector calendarConnector) *ReaderFactory` with `(*ReaderFactory).For(ctx context.Context, conn model.CalendarConnection) (docalendar.BusyReader, bool)` — builds a `*calendar.Service` from the connection's tokens (self-persisting source, mirroring `userService`/the MS factory) and wraps it.
- Consumes: the existing `calendarConnector` interface (`OAuthConfig(redirectURL) *oauth2.Config`) + `connectionStore` (for write-back) in this package; `model.CalendarConnection`.

- [ ] **Step 1: Write the failing test** — `reader_test.go` (`package google`):
```go
func TestGoogleReader_BusyTimes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendars":{"a@x.com":{"busy":[{"start":"2026-06-20T09:00:00Z","end":"2026-06-20T09:30:00Z"}]}}}`))
	}))
	defer srv.Close()
	svc, err := calendar.NewService(context.Background(), option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	r := newGoogleReader(svc)
	busy, err := r.BusyTimes(context.Background(), []string{"a@x.com"},
		time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(busy["a@x.com"]) != 1 {
		t.Fatalf("expected 1 busy interval, got %v", busy)
	}
	if !busy["a@x.com"][0].Start.Equal(time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("bad start: %v", busy["a@x.com"][0].Start)
	}
}
```

- [ ] **Step 2: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestGoogleReader -v`

- [ ] **Step 3: Implement** — `reader.go`:
```go
package google

import (
	"context"
	"time"

	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type googleReader struct {
	svc *calendar.Service
}

func newGoogleReader(svc *calendar.Service) *googleReader { return &googleReader{svc: svc} }

func (r *googleReader) BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]docalendar.Interval, error) {
	items := make([]*calendar.FreeBusyRequestItem, 0, len(emails))
	for _, e := range emails {
		items = append(items, &calendar.FreeBusyRequestItem{Id: e})
	}
	resp, err := r.svc.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: from.UTC().Format(time.RFC3339),
		TimeMax: to.UTC().Format(time.RFC3339),
		Items:   items,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]docalendar.Interval, len(resp.Calendars))
	for email, cal := range resp.Calendars {
		for _, b := range cal.Busy {
			start, _ := time.Parse(time.RFC3339, b.Start)
			end, _ := time.Parse(time.RFC3339, b.End)
			out[email] = append(out[email], docalendar.Interval{Start: start, End: end})
		}
	}
	return out, nil
}

type ReaderFactory struct {
	conns     connectionStore
	connector calendarConnector
}

func NewReaderFactory(conns connectionStore, connector calendarConnector) *ReaderFactory {
	return &ReaderFactory{conns: conns, connector: connector}
}

func (f *ReaderFactory) For(ctx context.Context, conn model.CalendarConnection) (docalendar.BusyReader, bool) {
	if f.connector == nil {
		return nil, false
	}
	cfg := f.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{AccessToken: conn.AccessToken, RefreshToken: conn.RefreshToken, Expiry: conn.Expiry})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = f.conns.UpsertCalendarConnection(ctx, conn)
	}}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, src)))
	if err != nil {
		return nil, false
	}
	return newGoogleReader(svc), true
}

var _ docalendar.BusyReader = (*googleReader)(nil)
```
(`connectionStore` + `calendarConnector` + `savingSource` already exist in this package from 1a — reuse them. `connectionStore` already has `UpsertCalendarConnection`.)

- [ ] **Step 4: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestGoogleReader -v`

- [ ] **Step 5: gofmt + lint + commit**
```bash
gofmt -w internal/infrastructure/calendar/google/reader.go internal/infrastructure/calendar/google/reader_test.go
golangci-lint run --config ../../config/.golangci.yml ./internal/infrastructure/calendar/google/...
git add apps/backend/internal/infrastructure/calendar/google/reader.go apps/backend/internal/infrastructure/calendar/google/reader_test.go
git commit -m "feat(calendar/google): FreeBusy BusyReader + reader factory"
```

---

### Task 2: `BusyResolver` port + impl + wiring

**Files:**
- Modify: `apps/backend/internal/application/ports.go` — add `BusyResolver` + `BusyReader` alias
- Modify: `apps/backend/internal/application/services.go` — add `Busy BusyResolver` field
- Create: `apps/backend/internal/infrastructure/calendar/resolver/busy.go`
- Test: `apps/backend/internal/infrastructure/calendar/resolver/busy_test.go`
- Modify: `apps/backend/cmd/server/main.go` — build + set `services.Busy`

**Interfaces:**
- Produces:
  - `application.BusyReader = docalendar.BusyReader` (alias); `application.BusyResolver interface { ReaderFor(ctx context.Context, email string) (BusyReader, bool) }`.
  - `resolver.NewBusyResolver(lister connLister, google googleReaderFactory, ms msReaderFactory) *BusyResolver` with `ReaderFor(ctx, email) (docalendar.BusyReader, bool)`.
- Consumes: `model.CalendarConnection`; `google.ReaderFactory` (Task 1, satisfies `googleReaderFactory = For(ctx, conn) (docalendar.BusyReader, bool)`); `microsoft.Factory` (1b — `For(ctx, conn) (docalendar.Service, bool)`, type-assert the result to `docalendar.BusyReader`).

- [ ] **Step 1: Application port** — append to `internal/application/ports.go`:
```go
type BusyReader = docalendar.BusyReader

type BusyResolver interface {
	ReaderFor(ctx context.Context, email string) (BusyReader, bool)
}
```
Add the `docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"` import if not present (it is, in calendar.go — but ports.go may need it; add if the build complains). Add `Busy BusyResolver` to the `Services` struct in `services.go`.

- [ ] **Step 2: Write the failing resolver test** — `resolver/busy_test.go` (`package resolver`):
```go
type fakeBusyReader struct{ tag string }

func (fakeBusyReader) BusyTimes(context.Context, []string, time.Time, time.Time) (map[string][]docalendar.Interval, error) {
	return nil, nil
}

type fakeGoogleRF struct{ ok bool }

func (f fakeGoogleRF) For(context.Context, model.CalendarConnection) (docalendar.BusyReader, bool) {
	return fakeBusyReader{tag: "google"}, f.ok
}

type fakeMSRF struct{ ok bool }

func (f fakeMSRF) For(context.Context, model.CalendarConnection) (docalendar.Service, bool) {
	return msBusyService{}, f.ok // msBusyService implements Service + BusyReader
}

func TestReaderFor_Microsoft(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "microsoft", UpdatedAt: time.Now()}}}
	r := NewBusyResolver(lister, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	rd, ok := r.ReaderFor(context.Background(), "u@x.com")
	if !ok || rd == nil {
		t.Fatal("expected MS reader")
	}
}

func TestReaderFor_Google(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "google", UpdatedAt: time.Now()}}}
	r := NewBusyResolver(lister, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	rd, ok := r.ReaderFor(context.Background(), "u@x.com")
	if !ok || rd.(fakeBusyReader).tag != "google" {
		t.Fatalf("expected google reader, got %v ok=%v", rd, ok)
	}
}

func TestReaderFor_None(t *testing.T) {
	r := NewBusyResolver(fakeLister{}, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	if _, ok := r.ReaderFor(context.Background(), "u@x.com"); ok {
		t.Fatal("expected no reader when no connections")
	}
}
```
Provide `msBusyService` in the test — a type implementing BOTH `docalendar.Service` (4 methods) and `docalendar.BusyReader`. Reuse `fakeLister` from `resolver_test.go` (already in this package from 1b).

- [ ] **Step 3: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/calendar/resolver/ -run TestReaderFor -v`

- [ ] **Step 4: Implement** — `resolver/busy.go`:
```go
package resolver

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type googleReaderFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.BusyReader, bool)
}

type msReaderFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool)
}

type BusyResolver struct {
	lister connLister
	google googleReaderFactory
	ms     msReaderFactory
}

func NewBusyResolver(lister connLister, google googleReaderFactory, ms msReaderFactory) *BusyResolver {
	return &BusyResolver{lister: lister, google: google, ms: ms}
}

func (r *BusyResolver) ReaderFor(ctx context.Context, email string) (docalendar.BusyReader, bool) {
	if email == "" || r.lister == nil {
		return nil, false
	}
	conns, err := r.lister.ListCalendarConnections(ctx, email)
	if err != nil {
		return nil, false
	}
	best, ok := mostRecent(conns)
	if !ok {
		return nil, false
	}
	switch best.Provider {
	case "microsoft":
		if r.ms == nil {
			return nil, false
		}
		if svc, built := r.ms.For(ctx, best); built {
			if br, isReader := svc.(docalendar.BusyReader); isReader {
				return br, true
			}
		}
	case "google":
		if r.google == nil {
			return nil, false
		}
		return r.google.For(ctx, best)
	}
	return nil, false
}
```
(`connLister` + `mostRecent` already exist in this package from 1b's `resolver.go`.)

- [ ] **Step 5: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/calendar/resolver/ -v`

- [ ] **Step 6: Wire `main.go`** — after the existing calendar-provider wiring (where `gprov`, `msConn`/`msFactory` are built), add a busy resolver and set it on services. Reuse the same `gconn`/`calendarConnector` (google) and `msConn` (microsoft) instances:
```go
var busyResolver application.BusyResolver
if calendarConnector != nil || msConn != nil {
	var grf *calendargoogle.ReaderFactory
	if calendarConnector != nil {
		grf = calendargoogle.NewReaderFactory(store, calendarConnector)
	}
	var mrf *calendarms.Factory
	if msConn != nil {
		mrf = calendarms.NewFactory(store, msConn)
	}
	busyResolver = calendarresolver.NewBusyResolver(store, grf, mrf)
}
services.Busy = busyResolver
```
Note the SAME typed-nil concern as 1b: `grf`/`mrf` are concrete pointers passed to `NewBusyResolver`'s interface params — a nil `*ReaderFactory` is a typed-nil that passes `r.google == nil`? NO: the resolver guards `r.google == nil`, and a typed-nil `*ReaderFactory` is NOT `== nil` as an interface. So pass untyped nil when unset: declare `var grf googleReaderFactory`-shaped interface vars OR guard before assigning. Simplest: only build+pass each factory when its connector is set, using interface-typed local vars set to nil otherwise (mirror 1b Task 3's anonymous-interface approach). The implementer must avoid the typed-nil trap exactly as in 1b. Verify `services.Busy` ends up nil (untyped) when neither connector is configured.

- [ ] **Step 7: Build/vet + commit**
```bash
env -u GOROOT go build ./... && env -u GOROOT go vet ./... && golangci-lint run --config ../../config/.golangci.yml ./...
gofmt -w internal/application/ports.go internal/application/services.go internal/infrastructure/calendar/resolver/busy.go internal/infrastructure/calendar/resolver/busy_test.go cmd/server/main.go
git add apps/backend/internal/application/ports.go apps/backend/internal/application/services.go \
        apps/backend/internal/infrastructure/calendar/resolver/busy.go apps/backend/internal/infrastructure/calendar/resolver/busy_test.go \
        apps/backend/cmd/server/main.go
git commit -m "feat(calendar): BusyResolver port + per-email reader impl + wiring"
```

---

### Task 3: `gatherExternalBusy` + engine integration (signature change)

**Files:**
- Create: `apps/backend/internal/application/availability.go`
- Modify: `apps/backend/internal/application/conflict.go` — `MeetingConflicts` + `FreeSlots` signatures + bodies; internal callers
- Modify: `apps/backend/internal/application/series_conflicts.go` — pass requester through
- Modify call sites: `apps/backend/internal/delivery/http/handlers/miniapp_write.go:~198`, `miniapp_read.go:~169`
- Modify interfaces + call sites: `apps/backend/internal/platform/checker/service.go` (interface :16 + call :149), `apps/backend/internal/platform/scheduler_agent/tools.go` (interface :27-28 + calls :151,:186)
- Test: `apps/backend/internal/application/availability_test.go`

**Interfaces:**
- New signatures (add `requesterEmail string` as the 2nd param):
  - `MeetingConflicts(ctx, requesterEmail string, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)`
  - `FreeSlots(ctx, requesterEmail string, emails []string, from, to time.Time, durMins int) ([]FreeSlot, error)`
- Produces: `(s *Services) gatherExternalBusy(ctx, requesterEmail string, emails []string, from, to time.Time) map[string][]meeting.Span` (best-effort, never errors).

- [ ] **Step 1: Write the failing test** — `availability_test.go` (`package application`):
```go
type fakeBusyResolver struct{ byEmail map[string]docalendar.BusyReader }

func (f fakeBusyResolver) ReaderFor(_ context.Context, email string) (docalendar.BusyReader, bool) {
	r, ok := f.byEmail[email]
	return r, ok
}

type stubReader struct {
	busy map[string][]docalendar.Interval
	err  error
}

func (s stubReader) BusyTimes(_ context.Context, emails []string, _ , _ time.Time) (map[string][]docalendar.Interval, error) {
	return s.busy, s.err
}

func TestGatherExternalBusy_UnionAndBestEffort(t *testing.T) {
	from := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	organizer := stubReader{busy: map[string][]docalendar.Interval{"a@x.com": {{Start: from, End: to}}}}
	ownB := stubReader{err: errors.New("boom")} // b's own reader fails -> skipped, no panic
	s := &Services{Busy: fakeBusyResolver{byEmail: map[string]docalendar.BusyReader{
		"org@x.com": organizer, "b@x.com": ownB,
	}}}
	got := s.gatherExternalBusy(context.Background(), "org@x.com", []string{"a@x.com", "b@x.com"}, from, to)
	if len(got["a@x.com"]) != 1 {
		t.Fatalf("expected a busy from organizer view, got %v", got)
	}
}

func TestGatherExternalBusy_NilResolver(t *testing.T) {
	s := &Services{}
	got := s.gatherExternalBusy(context.Background(), "org@x.com", []string{"a@x.com"}, time.Now(), time.Now().Add(time.Hour))
	if len(got) != 0 {
		t.Fatalf("expected empty with nil resolver, got %v", got)
	}
}
```

- [ ] **Step 2: Run; expect FAIL** — `env -u GOROOT go test ./internal/application/ -run TestGatherExternalBusy -v`

- [ ] **Step 3: Implement `gatherExternalBusy`** — `availability.go`:
```go
package application

import (
	"context"
	"sync"
	"time"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/platform/fanio"
)

func (s *Services) gatherExternalBusy(ctx context.Context, requesterEmail string, emails []string, from, to time.Time) map[string][]meeting.Span {
	out := map[string][]meeting.Span{}
	if s.Busy == nil || len(emails) == 0 {
		return out
	}
	var mu sync.Mutex
	add := func(busy map[string][]docalendarInterval) {
		mu.Lock()
		defer mu.Unlock()
		for email, ivs := range busy {
			for _, iv := range ivs {
				out[email] = append(out[email], meeting.Span{Start: iv.Start, End: iv.End})
			}
		}
	}
	if requesterEmail != "" {
		if reader, ok := s.Busy.ReaderFor(ctx, requesterEmail); ok {
			if busy, err := reader.BusyTimes(ctx, emails, from, to); err == nil {
				add(busy)
			} else {
				s.logExternalBusyFail(requesterEmail, err)
			}
		}
	}
	fanio.AllBestEffort(ctx, 4, len(emails), func(ctx context.Context, i int) {
		email := emails[i]
		reader, ok := s.Busy.ReaderFor(ctx, email)
		if !ok {
			return
		}
		busy, err := reader.BusyTimes(ctx, []string{email}, from, to)
		if err != nil {
			s.logExternalBusyFail(email, err)
			return
		}
		add(busy)
	})
	return dedupeSpans(out)
}
```
Use the package's existing `docalendar` alias for `docalendarInterval` — i.e. type the `add` param as `map[string][]docalendar.Interval` (import `docalendar "…/internal/domain/calendar"`; check how conflict.go already aliases it and match). Add helpers in the same file: `dedupeSpans(map[string][]meeting.Span) map[string][]meeting.Span` (sort by Start, drop exact duplicates) and `logExternalBusyFail(email string, err error)` (logs `external_busy_fetch_failed` with `zap.String("email_hash", hashEmail(email))` + `zap.Error(err)` via `s.Log` if non-nil; `hashEmail` = first 8 hex of sha256 — reuse `crypto.MaskToken`-style or a tiny local helper; NEVER log the raw email).

- [ ] **Step 4: Run; expect PASS** — `env -u GOROOT go test ./internal/application/ -run TestGatherExternalBusy -v`

- [ ] **Step 5: Integrate into `MeetingConflicts`** — change the signature to add `requesterEmail string` and, after the existing DB-conflict loop, before the final sort:
```go
ext := s.gatherExternalBusy(ctx, requesterEmail, emails, start, end)
for _, email := range emails {
	for _, sp := range ext[email] {
		if sp.End.After(start) && sp.Start.Before(end) {
			out = append(out, Conflict{Email: email, PersonName: s.personName(ctx, email), Start: sp.Start, End: sp.End})
		}
	}
}
```
(MeetingName left zero — privacy.) Keep the existing `sort.Slice`.

- [ ] **Step 6: Integrate into `FreeSlots`** — change the signature to add `requesterEmail string` and, after building `busy` from DB meetings:
```go
ext := s.gatherExternalBusy(ctx, requesterEmail, emails, from, to)
for _, spans := range ext {
	busy = append(busy, spans...)
}
```
(The per-day filter loop already intersects against the window.)

- [ ] **Step 7: Update internal callers + handlers + platform interfaces**
- `conflict.go:~120` `MeetingUpdateConflicts` → it resolves the organizer; pass that organizer email as `requesterEmail` to `MeetingConflicts`. (If it builds `emails` via `meetingEmails`, also capture the organizer email there.)
- `series_conflicts.go:28` → thread a `requesterEmail` (the series organizer; the function already has the meeting/organizer context — resolve it once and pass through).
- `delivery/http/handlers/miniapp_write.go:~198` → pass the caller's email (`botUserEmail(c)` — already used in this package) as `requesterEmail`.
- `delivery/http/handlers/miniapp_read.go:~169` (FreeSlots) → pass `botUserEmail(c)`.
- `platform/checker/service.go` — update the `backend` interface method `FreeSlots` to the new signature, and the call at :149 to pass the checker's acting user email if available in scope, else `""`.
- `platform/scheduler_agent/tools.go` — update the `Backend` interface `FreeSlots`/`MeetingConflicts` (:27-28) to the new signatures and the calls (:151,:186) to pass the agent's acting user email if available, else `""`.
Search the whole module for any remaining caller: `grep -rn "MeetingConflicts(\|FreeSlots(" apps/backend --include='*.go'` (skip `meeting.FreeSlots` in `domain/meeting` and `domain` decls — those are the domain helper, different).

- [ ] **Step 8: Add an integration test** — in `availability_test.go`, a `MeetingConflicts` test with a fake `Busy` resolver that injects an overlapping external interval for a participant → assert an external `Conflict` (empty `MeetingName`) appears; and a `FreeSlots` test where external busy shrinks a slot. Plus a regression case: `Services{}` with `Busy == nil` → output identical to the DB-only path (no panic, no extra conflicts). (These need a fake `Store`/`Repository` — reuse the application test fakes if present, or a minimal in-memory one; keep it focused.)

- [ ] **Step 9: Full verify** — `env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green; existing checker/agent/series/conflict tests still pass.

- [ ] **Step 10: gofmt + commit**
```bash
gofmt -w internal/application/availability.go internal/application/conflict.go internal/application/series_conflicts.go \
        internal/application/availability_test.go internal/delivery/http/handlers/miniapp_write.go internal/delivery/http/handlers/miniapp_read.go \
        internal/platform/checker/service.go internal/platform/scheduler_agent/tools.go
git add apps/backend/internal/application/availability.go apps/backend/internal/application/availability_test.go \
        apps/backend/internal/application/conflict.go apps/backend/internal/application/series_conflicts.go \
        apps/backend/internal/delivery/http/handlers/miniapp_write.go apps/backend/internal/delivery/http/handlers/miniapp_read.go \
        apps/backend/internal/platform/checker/service.go apps/backend/internal/platform/scheduler_agent/tools.go
git commit -m "feat(calendar): merge external calendar busy into conflicts + free-slots"
```

---

### Task 4: Whole-slice verification

**Files:** none (verification only)

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green. The Google reader, BusyResolver, and availability tests run + PASS (httptest/fakes, no Docker).
- [ ] **Step 2: Regression** — confirm the pre-existing conflict / free-slot / series-conflict / checker / scheduler_agent tests still pass (they exercise the DB-only path with `Busy == nil`).
- [ ] **Step 3: Frontend** — none changed by this slice; optionally confirm `pnpm --filter mini-app build` still green (no API shape change — conflict/free-slot response types are unchanged; external conflicts reuse the same `Conflict` shape with empty meeting name).
- [ ] **Step 4: Ops note (documented)** — record: Google FreeBusy needs the `calendar.readonly`/`calendar` scope (already requested at connect in 1a); MS getSchedule needs `Calendars.Read` (covered by `Calendars.ReadWrite` from 1b). No real-calendar test (deferred to WS4 E2E).
- [ ] **Step 5: Tree clean** — verify HEAD; `git status` shows no stray staged files; user parallel WIP untouched.

---

## Notes for the executor

- **Best-effort is the invariant:** `gatherExternalBusy` must never return an error or panic; a failed/slow reader is logged and skipped. The conflict/free-slot query degrades to DB-only.
- **Typed-nil trap** (Task 2 wiring): pass untyped-nil factories to `NewBusyResolver` when a provider is unconfigured, so `r.google/r.ms == nil` guards work (same lesson as 1b Task 3).
- **Signature blast radius** (Task 3): `requesterEmail` is added to two `Services` methods + the `checker`/`scheduler_agent` interface copies; pass `""` where a requester isn't in scope (degrades to per-participant-own only). Keep the build green by changing all call sites in the one commit.
- **Privacy:** external conflicts never carry a meeting name. No tokens/raw-email in logs (hash the email).
- **Deferred:** onboarding (2a), booking links (3), FE "external busy" label, connection-lookup batching.
