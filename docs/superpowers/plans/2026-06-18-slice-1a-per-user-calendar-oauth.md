# Slice 1a — Per-user Calendar OAuth + Token Vault Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a user connect their own Google calendar via OAuth (tokens persisted, encrypted, auto-refreshing) so the meetings they organize are created on their own calendar, with the existing corporate service-account path as fallback — exposed on both the web dashboard and the Telegram mini-app.

**Architecture:** A `calendar_connections` vault (keyed by `(email, provider)`) plus a one-time `calendar_oauth_states` store back a surface-agnostic OAuth connect flow (web session **or** TMA JWT initiates; a public callback completes it; email travels in the server-side state row, not the URL). The Google `CalendarProvider.For` becomes organizer-aware: a connected organizer's email resolves to a per-user `*calendar.Service` built from a self-persisting `oauth2.TokenSource`; otherwise it falls through to the unchanged SA path.

**Tech Stack:** Go 1.26 (Fiber, pgx v5, goose migrations, `golang.org/x/oauth2` + `google.golang.org/api/calendar/v3`, `coreos/go-oidc`), testcontainers-go + httptest for tests; React Router v7 / shadcn / Tailwind v4 frontends (`apps/admin`, `apps/mini-app`).

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...` from there. Spec: `docs/superpowers/specs/2026-06-18-slice-1a-per-user-calendar-oauth-design.md`.
- Clean Architecture + depguard: `application` imports ZERO `internal/infrastructure`; calendar OAuth is an infrastructure adapter behind an `application` port (mirror the existing `SSOProvider` pattern). `domain` stays free of Fiber/pgx/oauth2.
- **No code comments** in new Go or TS files (repo convention; relaxed doc-lint).
- Tests: pure logic + adapters unit-tested; persistence via testcontainers (`package postgres_test`, boots a container or skips when Docker absent — mirror `internal/testsupport/pgtest`). `go test -race ./...` + `golangci-lint run ./...` must stay green.
- Logging (zap): no tokens, `code`, `state`, `verifier`, or raw email in logs; use `crypto.MaskToken` / an email hash. Stable snake_case messages.
- Frontend: files ≤300 lines, no emoji (lucide/SVG only), no code comments; i18n keys added to **all three** dictionaries (en/ru/kk) — key parity is compile-enforced. Match each file's existing one-line/style; never run repo-wide `prettier --write` (reflows unrelated files) — format only your own additions with the per-app `config/prettier.config.mjs`.
- Work on `main`; never `git add -A` (stage explicit paths); verify `HEAD` between tasks. Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- Provider string constant is the literal `"google"` (Microsoft is slice 1b).

---

### Task 1: `calendar_connections` vault (migration + model + repo)

**Files:**
- Create: `apps/backend/migrations/20260618120000_calendar_connections.sql`
- Modify: `apps/backend/internal/application/model/` — add `calendar_connection.go`
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/` — add `calendar_connection_repo.go`
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/types.go` (or wherever `model` re-exports live) — re-export alias if the package follows the existing `postgres.X = model.X` pattern
- Test: `apps/backend/internal/infrastructure/persistence/postgres/calendar_connection_repo_test.go`

**Interfaces:**
- Produces:
  - `model.CalendarConnection{ Email string; Provider string; AccessToken string; RefreshToken string; Expiry time.Time; Scopes string; ConnectedAt time.Time; UpdatedAt time.Time }` — tokens are **plaintext in memory**; the repo encrypts on write / decrypts on read.
  - `(*postgres.Store).UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error`
  - `(*postgres.Store).GetCalendarConnection(ctx context.Context, email, provider string) (model.CalendarConnection, error)` — returns an error satisfying `model.IsNotFound` when absent.
  - `(*postgres.Store).ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error)`
  - `(*postgres.Store).DeleteCalendarConnection(ctx context.Context, email, provider string) error`
- Consumes: `(*crypto.TokenCipher).Encrypt(string)([]byte,error)` / `.Decrypt([]byte)(string,error)` (the `Store` already holds a cipher; confirm the field name and reuse it — if the `Store` does not already hold the cipher, add a `cipher *crypto.TokenCipher` field set in the store constructor used by `cmd/server/main.go`).

- [ ] **Step 1: Write the migration**

`apps/backend/migrations/20260618120000_calendar_connections.sql`:
```sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE TABLE calendar_connections (
    email             CITEXT      NOT NULL,
    provider          TEXT        NOT NULL,
    access_token_enc  BYTEA       NOT NULL,
    refresh_token_enc BYTEA       NOT NULL,
    expiry            TIMESTAMPTZ NOT NULL,
    scopes            TEXT        NOT NULL DEFAULT '',
    connected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (email, provider)
);

-- +goose Down
DROP TABLE calendar_connections;
```

- [ ] **Step 2: Add the model type**

`apps/backend/internal/application/model/calendar_connection.go`:
```go
package model

import "time"

type CalendarConnection struct {
	Email        string
	Provider     string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Scopes       string
	ConnectedAt  time.Time
	UpdatedAt    time.Time
}
```
If the `postgres` package re-exports model types via aliases (it does for `Meeting` etc.), add `type CalendarConnection = model.CalendarConnection` alongside the others.

- [ ] **Step 3: Write the failing repo test**

`apps/backend/internal/infrastructure/persistence/postgres/calendar_connection_repo_test.go` (`package postgres_test`, mirror the existing TestMain/`pg` harness in this directory):
```go
func TestCalendarConnection_UpsertGetDelete(t *testing.T) {
	db := newTestDB(t) // existing helper: boots container or t.Skip when Docker absent
	db.Truncate(t)
	store := db.Store(zap.NewNop())
	ctx := context.Background()

	conn := pg.CalendarConnection{
		Email: "Alice@Example.com", Provider: "google",
		AccessToken: "at-1", RefreshToken: "rt-1",
		Expiry: time.Now().Add(time.Hour).UTC().Truncate(time.Second), Scopes: "calendar.events",
	}
	if err := store.UpsertCalendarConnection(ctx, conn); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := store.GetCalendarConnection(ctx, "alice@example.com", "google")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccessToken != "at-1" || got.RefreshToken != "rt-1" {
		t.Fatalf("tokens not round-tripped: %+v", got)
	}
	if got.Scopes != "calendar.events" {
		t.Fatalf("scopes: %q", got.Scopes)
	}

	conn.AccessToken = "at-2"
	if err := store.UpsertCalendarConnection(ctx, conn); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	got, _ = store.GetCalendarConnection(ctx, "alice@example.com", "google")
	if got.AccessToken != "at-2" {
		t.Fatalf("upsert did not overwrite: %q", got.AccessToken)
	}

	list, err := store.ListCalendarConnections(ctx, "alice@example.com")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}

	if err := store.DeleteCalendarConnection(ctx, "alice@example.com", "google"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetCalendarConnection(ctx, "alice@example.com", "google"); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound after delete, got %v", err)
	}
}
```
(Confirm the exact harness helper name in this directory — earlier tasks used a `pgtest`-backed `TestMain`; reuse whatever `meeting_repo_test.go` uses to get a `*Store`.)

- [ ] **Step 4: Run it; expect FAIL (methods undefined)**

Run: `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestCalendarConnection -v`
Expected: compile error / FAIL — `UpsertCalendarConnection` undefined.

- [ ] **Step 5: Implement the repo**

`apps/backend/internal/infrastructure/persistence/postgres/calendar_connection_repo.go`:
```go
package postgres

import (
	"context"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error {
	atEnc, err := s.cipher.Encrypt(conn.AccessToken)
	if err != nil {
		return err
	}
	rtEnc, err := s.cipher.Encrypt(conn.RefreshToken)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO calendar_connections (email, provider, access_token_enc, refresh_token_enc, expiry, scopes, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (email, provider) DO UPDATE SET
			access_token_enc = EXCLUDED.access_token_enc,
			refresh_token_enc = EXCLUDED.refresh_token_enc,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			updated_at = now()`,
		conn.Email, conn.Provider, atEnc, rtEnc, conn.Expiry, conn.Scopes)
	return err
}

func (s *Store) GetCalendarConnection(ctx context.Context, email, provider string) (model.CalendarConnection, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT email, provider, access_token_enc, refresh_token_enc, expiry, scopes, connected_at, updated_at
		FROM calendar_connections WHERE email = $1 AND provider = $2`, email, provider)
	return scanCalendarConnection(s, row)
}

func (s *Store) ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT email, provider, access_token_enc, refresh_token_enc, expiry, scopes, connected_at, updated_at
		FROM calendar_connections WHERE email = $1 ORDER BY provider`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.CalendarConnection
	for rows.Next() {
		conn, err := scanCalendarConnection(s, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, conn)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCalendarConnection(ctx context.Context, email, provider string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM calendar_connections WHERE email = $1 AND provider = $2`, email, provider)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanCalendarConnection(s *Store, row rowScanner) (model.CalendarConnection, error) {
	var (
		conn         model.CalendarConnection
		atEnc, rtEnc []byte
		connectedAt  time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(&conn.Email, &conn.Provider, &atEnc, &rtEnc, &conn.Expiry, &conn.Scopes, &connectedAt, &updatedAt); err != nil {
		return model.CalendarConnection{}, err
	}
	at, err := s.cipher.Decrypt(atEnc)
	if err != nil {
		return model.CalendarConnection{}, err
	}
	rt, err := s.cipher.Decrypt(rtEnc)
	if err != nil {
		return model.CalendarConnection{}, err
	}
	conn.AccessToken, conn.RefreshToken = at, rt
	conn.ConnectedAt, conn.UpdatedAt = connectedAt, updatedAt
	return conn, nil
}
```
Adapt `s.pool` / `s.cipher` to the real field names on `Store` (check `meeting_repo.go` for the pool accessor; check the store constructor for the cipher — wire it in if absent, updating `cmd/server/main.go`'s store construction).

- [ ] **Step 6: Run the test; expect PASS** (or SKIP with no Docker)

Run: `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestCalendarConnection -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/backend/migrations/20260618120000_calendar_connections.sql \
        apps/backend/internal/application/model/calendar_connection.go \
        apps/backend/internal/infrastructure/persistence/postgres/calendar_connection_repo.go \
        apps/backend/internal/infrastructure/persistence/postgres/calendar_connection_repo_test.go
# include types.go / store constructor if modified
git commit -m "feat(calendar): per-user calendar_connections vault + repo"
```

---

### Task 2: `calendar_oauth_states` one-time pending-state store

**Files:**
- Create: `apps/backend/migrations/20260618120100_calendar_oauth_states.sql`
- Modify: `apps/backend/internal/application/model/calendar_connection.go` — add `CalendarOAuthState`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/calendar_oauth_state_repo.go`
- Test: `apps/backend/internal/infrastructure/persistence/postgres/calendar_oauth_state_repo_test.go`

**Interfaces:**
- Produces:
  - `model.CalendarOAuthState{ State string; Email string; Provider string; Verifier string; ExpiresAt time.Time }`
  - `(*postgres.Store).CreateCalendarOAuthState(ctx, st model.CalendarOAuthState) error`
  - `(*postgres.Store).ConsumeCalendarOAuthState(ctx, state string) (model.CalendarOAuthState, error)` — atomically SELECT+DELETE; returns a `model.IsNotFound`-satisfying error when the row is missing **or** expired.

- [ ] **Step 1: Migration**

`apps/backend/migrations/20260618120100_calendar_oauth_states.sql`:
```sql
-- +goose Up
CREATE TABLE calendar_oauth_states (
    state      TEXT        PRIMARY KEY,
    email      CITEXT      NOT NULL,
    provider   TEXT        NOT NULL,
    verifier   TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE calendar_oauth_states;
```

- [ ] **Step 2: Model type** — append to `calendar_connection.go`:
```go
type CalendarOAuthState struct {
	State     string
	Email     string
	Provider  string
	Verifier  string
	ExpiresAt time.Time
}
```
(Add the `postgres.CalendarOAuthState = model.CalendarOAuthState` alias if the package aliases model types.)

- [ ] **Step 3: Failing test**

`calendar_oauth_state_repo_test.go` (`package postgres_test`):
```go
func TestCalendarOAuthState_CreateConsume(t *testing.T) {
	db := newTestDB(t)
	db.Truncate(t)
	store := db.Store(zap.NewNop())
	ctx := context.Background()

	st := pg.CalendarOAuthState{
		State: "st-123", Email: "bob@example.com", Provider: "google",
		Verifier: "ver-xyz", ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := store.CreateCalendarOAuthState(ctx, st); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.ConsumeCalendarOAuthState(ctx, "st-123")
	if err != nil || got.Email != "bob@example.com" || got.Verifier != "ver-xyz" {
		t.Fatalf("consume: %v %+v", err, got)
	}
	if _, err := store.ConsumeCalendarOAuthState(ctx, "st-123"); !model.IsNotFound(err) {
		t.Fatalf("second consume should be IsNotFound, got %v", err)
	}

	expired := pg.CalendarOAuthState{State: "st-exp", Email: "c@x.com", Provider: "google", Verifier: "v", ExpiresAt: time.Now().Add(-time.Minute)}
	_ = store.CreateCalendarOAuthState(ctx, expired)
	if _, err := store.ConsumeCalendarOAuthState(ctx, "st-exp"); !model.IsNotFound(err) {
		t.Fatalf("expired consume should be IsNotFound, got %v", err)
	}
}
```

- [ ] **Step 4: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestCalendarOAuthState -v`

- [ ] **Step 5: Implement**

`calendar_oauth_state_repo.go`:
```go
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateCalendarOAuthState(ctx context.Context, st model.CalendarOAuthState) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_oauth_states (state, email, provider, verifier, expires_at)
		VALUES ($1,$2,$3,$4,$5)`,
		st.State, st.Email, st.Provider, st.Verifier, st.ExpiresAt)
	return err
}

func (s *Store) ConsumeCalendarOAuthState(ctx context.Context, state string) (model.CalendarOAuthState, error) {
	var st model.CalendarOAuthState
	err := s.pool.QueryRow(ctx, `
		DELETE FROM calendar_oauth_states
		WHERE state = $1 AND expires_at > now()
		RETURNING state, email, provider, verifier, expires_at`, state).
		Scan(&st.State, &st.Email, &st.Provider, &st.Verifier, &st.ExpiresAt)
	if err != nil {
		return model.CalendarOAuthState{}, err
	}
	return st, nil
}

var _ = sql.ErrNoRows
var _ = time.Now
```
Note: `DELETE ... RETURNING` with no matching (missing **or** expired) row yields `pgx.ErrNoRows`, which wraps `sql.ErrNoRows`, so `model.IsNotFound` is true — no extra branch needed. Drop the two `var _ =` lines if those imports are otherwise used.

- [ ] **Step 6: Run; expect PASS** (or SKIP).

- [ ] **Step 7: Commit**
```bash
git add apps/backend/migrations/20260618120100_calendar_oauth_states.sql \
        apps/backend/internal/application/model/calendar_connection.go \
        apps/backend/internal/infrastructure/persistence/postgres/calendar_oauth_state_repo.go \
        apps/backend/internal/infrastructure/persistence/postgres/calendar_oauth_state_repo_test.go
git commit -m "feat(calendar): one-time calendar_oauth_states pending store"
```

---

### Task 3: `CalendarConnector` port + Google connector adapter + Services wiring

**Files:**
- Modify: `apps/backend/internal/application/ports.go` — add `CalendarToken` + `CalendarConnector`
- Modify: `apps/backend/internal/application/services.go` — add `connectors` field, `ConfigureCalendarConnectors`, `CalendarConnectorByName`
- Create: `apps/backend/internal/infrastructure/oauth/google/calendar_connector.go`
- Test: `apps/backend/internal/infrastructure/oauth/google/calendar_connector_test.go`
- Modify: `apps/backend/cmd/server/main.go` — construct + register the Google connector

**Interfaces:**
- Produces:
  - `application.CalendarToken{ AccessToken string; RefreshToken string; Expiry time.Time; Scopes string }`
  - `application.CalendarConnector` interface: `Name() string`, `AuthURL(state, challenge, redirectURL string) string`, `Exchange(ctx, code, verifier, redirectURL string) (CalendarToken, error)`
  - `(*Services).CalendarConnectorByName(name string) (CalendarConnector, bool)`
  - `(*Services).ConfigureCalendarConnectors(map[string]CalendarConnector)`
  - `google.NewCalendarConnector(clientID, clientSecret string) *CalendarConnector` (infrastructure)
- Consumes: `golang.org/x/oauth2`, `golang.org/x/oauth2/google`, `google.golang.org/api/calendar/v3` (scope constants `calendar.CalendarEventsScope`, `calendar.CalendarReadonlyScope`).

- [ ] **Step 1: Add the application port** — append to `internal/application/ports.go`:
```go
type CalendarToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Scopes       string
}

type CalendarConnector interface {
	Name() string
	AuthURL(state, pkceChallenge, redirectURL string) string
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (CalendarToken, error)
}
```
Add `"time"` to the import block.

- [ ] **Step 2: Services wiring** — in `internal/application/services.go`, add to the `Services` struct a field `connectors map[string]CalendarConnector` and:
```go
func (s *Services) ConfigureCalendarConnectors(c map[string]CalendarConnector) { s.connectors = c }

func (s *Services) CalendarConnectorByName(name string) (CalendarConnector, bool) {
	c, ok := s.connectors[name]
	return c, ok
}
```

- [ ] **Step 3: Failing connector test**

`internal/infrastructure/oauth/google/calendar_connector_test.go` (`package google`):
```go
func TestCalendarConnector_AuthURL(t *testing.T) {
	c := NewCalendarConnector("client-id", "secret")
	u := c.AuthURL("state-1", "challenge-1", "https://app.example.com/cb")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if q.Get("access_type") != "offline" {
		t.Errorf("access_type=%q want offline", q.Get("access_type"))
	}
	if q.Get("prompt") != "consent" {
		t.Errorf("prompt=%q want consent", q.Get("prompt"))
	}
	if q.Get("code_challenge") != "challenge-1" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("pkce missing: %v", q)
	}
	if q.Get("state") != "state-1" {
		t.Errorf("state=%q", q.Get("state"))
	}
	scope := q.Get("scope")
	if !strings.Contains(scope, "calendar.events") || !strings.Contains(scope, "calendar.readonly") {
		t.Errorf("scope=%q must include calendar.events + calendar.readonly", scope)
	}
}
```

- [ ] **Step 4: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/oauth/google/ -run TestCalendarConnector -v`

- [ ] **Step 5: Implement the connector**

`internal/infrastructure/oauth/google/calendar_connector.go`:
```go
package google

import (
	"context"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/luckyrogue/lead-cat/internal/application"
)

type CalendarConnector struct {
	clientID, clientSecret string
}

func NewCalendarConnector(clientID, clientSecret string) *CalendarConnector {
	return &CalendarConnector{clientID: clientID, clientSecret: clientSecret}
}

func (c *CalendarConnector) Name() string { return "google" }

func (c *CalendarConnector) cfg(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     googleoauth.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{calendar.CalendarEventsScope, calendar.CalendarReadonlyScope},
	}
}

func (c *CalendarConnector) AuthURL(state, challenge, redirectURL string) string {
	return c.cfg(redirectURL).AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (c *CalendarConnector) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.CalendarToken, error) {
	tok, err := c.cfg(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return application.CalendarToken{}, err
	}
	scopes, _ := tok.Extra("scope").(string)
	return application.CalendarToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
		Scopes:       scopes,
	}, nil
}

var _ application.CalendarConnector = (*CalendarConnector)(nil)
```
`oauth2.ApprovalForce` emits `prompt=consent` (this oauth2 version). If lint/build flags it, use `oauth2.SetAuthURLParam("prompt", "consent")` instead.

- [ ] **Step 6: Add an Exchange httptest** — append to the test file: stand up an `httptest.Server` that serves the token endpoint, point the connector's endpoint at it (extract `cfg` to accept an endpoint override in the test, or use a package-level `var tokenEndpoint = googleoauth.Endpoint` you can reassign in the test), POST returns `{"access_token":"at","refresh_token":"rt","expires_in":3600,"scope":"...calendar.events ...calendar.readonly","token_type":"Bearer"}`; assert `Exchange` returns those tokens and scopes. (Mirror the WS2d httptest pattern.)

- [ ] **Step 7: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/oauth/google/ -v`

- [ ] **Step 8: Wire in `cmd/server/main.go`** — where SSO providers are constructed/registered (find `ConfigureWebAuth` / `SSOProviderByName` wiring), construct the connector from the **same Google client id/secret** and register:
```go
connectors := map[string]application.CalendarConnector{}
if googleClientID != "" && googleClientSecret != "" {
	connectors["google"] = oauthgoogle.NewCalendarConnector(googleClientID, googleClientSecret)
}
services.ConfigureCalendarConnectors(connectors)
```
Use the existing variable names for the Google credentials (the SSO setup already reads them).

- [ ] **Step 9: Build + commit**
```bash
env -u GOROOT go build ./... && env -u GOROOT go vet ./...
git add apps/backend/internal/application/ports.go apps/backend/internal/application/services.go \
        apps/backend/internal/infrastructure/oauth/google/calendar_connector.go \
        apps/backend/internal/infrastructure/oauth/google/calendar_connector_test.go \
        apps/backend/cmd/server/main.go
git commit -m "feat(calendar): CalendarConnector port + Google adapter + wiring"
```

---

### Task 4: Organizer-aware `CalendarProvider.For` (per-user resolution + SA fallback)

**Files:**
- Modify: `apps/backend/internal/application/calendar.go` — `CalendarProvider.For` signature
- Modify: `apps/backend/internal/application/command/ports.go` — `CalendarProvider.For` signature
- Modify: `apps/backend/internal/infrastructure/calendar/google/provider.go` — connection store dep + per-user adapter + self-persisting source + SA fallback
- Create: `apps/backend/internal/infrastructure/calendar/google/usersource.go` — self-persisting token source
- Modify: `apps/backend/internal/infrastructure/calendar/stub/provider.go` — new signature
- Modify call sites: `apps/backend/internal/application/command/meetings.go` (3), `apps/backend/internal/application/series_edit.go` (6), `apps/backend/internal/application/participants.go` (1)
- Modify: `apps/backend/cmd/server/main.go` — `NewProvider` now takes the connection store
- Modify all fakes: `apps/backend/internal/application/command/fakes_test.go` (and any other `fakeCalProvider`)
- Test: `apps/backend/internal/infrastructure/calendar/google/provider_resolution_test.go`

**Interfaces:**
- Produces (both interfaces, identical shape):
  - `For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (CalendarService, error)` — `organizerEmail` may be empty (then SA path).
  - `google.NewProvider(connStore connectionStore, cfgStore configStore, cipher *crypto.TokenCipher, connector calendarConnector) *Provider` — where `connectionStore` exposes `GetCalendarConnection`/`UpsertCalendarConnection`, `configStore` is the existing SA-config interface, and `calendarConnector` builds the per-user oauth2 config (see below).
- Consumes: `model.CalendarConnection`, `model.IsNotFound`, the Task-3 `CalendarConnector` (for the oauth2 config used to build a refreshing token source).

> Resolution rule: if `organizerEmail != ""` and a `"google"` connection exists → build a per-user `*calendar.Service`; on any per-user build/refresh error → log `calendar_user_resolve_failed` and **fall through to the SA path** (creation must not hard-fail). If no connection (`IsNotFound`) → SA path silently.

- [ ] **Step 1: Write the failing resolution test**

`internal/infrastructure/calendar/google/provider_resolution_test.go` (`package google`):
```go
type fakeConnStore struct {
	conn   *model.CalendarConnection
	upserts int
}

func (f *fakeConnStore) GetCalendarConnection(_ context.Context, _, _ string) (model.CalendarConnection, error) {
	if f.conn == nil {
		return model.CalendarConnection{}, sql.ErrNoRows
	}
	return *f.conn, nil
}
func (f *fakeConnStore) UpsertCalendarConnection(_ context.Context, _ model.CalendarConnection) error {
	f.upserts++
	return nil
}

func TestFor_NoConnection_FallsBackToSA(t *testing.T) {
	// configStore returns empty SA → ErrNotConfigured proves the SA branch was taken.
	p := NewProvider(&fakeConnStore{conn: nil}, emptyConfigStore{}, testCipher(t), &CalendarConnector{})
	_, err := p.For(context.Background(), uuid.New(), "nobody@example.com")
	if !errors.Is(err, docalendar.ErrNotConfigured) {
		t.Fatalf("expected SA fallback (ErrNotConfigured), got %v", err)
	}
}

func TestFor_WithConnection_BuildsUserService(t *testing.T) {
	conn := &model.CalendarConnection{Email: "a@x.com", Provider: "google", AccessToken: "at", RefreshToken: "rt", Expiry: time.Now().Add(time.Hour)}
	p := NewProvider(&fakeConnStore{conn: conn}, emptyConfigStore{}, testCipher(t), &CalendarConnector{})
	svc, err := p.For(context.Background(), uuid.New(), "a@x.com")
	if err != nil || svc == nil {
		t.Fatalf("expected per-user service, got svc=%v err=%v", svc, err)
	}
}
```
Provide `emptyConfigStore` (returns `nil, "", "", nil` so the SA branch yields `ErrNotConfigured`) and `testCipher(t)` helpers in the test file (reuse the existing cipher test helper if one exists in this package).

- [ ] **Step 2: Run; expect FAIL** (signature mismatch / NewProvider arity).

- [ ] **Step 3: Self-persisting token source**

`internal/infrastructure/calendar/google/usersource.go`:
```go
package google

import "golang.org/x/oauth2"

type savingSource struct {
	base oauth2.TokenSource
	last string
	save func(*oauth2.Token)
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		s.last = tok.AccessToken
		if s.save != nil {
			s.save(tok)
		}
	}
	return tok, nil
}
```

- [ ] **Step 4: Update the Google provider**

In `provider.go`: add the connection-store interface and connector, extend `Provider` + `NewProvider`, and branch in `For`:
```go
type connectionStore interface {
	GetCalendarConnection(ctx context.Context, email, provider string) (model.CalendarConnection, error)
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
}

type calendarConnector interface {
	OAuthConfig(redirectURL string) *oauth2.Config
}
```
Add `OAuthConfig(redirectURL string) *oauth2.Config` to the Task-3 `CalendarConnector` (expose its private `cfg`; redirect URL is irrelevant for refresh, pass `""`). Extend:
```go
type Provider struct {
	conns     connectionStore
	store     configStore
	cipher    *crypto.TokenCipher
	connector calendarConnector
	cache     sync.Map
}

func NewProvider(conns connectionStore, store configStore, cipher *crypto.TokenCipher, connector calendarConnector) *Provider {
	return &Provider{conns: conns, store: store, cipher: cipher, connector: connector}
}

func (p *Provider) For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error) {
	if organizerEmail != "" && p.conns != nil && p.connector != nil {
		if svc, ok := p.userService(ctx, organizerEmail); ok {
			return svc, nil
		}
	}
	return p.saService(ctx, organizationID) // existing body extracted into saService
}

func (p *Provider) userService(ctx context.Context, email string) (docalendar.Service, bool) {
	conn, err := p.conns.GetCalendarConnection(ctx, email, "google")
	if err != nil {
		return nil, false // IsNotFound or unexpected: fall back to SA
	}
	cfg := p.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{
		AccessToken:  conn.AccessToken,
		RefreshToken: conn.RefreshToken,
		Expiry:       conn.Expiry,
	})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = p.conns.UpsertCalendarConnection(ctx, conn)
	}}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, src)))
	if err != nil {
		return nil, false
	}
	return &adapter{svc: svc, calendarID: "primary"}, true
}
```
Move the current `For` body (SA decrypt → JWT → service + cache) into `saService(ctx, organizationID)` unchanged.

- [ ] **Step 5: Change both port signatures + stub**

`application/calendar.go` and `application/command/ports.go`:
```go
For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (CalendarService, error) // / docalendar.Service in command
```
`calendar/stub/provider.go`:
```go
func (p *Provider) For(_ context.Context, _ uuid.UUID, _ string) (docalendar.Service, error) {
	return p.svc, nil
}
```

- [ ] **Step 6: Update all ~10 call sites** — pass the organizer email:
- `command/meetings.go:102` (create) → organizer is the create input's organizer email; `:192` / `:233` (update/cancel) → `m.Organizer` (the loaded meeting).
- `series_edit.go` (97,139,190,231,297,348) → the series/meeting `.Organizer`.
- `participants.go:152` → `meeting.Organizer`.
Each currently reads `...For(ctx, organizationID)`; change to `...For(ctx, organizationID, <organizerEmail>)`. Confirm each scope already has the meeting/organizer in hand (it does — these are event-mutating paths).

- [ ] **Step 7: Update fakes + cmd wiring**
- `command/fakes_test.go` `fakeCalProvider.For` → add the `string` param. Any other fake provider (search `func.*For(ctx`).
- `cmd/server/main.go:85` → `calProvider = calendargoogle.NewProvider(store, store, cipher, calendarConnectorForProvider)` — pass the store as both connection + config store, plus the Google connector (reuse the Task-3 connector instance; if nil because creds unset, pass a no-op or guard `organizerEmail` branch with the `p.connector != nil` check already added).

- [ ] **Step 8: Run resolution test + full build/vet/race**

Run:
```
env -u GOROOT go test ./internal/infrastructure/calendar/google/ -v
env -u GOROOT go build ./... && env -u GOROOT go vet ./...
env -u GOROOT go test -race ./...
```
Expected: PASS / EXIT 0.

- [ ] **Step 9: Commit**
```bash
git add apps/backend/internal/application/calendar.go apps/backend/internal/application/command/ports.go \
        apps/backend/internal/infrastructure/calendar/google/provider.go \
        apps/backend/internal/infrastructure/calendar/google/usersource.go \
        apps/backend/internal/infrastructure/calendar/google/provider_resolution_test.go \
        apps/backend/internal/infrastructure/calendar/stub/provider.go \
        apps/backend/internal/application/command/meetings.go apps/backend/internal/application/series_edit.go \
        apps/backend/internal/application/participants.go apps/backend/internal/application/command/fakes_test.go \
        apps/backend/internal/infrastructure/oauth/google/calendar_connector.go apps/backend/cmd/server/main.go
git commit -m "feat(calendar): organizer-aware For() — per-user calendar + SA fallback"
```

---

### Task 5: Connect/callback/status/disconnect HTTP handlers

**Files:**
- Create: `apps/backend/internal/delivery/http/handlers/calendar_connect.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — register routes under web + miniapp groups + public callback
- Modify: `apps/backend/internal/application/calendar_connect.go` (new) — `Services` orchestration methods (start/finish/list/disconnect) so handlers stay thin
- Test: `apps/backend/internal/delivery/http/handlers/calendar_connect_test.go`

**Interfaces:**
- Consumes: `CalendarConnectorByName`, `CreateCalendarOAuthState`/`ConsumeCalendarOAuthState`, `UpsertCalendarConnection`/`ListCalendarConnections`/`DeleteCalendarConnection`, `authweb.NewState`/`authweb.NewPKCE`, `a.App.AppBaseURL()`.
- Produces:
  - `POST .../calendar/connect/google/start` → `{"auth_url":"..."}`
  - `GET /api/calendar/connect/google/callback?code=&state=` (public) → HTML "connected" page
  - `GET .../calendar/connections` → `[{"provider":"google","connected":true,"email":"...","scopes":"..."}]`
  - `DELETE .../calendar/connections/google` → 204
- Application methods (in `internal/application/calendar_connect.go`):
  - `(s *Services) StartCalendarConnect(ctx, email, provider, redirectURL string) (authURL string, err error)`
  - `(s *Services) FinishCalendarConnect(ctx, state, code, redirectURL string) error`
  - `(s *Services) ListCalendarConnections(ctx, email string) ([]CalendarConnectionView, error)` with `CalendarConnectionView{Provider string; Connected bool; Email string; Scopes string}`
  - `(s *Services) DisconnectCalendar(ctx, email, provider string) error`

- [ ] **Step 1: Application orchestration**

`internal/application/calendar_connect.go`:
```go
package application

import (
	"context"
	"errors"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

var ErrUnknownConnector = errors.New("unknown_connector")

type CalendarConnectionView struct {
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
	Scopes    string `json:"scopes"`
}

func (s *Services) StartCalendarConnect(ctx context.Context, email, provider, redirectURL string) (string, error) {
	conn, ok := s.CalendarConnectorByName(provider)
	if !ok {
		return "", ErrUnknownConnector
	}
	state, err := authweb.NewState(nil)
	if err != nil {
		return "", err
	}
	verifier, challenge, err := authweb.NewPKCE(nil)
	if err != nil {
		return "", err
	}
	if err := s.Store.CreateCalendarOAuthState(ctx, model.CalendarOAuthState{
		State: state, Email: email, Provider: provider, Verifier: verifier,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return "", err
	}
	return conn.AuthURL(state, challenge, redirectURL), nil
}

func (s *Services) FinishCalendarConnect(ctx context.Context, state, code, redirectURL string) error {
	pending, err := s.Store.ConsumeCalendarOAuthState(ctx, state)
	if err != nil {
		return err
	}
	conn, ok := s.CalendarConnectorByName(pending.Provider)
	if !ok {
		return ErrUnknownConnector
	}
	tok, err := conn.Exchange(ctx, code, pending.Verifier, redirectURL)
	if err != nil {
		return err
	}
	return s.Store.UpsertCalendarConnection(ctx, model.CalendarConnection{
		Email: pending.Email, Provider: pending.Provider,
		AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		Expiry: tok.Expiry, Scopes: tok.Scopes,
	})
}

func (s *Services) ListCalendarConnections(ctx context.Context, email string) ([]CalendarConnectionView, error) {
	rows, err := s.Store.ListCalendarConnections(ctx, email)
	if err != nil {
		return nil, err
	}
	out := []CalendarConnectionView{}
	for _, r := range rows {
		out = append(out, CalendarConnectionView{Provider: r.Provider, Connected: true, Email: r.Email, Scopes: r.Scopes})
	}
	return out, nil
}

func (s *Services) DisconnectCalendar(ctx context.Context, email, provider string) error {
	return s.Store.DeleteCalendarConnection(ctx, email, provider)
}
```
Add the four new repo methods to the `application` `Store`/`Repository` port interface so `s.Store.*` compiles (find where `Repository` is declared and add `CreateCalendarOAuthState`, `ConsumeCalendarOAuthState`, `UpsertCalendarConnection`, `ListCalendarConnections`, `DeleteCalendarConnection` with the Task-1/2 signatures).

- [ ] **Step 2: HTTP handlers**

`internal/delivery/http/handlers/calendar_connect.go`:
```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func callerEmail(c *fiber.Ctx) (string, bool) {
	if u, ok := c.Locals("web_user").(model.PlatformUser); ok && u.Email != "" {
		return u.Email, true
	}
	if bu, ok := c.Locals("bot_user").(model.BotUser); ok && bu.Email != "" {
		return bu.Email, true
	}
	return "", false
}

func (a *API) calendarCallbackURL(provider string) string {
	return a.App.AppBaseURL() + "/api/calendar/connect/" + provider + "/callback"
}

func (a *API) CalendarConnectStart(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	provider := c.Params("provider")
	authURL, err := a.App.StartCalendarConnect(c.UserContext(), email, provider, a.calendarCallbackURL(provider))
	if errors.Is(err, application.ErrUnknownConnector) {
		return fiber.NewError(fiber.StatusNotFound, "unknown_provider")
	}
	if err != nil {
		a.Log.Error("calendar_connect_start_failed", zap.String("provider", provider), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "start_failed")
	}
	return c.JSON(fiber.Map{"auth_url": authURL})
}

func (a *API) CalendarConnectCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	state, code := c.Query("state"), c.Query("code")
	if state == "" || code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request")
	}
	if err := a.App.FinishCalendarConnect(c.UserContext(), state, code, a.calendarCallbackURL(provider)); err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "bad_state")
		}
		a.Log.Warn("calendar_connect_callback_failed", zap.String("provider", provider), zap.Error(err))
		return fiber.NewError(fiber.StatusBadRequest, "connect_failed")
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(`<!doctype html><meta charset=utf-8><title>Connected</title><body style="font-family:system-ui;text-align:center;padding:3rem"><h2>Calendar connected</h2><p>You can close this tab.</p></body>`)
}

func (a *API) CalendarConnectionsList(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	views, err := a.App.ListCalendarConnections(c.UserContext(), email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) CalendarDisconnect(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if err := a.App.DisconnectCalendar(c.UserContext(), email, c.Params("provider")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "disconnect_failed")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 3: Register routes** in `internal/delivery/http/app.go` (match existing group helpers/middleware):
- Public: `app.Get("/api/calendar/connect/:provider/callback", a.CalendarConnectCallback)`.
- Web group (webAuth + CSRF, where other `/api/auth/web/me*` routes live): `start` (POST), `connections` (GET), `connections/:provider` (DELETE) under `/api/calendar/...`.
- Mini-app group (miniapp auth, where `/api/miniapp/*` lives): the same three under `/api/miniapp/calendar/...` pointing at the same handler methods.
The callback URL passed to Google is always the public `/api/calendar/connect/:provider/callback` regardless of initiating surface.

- [ ] **Step 4: Failing handler test**

`calendar_connect_test.go` — build a Fiber app with the handler, inject `web_user` via a test middleware, stub `a.App` with a fake connector + an in-memory state/connection store (or use the real `Services` over a testcontainers store + a fake connector). Assert:
  - `POST /api/calendar/connect/google/start` returns 200 with a non-empty `auth_url` containing the state, and a pending row was written.
  - `GET /callback?state=<that>&code=x` (fake connector returns a token) → 200 HTML, connection persisted.
  - `GET /callback?state=replay&code=x` with no row → 400 `bad_state`.
Use the WS2d httptest style; a fake `CalendarConnector` whose `Exchange` returns a canned `application.CalendarToken` avoids real Google.

- [ ] **Step 5: Run; expect PASS** — `env -u GOROOT go test ./internal/delivery/http/... -run Calendar -v`

- [ ] **Step 6: Build/vet/lint + commit**
```bash
env -u GOROOT go build ./... && env -u GOROOT go vet ./... && (cd apps/backend && golangci-lint run ./...)
git add apps/backend/internal/application/calendar_connect.go \
        apps/backend/internal/delivery/http/handlers/calendar_connect.go \
        apps/backend/internal/delivery/http/app.go \
        apps/backend/internal/delivery/http/handlers/calendar_connect_test.go
# include the Repository port file you extended
git commit -m "feat(calendar): connect/callback/status/disconnect endpoints (web + miniapp)"
```

---

### Task 6: Web (admin) — Calendar connection UI + i18n

**Files:**
- Create: `apps/admin/app/entities/calendar-connection/{types.ts,api.ts,queries.ts}`
- Create: `apps/admin/app/features/calendar-connections/components/calendar-connections-card.tsx`
- Modify: the Settings page (`apps/admin/app/features/.../pages/settings-page.tsx` — the one added in Phase 4a) to render the card
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — add `settings.calendars.*` keys

**Interfaces:**
- Consumes: `@leadcat/api-client` (or the existing fetch wrapper used by other admin entities) hitting `POST /api/calendar/connect/google/start`, `GET /api/calendar/connections`, `DELETE /api/calendar/connections/google`.
- Produces: `useCalendarConnections()` query + `useStartConnect()` / `useDisconnect()` mutations; `<CalendarConnectionsCard/>`.

- [ ] **Step 1: Entity types** — `types.ts`:
```ts
export type CalendarConnection = {
  provider: "google" | "microsoft"
  connected: boolean
  email: string
  scopes: string
}
```

- [ ] **Step 2: api.ts** (mirror an existing admin entity api file's client + base):
```ts
import { api } from "~/shared/api/client"
import type { CalendarConnection } from "./types"

export async function listConnections(): Promise<CalendarConnection[]> {
  return api.get("/api/calendar/connections")
}
export async function startConnect(provider: string): Promise<{ auth_url: string }> {
  return api.post(`/api/calendar/connect/${provider}/start`, {})
}
export async function disconnect(provider: string): Promise<void> {
  await api.del(`/api/calendar/connections/${provider}`)
}
```
Match the real `~/shared/api` helper names (`get/post/del`) used by `entities/meeting/api.ts`.

- [ ] **Step 3: queries.ts** — TanStack Query hooks mirroring `entities/meeting/queries.ts` (`useQuery` for `listConnections`, `useMutation` for start [then `window.location.href = auth_url`] and disconnect [invalidate the list]).

- [ ] **Step 4: Card component** — `calendar-connections-card.tsx` using `@leadcat/ui` `Card`/`Button` + `useT()`:
```tsx
export function CalendarConnectionsCard() {
  const t = useT()
  const { data = [] } = useCalendarConnections()
  const start = useStartConnect()
  const disconnect = useDisconnect()
  const google = data.find((c) => c.provider === "google")
  return (
    <Card>
      {/* header: t("settings.calendars.title") / t("settings.calendars.subtitle") */}
      {google?.connected ? (
        <Button variant="outline" onClick={() => disconnect.mutate("google")}>
          {t("settings.calendars.disconnect")}
        </Button>
      ) : (
        <Button onClick={() => start.mutate("google")}>{t("settings.calendars.connectGoogle")}</Button>
      )}
    </Card>
  )
}
```
(No comments in the real file; ≤300 lines; lucide icon for the calendar.)

- [ ] **Step 5: Mount the card** in the Settings page next to the timezone/language controls.

- [ ] **Step 6: i18n** — add to en/ru/kk (formal "вы" for admin):
```
settings.calendars.title          EN "Calendars"            RU "Календари"            KK "Күнтізбелер"
settings.calendars.subtitle       EN "Connect your own calendar so meetings you organize land on it."
settings.calendars.connectGoogle  EN "Connect Google"       RU "Подключить Google"    KK "Google қосу"
settings.calendars.disconnect     EN "Disconnect"           RU "Отключить"            KK "Ажырату"
settings.calendars.connected      EN "Connected as {email}"
```
(Provide complete RU/KK strings for every key — key parity is compile-enforced.)

- [ ] **Step 7: Verify + commit**
```bash
pnpm --filter @leadcat/admin typecheck && pnpm --filter @leadcat/admin lint && pnpm --filter @leadcat/admin build
git add apps/admin/app/entities/calendar-connection apps/admin/app/features/calendar-connections \
        apps/admin/app/shared/i18n/dictionaries apps/admin/app/features/*/pages/settings-page.tsx
git commit -m "feat(admin): calendar connection settings card + i18n"
```
(Confirm the admin pnpm filter name from `apps/admin/package.json`.)

---

### Task 7: Mini-app — Connected calendar row + i18n

**Files:**
- Create: `apps/mini-app/app/entities/calendar-connection/{types.ts,api.ts,queries.ts}` (same shapes as Task 6, mini-app's api wrapper, base path `/api/miniapp/calendar/...`)
- Create: `apps/mini-app/app/features/profile/components/calendar-connection-row.tsx`
- Modify: the Profile page to render the row
- Modify: `apps/mini-app/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — `profile.calendar.*` (informal "ты")

**Interfaces:**
- Consumes: `POST /api/miniapp/calendar/connect/google/start`, `GET /api/miniapp/calendar/connections`, `DELETE /api/miniapp/calendar/connections/google`; `window.Telegram.WebApp.openLink`.
- Produces: `<CalendarConnectionRow/>` + the same query/mutation hooks.

- [ ] **Step 1–3:** entity types/api/queries mirroring Task 6 but with the mini-app api wrapper and `/api/miniapp/calendar/...` paths.

- [ ] **Step 4: Row component** — on Connect: call `startConnect("google")` then `window.Telegram?.WebApp?.openLink(res.auth_url)` (external browser); re-query connections on `window` `focus` (add a `useEffect` that calls `refetch` on focus) so the row flips to "Connected" when the user returns. Disconnect inline.
```tsx
export function CalendarConnectionRow() {
  const t = useT()
  const { data = [], refetch } = useCalendarConnections()
  const start = useStartConnect()
  const disconnect = useDisconnect()
  useEffect(() => {
    const onFocus = () => refetch()
    window.addEventListener("focus", onFocus)
    return () => window.removeEventListener("focus", onFocus)
  }, [refetch])
  const google = data.find((c) => c.provider === "google")
  const connect = async () => {
    const res = await start.mutateAsync("google")
    window.Telegram?.WebApp?.openLink?.(res.auth_url)
  }
  // render row: connected → email + Disconnect; else Connect button
}
```

- [ ] **Step 5: Mount** the row in the Profile page (near the timezone/language preferences).

- [ ] **Step 6: i18n** — `profile.calendar.{title,connectGoogle,disconnect,connected}` in en/ru/kk (informal RU "ты"), full strings for all three.

- [ ] **Step 7: Verify + commit**
```bash
pnpm --filter @leadcat/mini-app typecheck && pnpm --filter @leadcat/mini-app lint && pnpm --filter @leadcat/mini-app build
git add apps/mini-app/app/entities/calendar-connection apps/mini-app/app/features/profile/components/calendar-connection-row.tsx \
        apps/mini-app/app/shared/i18n/dictionaries apps/mini-app/app/features/profile/pages/*.tsx
git commit -m "feat(mini-app): connected calendar row + i18n"
```
(Confirm the mini-app pnpm filter name; watch the known `entities/meeting/api.ts` >80-col reflow gotcha — keep additions additive, never reformat unrelated files.)

---

### Task 8: Whole-slice verification

**Files:** none (verification only)

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run ./...`. Expected: all green; persistence tests run (Docker up) or SKIP cleanly.
- [ ] **Step 2: Frontend** — `pnpm --filter @leadcat/admin typecheck lint build` and the same for `@leadcat/mini-app`. Expected: green; i18n key parity holds.
- [ ] **Step 3: OpenAPI** — if the repo gates OpenAPI, hand-edit `apps/backend/openapi/openapi.json` (compact inline style, do NOT prettier it) to add the four calendar routes + the `CalendarConnection` schema, then regenerate `@leadcat/api-client` (`pnpm --filter @leadcat/api-client generate`). If the new endpoints are consumed via the plain fetch wrapper (not the generated client), note that in the commit and skip regen.
- [ ] **Step 4: Manual smoke (documented, not automated)** — list the env/config prerequisite in the commit body: the Google OAuth client's consent screen must include `calendar.events` + `calendar.readonly` scopes; reuse the existing login client id/secret. No real-Google test here (deferred to WS4 E2E).
- [ ] **Step 5: Tree clean** — `git status` clean; `git log --oneline -8` shows the 1a commits. No `bin/` or stray generated artifacts staged.
- [ ] **Step 6: Final commit (if Step 3 changed files)**
```bash
git add apps/backend/openapi/openapi.json packages/api-client/...
git commit -m "chore(calendar): openapi + api-client for connection endpoints"
```

---

## Notes for the executor

- **Risk — Google consent screen / scope verification** is an ops prerequisite, not a code task; surface it (Task 8 Step 4) but don't block on it (tests use httptest).
- **`For` blast radius** (Task 4) is the highest-risk task: keep the build green by changing both interfaces + stub + all call sites + fakes in the *same* commit.
- **Stale connections:** 1a does not persist a `stale` flag; a failed refresh logs `calendar_user_resolve_failed` and falls back to the SA path. Reconnect is always available. (Deferred richer status to a later slice.)
- **Deferred to 1b/1c/2a:** Microsoft Graph, reading participants' calendars for unified availability, onboarding flow.
