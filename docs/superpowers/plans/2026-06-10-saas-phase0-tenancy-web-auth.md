# SaaS Phase 0 — Multi-tenancy + Web SSO foundation: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn single-tenant lead-cat into the multi-tenant foundation of a web SaaS: rename `workspaces`→`organizations`, drop the singleton, add web SSO (Google + Microsoft) + magic-link sign-in, server-side sessions, self-serve org onboarding + invites, and tenant-scoped access control.

**Architecture:** Go monolith (Fiber/pgx/goose) gains a new `/api/auth/web/*` + `/api/orgs/*` route surface guarded by cookie-based `WebAuth` + `RequireOrgMember` middleware, distinct from the existing Telegram `MiniAppAuth`. Auth providers sit behind `application.SSOProvider` / `application.EmailSender` ports with infra implementations. The React frontend gains a non-Telegram web shell (`/login`, `/onboarding`, dashboard) selected by a bootstrap surface detector. The Telegram path is parked and allowed to break this phase.

**Tech Stack:** Go 1.22 (Fiber v2, pgx/v5, goose migrations, zap), `golang.org/x/oauth2` + OIDC discovery, `net/smtp`, Postgres 15, React 18 + TanStack Router/Query + Vitest, Vite.

**Spec:** [`docs/superpowers/specs/2026-06-10-saas-phase0-tenancy-web-auth-design.md`](../specs/2026-06-10-saas-phase0-tenancy-web-auth-design.md).

**Branch:** `feat/saas-phase0-tenancy-web-auth` (created from `main` at `6ea054e`).

---

## Canonical names (use these EXACT identifiers in every task)

**DB tables/columns:** `organizations` (was `workspaces`), `organization_members` (was `workspace_members`), `organization_invites`, `magic_link_tokens`, `web_sessions`, `platform_users`. FK column everywhere: `organization_id` (was `workspace_id`). Member roles: `'owner'`, `'admin'`, `'member'`.

**Go types:** `postgres.Organization`, `postgres.OrganizationMember`, `postgres.OrganizationInvite`, `postgres.MagicLinkToken`, `postgres.WebSession`. (Existing `postgres.Workspace` is renamed to `postgres.Organization`.)

**Application ports:** `application.SSOProvider` (returns `application.SSOProfile`), `application.EmailSender`.

**Packages (new):** `internal/platform/authweb` (pure: PKCE, state, token hashing), `internal/infrastructure/oauth/google`, `internal/infrastructure/oauth/microsoft`, `internal/infrastructure/email/smtp`.

**Middleware:** `middleware.WebAuth` (sets `c.Locals("web_user")` → `postgres.PlatformUser`), `middleware.RequireOrgMember` (sets `c.Locals("org_member")` → `postgres.OrganizationMember`).

**Cookies:** `lc_session` (opaque session id), `lc_csrf` (CSRF double-submit), `lc_oauth_state`, `lc_pkce` (short-lived, auth-flow only).

**Routes:** auth under `/api/auth/web/*`; tenant management under `/api/orgs*`.

**Config keys:** `APP_BASE_URL`, `WEB_COOKIE_DOMAIN`, `WEB_SESSION_TTL_HOURS` (default 720), `GOOGLE_OAUTH_CLIENT_ID`/`_SECRET`/`_REDIRECT_URL`, `MICROSOFT_OAUTH_CLIENT_ID`/`_SECRET`/`_REDIRECT_URL`, `SMTP_HOST`/`_PORT`/`_USERNAME`/`_PASSWORD`/`_FROM`, `MAGIC_LINK_TTL_MINUTES` (default 15).

**Verification commands** (run from repo root):
- Backend tests: `cd backend && env -u GOROOT go test ./...`
- Backend build: `make build`
- Lint (golangci-lint incl. gofmt): `make lint`
- Migrations up/down: `make migrate` (and reversibility via goose down in a scratch DB)
- Frontend: `cd frontend && npm run build` and `npm run test` (Vitest)

> **Concurrency note** (see `concurrent-git-on-shared-branch` memory): the user may commit on this branch in parallel. Each task stages only its explicit paths (`git add <path>…`, never `-A`). Before each task, snapshot HEAD; after, verify HEAD advanced by exactly one commit you authored. Never touch `frontend/vite.config.ts` (ngrok WIP).

---

## File structure

```
backend/
├── migrations/
│   ├── 20260610140000_rename_workspaces_to_organizations.sql   [NEW]
│   ├── 20260610150000_org_auth_tables.sql                      [NEW]
│   └── 20260610160000_platform_users_auth_method.sql           [NEW]
├── internal/
│   ├── platform/
│   │   ├── authweb/
│   │   │   ├── pkce.go            [NEW]  pure PKCE + state + token hashing
│   │   │   └── pkce_test.go       [NEW]
│   │   └── config/config.go       [MODIFY] new keys
│   ├── application/
│   │   ├── ports.go               [MODIFY/NEW] SSOProvider, SSOProfile, EmailSender
│   │   ├── web_auth.go            [NEW]  identity provisioning, magic-link issue/verify
│   │   ├── web_auth_test.go       [NEW]
│   │   ├── org.go                 [NEW]  create org, slug, membership resolve, last-owner guard
│   │   ├── org_test.go            [NEW]
│   │   └── services.go            [MODIFY] wire new repos/ports
│   ├── infrastructure/
│   │   ├── oauth/
│   │   │   ├── google/provider.go      [NEW]
│   │   │   └── microsoft/provider.go   [NEW]
│   │   ├── email/smtp/sender.go        [NEW]
│   │   └── persistence/postgres/
│   │       ├── models.go               [MODIFY] rename Workspace→Organization, new structs
│   │       ├── organization_repo.go    [NEW/RENAMED from workspace_repo.go]
│   │       ├── web_session_repo.go     [NEW]
│   │       ├── magic_link_repo.go       [NEW]
│   │       ├── org_invite_repo.go      [NEW]
│   │       └── (all repos)             [MODIFY] table/column renames
│   └── delivery/http/
│       ├── app.go                  [MODIFY] wire web auth + org routes + config
│       ├── middleware/
│       │   ├── web_auth.go         [NEW]
│       │   ├── web_auth_test.go    [NEW]
│       │   ├── require_org_member.go      [NEW]
│       │   └── require_org_member_test.go [NEW]
│       └── handlers/
│           ├── web_auth.go         [NEW]  start/callback/magic/logout/me
│           ├── web_auth_test.go    [NEW]  integration test
│           └── orgs.go             [NEW]  create/list orgs, members, invites
└── (openapi/openapi.json + docs/openapi.json regenerated at the end)

frontend/
├── src/
│   ├── shared/web-auth/
│   │   ├── api.ts          [NEW] cookie-based fetchers (me, logout, magic request)
│   │   ├── queries.ts      [NEW] React Query hooks
│   │   └── context.tsx     [NEW] WebAuthProvider/gate
│   ├── shared/lib/surface.ts   [NEW] Telegram-vs-web detection
│   ├── features/web-login/
│   │   └── login-page.tsx  [NEW]
│   ├── features/onboarding/
│   │   └── onboarding-page.tsx [NEW]
│   ├── components/web-shell/
│   │   ├── web-layout.tsx  [NEW] shell + org switcher + logout
│   │   └── org-switcher.tsx [NEW]
│   ├── routes/
│   │   ├── login.tsx       [NEW]
│   │   ├── onboarding.tsx  [NEW]
│   │   └── _web.tsx        [NEW] web layout route
│   └── app/app-content.tsx [MODIFY] surface branch
└── (routeTree.gen.ts regenerated by the router plugin)
```

---

## Task 1: Rename `workspaces`→`organizations` (migration + Go identifiers)

This is a mechanical but cross-cutting rename. Keep the build red until every reference is updated, then commit once green. **No behavior change** — singleton still effectively in place until Task 2 drops the index. (We drop it here in the same migration.)

**Files:**
- Create: `backend/migrations/20260610140000_rename_workspaces_to_organizations.sql`
- Modify (rename identifiers): `backend/internal/infrastructure/persistence/postgres/models.go`, `workspace_repo.go`→`organization_repo.go`, `workspace_access.go`, `workspace_access_test.go`, `audit_repo.go`, `employee_repo.go`, `google_config_repo.go`, `meeting_repo.go`, `user_repo.go`; `backend/internal/application/{admin_workspace.go,calendar.go,conflict.go,google_verify.go,meeting_service.go,participants.go,series_edit.go,services.go}`; `backend/internal/delivery/http/handlers/{miniapp_admin.go,miniapp_write.go,log_helpers.go}`; `backend/internal/infrastructure/queue/asynq/queue.go`; `backend/internal/infrastructure/telegram/chat_sync.go`; `backend/internal/platform/{meeting_notifier/notifier.go,meetingedit/service.go,meetingedit/state.go,observability/log/context.go}`.

**Identifier mapping:**
- SQL: table `workspaces`→`organizations`, `workspace_members`→`organization_members`, column `workspace_id`→`organization_id` (in `meetings`, `employees`, `scenarios`, `scenario_runs`, `admin_audit_log` if present, `developer_vcs_map`, `pending_chat_links`), indexes/constraints renamed to match. Add `organizations.created_by_user_id UUID REFERENCES platform_users(id)` and `organizations.plan TEXT NOT NULL DEFAULT 'free'`. **Drop `workspaces_singleton_idx`.**
- Go: type `Workspace`→`Organization`, `WorkspaceMember`→`OrganizationMember`; method/func/var fragments `Workspace`→`Organization`, `workspaceID`→`organizationID`. Keep `ChatSyncer`'s parameter name change consistent. The `application.ChatSyncer` doc comment "into workspace_members" → "into organization_members".

- [ ] **Step 1: Write the rename migration**

Create `backend/migrations/20260610140000_rename_workspaces_to_organizations.sql`:

```sql
-- +goose Up
ALTER TABLE workspaces RENAME TO organizations;
ALTER TABLE workspace_members RENAME TO organization_members;
ALTER TABLE organization_members RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE developer_vcs_map RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE pending_chat_links RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE scenarios RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE scenario_runs RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE meetings RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE employees RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE admin_audit_log RENAME COLUMN target_id TO target_id;  -- no-op marker; audit uses target_id string

ALTER INDEX IF EXISTS workspaces_notify_chat_id_unique RENAME TO organizations_notify_chat_id_unique;

DROP INDEX IF EXISTS workspaces_singleton_idx;

ALTER TABLE organizations ADD COLUMN created_by_user_id UUID REFERENCES platform_users(id);
ALTER TABLE organizations ADD COLUMN plan TEXT NOT NULL DEFAULT 'free';

-- +goose Down
ALTER TABLE organizations DROP COLUMN plan;
ALTER TABLE organizations DROP COLUMN created_by_user_id;
CREATE UNIQUE INDEX workspaces_singleton_idx ON organizations ((true)) WHERE name = 'Lead Cat';
ALTER INDEX IF EXISTS organizations_notify_chat_id_unique RENAME TO workspaces_notify_chat_id_unique;
ALTER TABLE employees RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE meetings RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE scenario_runs RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE scenarios RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE pending_chat_links RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE developer_vcs_map RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE organization_members RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE organization_members RENAME TO workspace_members;
ALTER TABLE organizations RENAME TO workspaces;
```

> Before writing, confirm the exact set of `workspace_id` columns by running `grep -rn "workspace_id" backend/migrations/` and adjust the ALTERs to match reality (remove the no-op `admin_audit_log` line if that table has no such column; it uses `target_id`/`target_kind` strings). The migration must list every real column.

- [ ] **Step 2: Apply + verify reversibility on a scratch DB**

Run: `cd backend && env -u GOROOT go run ./cmd/migrate up` (or `make migrate`), then `... down` once, then `up` again.
Expected: up/down/up all succeed; `\d organizations` shows new columns and no singleton index.

- [ ] **Step 3: Rename Go SQL strings + identifiers**

Update every file in the mapping. In each repo, change SQL `FROM workspaces`/`workspace_members`/`workspace_id` to the new names, and rename Go identifiers per the mapping. Rename the file `workspace_repo.go`→`organization_repo.go` (`git mv`). Representative change in `organization_repo.go`:

```go
// before: func (s *Store) GetWorkspace(ctx context.Context, id uuid.UUID) (Workspace, error)
func (s *Store) GetOrganization(ctx context.Context, id uuid.UUID) (Organization, error) {
	const q = `SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id,
	       created_by_user_id, plan, chat_linked_at, created_at, updated_at
	  FROM organizations WHERE id = $1`
	// ... scan into Organization
}
```

Keep public method names used by `application` consistent (e.g. `Services.GetWorkspace` → `Services.GetOrganization`; update callers). The audit "EnsureLeadCatWorkspaceID" helper keeps working but rename to `EnsureDefaultOrganizationID` and update its SQL/callers.

- [ ] **Step 4: Build + existing tests green**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
Expected: compiles; `workspace_access_test.go` (renamed `organization_access_test.go`) passes against renamed identifiers.

- [ ] **Step 5: Grep verification — no stale identifiers**

Run: `grep -rn "workspace" backend/internal --include="*.go" | grep -vi "// "`
Expected: no remaining `workspaces`/`workspace_id`/`WorkspaceMember` (a few doc comments referencing history are acceptable; code identifiers must be gone). Then `make lint`.

- [ ] **Step 6: Commit**

```bash
git add backend/migrations/20260610140000_rename_workspaces_to_organizations.sql backend/internal
git commit -m "refactor(db): rename workspaces->organizations, drop singleton index"
```

---

## Task 2: New auth tables migration + models

**Files:**
- Create: `backend/migrations/20260610150000_org_auth_tables.sql`
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE organization_members ADD COLUMN invited_email TEXT;
-- normalize member roles to the SaaS vocabulary
UPDATE organization_members SET role = 'owner'  WHERE role = 'owner';
UPDATE organization_members SET role = 'admin'  WHERE role = 'admin';
UPDATE organization_members SET role = 'member' WHERE role NOT IN ('owner','admin');
ALTER TABLE organization_members ALTER COLUMN role SET DEFAULT 'member';

CREATE TABLE organization_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    token_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_by_user_id UUID REFERENCES platform_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX organization_invites_email_idx ON organization_invites (lower(email)) WHERE accepted_at IS NULL;

CREATE TABLE magic_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token_hash BYTEA NOT NULL,
    purpose TEXT NOT NULL DEFAULT 'login',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX magic_link_tokens_hash_idx ON magic_link_tokens (token_hash);

CREATE TABLE web_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT ''
);
CREATE INDEX web_sessions_user_idx ON web_sessions (user_id);

-- +goose Down
DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS magic_link_tokens;
DROP TABLE IF EXISTS organization_invites;
ALTER TABLE organization_members DROP COLUMN invited_email;
```

- [ ] **Step 2: Apply + reverse**

Run: `make migrate` then goose down/up on scratch DB. Expected: clean.

- [ ] **Step 3: Add model structs**

In `models.go`, add:

```go
type OrganizationInvite struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	Role           string
	TokenHash      []byte
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	CreatedByUserID *uuid.UUID
	CreatedAt      time.Time
}

type MagicLinkToken struct {
	ID         uuid.UUID
	Email      string
	TokenHash  []byte
	Purpose    string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

type WebSession struct {
	ID         uuid.UUID
	TokenHash  []byte
	UserID     uuid.UUID
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	UserAgent  string
	IP         string
}
```

Add `InvitedEmail *string` to `OrganizationMember`.

- [ ] **Step 4: Build**

Run: `cd backend && env -u GOROOT go build ./...` Expected: compiles.

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/20260610150000_org_auth_tables.sql backend/internal/infrastructure/persistence/postgres/models.go
git commit -m "feat(db): org invites, magic-link tokens, web sessions tables + models"
```

---

## Task 3: `platform_users.auth_method` + identity provisioning repo

**Files:**
- Create: `backend/migrations/20260610160000_platform_users_auth_method.sql`
- Modify: `backend/internal/infrastructure/persistence/postgres/user_repo.go`, `models.go`
- Test: `backend/internal/infrastructure/persistence/postgres/user_repo_test.go` (only if a pure helper is added; provisioning is I/O → build-verified)

- [ ] **Step 1: Migration**

```sql
-- +goose Up
ALTER TABLE platform_users ADD COLUMN auth_method TEXT NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE platform_users DROP COLUMN auth_method;
```

- [ ] **Step 2: Add `PlatformUser` model + `UpsertWebIdentity` repo method**

In `models.go` add (if not present):

```go
type PlatformUser struct {
	ID         uuid.UUID
	AuthSub    string
	Email      string
	TelegramID *int64
	AvatarURL  string
	AuthMethod string
	CreatedAt  time.Time
}
```

In `user_repo.go`:

```go
// UpsertWebIdentity provisions/refreshes the canonical account for a web sign-in,
// keyed by auth_sub ("email:<lower>"). Returns the platform_users row.
func (s *Store) UpsertWebIdentity(ctx context.Context, email, name, avatarURL, authMethod string) (PlatformUser, error) {
	sub := authsub.FromEmail(email) // reuse internal/platform/auth/sub.go
	const q = `
	INSERT INTO platform_users (auth_sub, email, avatar_url, auth_method)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (auth_sub) DO UPDATE SET
	  email = EXCLUDED.email,
	  avatar_url = CASE WHEN EXCLUDED.avatar_url <> '' THEN EXCLUDED.avatar_url ELSE platform_users.avatar_url END,
	  auth_method = EXCLUDED.auth_method
	RETURNING id, auth_sub, email, telegram_id, avatar_url, auth_method, created_at`
	var u PlatformUser
	err := s.pool.QueryRow(ctx, q, sub, email, avatarURL, authMethod).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &u.AvatarURL, &u.AuthMethod, &u.CreatedAt)
	return u, err
}
```

Confirm the existing `sub.go` package path/func (`grep -rn "func.*Sub" backend/internal/platform/auth/sub.go`) and use it; do not duplicate the builder.

- [ ] **Step 3: Build**

Run: `cd backend && env -u GOROOT go build ./...` Expected: compiles.

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/20260610160000_platform_users_auth_method.sql backend/internal/infrastructure/persistence/postgres/user_repo.go backend/internal/infrastructure/persistence/postgres/models.go
git commit -m "feat(db): platform_users.auth_method + UpsertWebIdentity"
```

---

## Task 4: `authweb` pure helpers (PKCE, state, token hashing)

**Files:**
- Create: `backend/internal/platform/authweb/pkce.go`
- Test: `backend/internal/platform/authweb/pkce_test.go`

- [ ] **Step 1: Write failing tests**

```go
package authweb

import (
	"strings"
	"testing"
)

func TestNewVerifierAndChallengeAreDeterministicPair(t *testing.T) {
	v, c, err := NewPKCE(func(b []byte) (int, error) { for i := range b { b[i] = byte(i) }; return len(b), nil })
	if err != nil { t.Fatal(err) }
	if len(v) < 43 || strings.ContainsAny(v, "+/=") { t.Fatalf("verifier not url-safe: %q", v) }
	if c == "" || c == v { t.Fatalf("challenge must be S256 of verifier, got %q", c) }
	if c2 := Challenge(v); c2 != c { t.Fatalf("Challenge(verifier) mismatch: %q vs %q", c2, c) }
}

func TestNewStateIsUrlSafeAndUnique(t *testing.T) {
	s1, _ := NewState(realRand)
	s2, _ := NewState(realRand)
	if s1 == s2 { t.Fatal("state collision") }
	if strings.ContainsAny(s1, "+/=") { t.Fatalf("state not url-safe: %q", s1) }
}

func TestHashTokenStable(t *testing.T) {
	if HashToken("abc") == nil { t.Fatal("nil hash") }
	if string(HashToken("abc")) != string(HashToken("abc")) { t.Fatal("unstable") }
	if string(HashToken("abc")) == string(HashToken("abd")) { t.Fatal("collision") }
}
```

(Add a small `realRand` reader helper in the test.)

- [ ] **Step 2: Run — fails to compile**

Run: `cd backend && env -u GOROOT go test ./internal/platform/authweb/...`
Expected: FAIL (undefined: NewPKCE, Challenge, NewState, HashToken).

- [ ] **Step 3: Implement**

```go
// Package authweb holds pure helpers for the web auth flow: PKCE, CSRF state,
// and one-way token hashing for storage. No I/O.
package authweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewPKCE returns (verifier, challenge). readFull defaults to crypto/rand.Read.
func NewPKCE(readFull func([]byte) (int, error)) (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if readFull == nil { readFull = rand.Read }
	if _, err = readFull(b); err != nil { return "", "", err }
	verifier = base64.RawURLEncoding.EncodeToString(b)
	return verifier, Challenge(verifier), nil
}

// Challenge is the S256 PKCE challenge of a verifier.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewState returns a url-safe random CSRF state token.
func NewState(readFull func([]byte) (int, error)) (string, error) {
	b := make([]byte, 32)
	if readFull == nil { readFull = rand.Read }
	if _, err := readFull(b); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 of a secret for at-rest storage (sessions,
// magic-link, invites). Compare hashes, never store raw tokens.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
```

- [ ] **Step 4: Run — passes**

Run: `cd backend && env -u GOROOT go test ./internal/platform/authweb/...` Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/authweb
git commit -m "feat(authweb): pure PKCE, CSRF state, token hashing helpers"
```

---

## Task 5: Application ports (`SSOProvider`, `SSOProfile`, `EmailSender`)

**Files:**
- Create/Modify: `backend/internal/application/ports.go`

- [ ] **Step 1: Define the ports**

```go
package application

import "context"

// SSOProfile is the normalized identity returned by any SSO provider.
type SSOProfile struct {
	Email     string
	Name      string
	AvatarURL string
	Subject   string // provider's stable subject id
	Provider  string // "google" | "microsoft"
}

// SSOProvider abstracts an OIDC authorization-code + PKCE flow.
type SSOProvider interface {
	Name() string
	// AuthURL builds the provider authorization URL.
	AuthURL(state, pkceChallenge, redirectURL string) string
	// Exchange swaps the callback code for a verified profile.
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (SSOProfile, error)
}

// EmailSender delivers transactional email (magic-link, invites).
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
```

- [ ] **Step 2: Build**

Run: `cd backend && env -u GOROOT go build ./...` Expected: compiles (no implementors yet).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application/ports.go
git commit -m "feat(application): SSOProvider + EmailSender ports"
```

---

## Task 6: Google + Microsoft OIDC provider implementations

I/O adapters → build-verified (no live network in tests).

**Files:**
- Create: `backend/internal/infrastructure/oauth/google/provider.go`, `backend/internal/infrastructure/oauth/microsoft/provider.go`

- [ ] **Step 1: Add dependency**

Run: `cd backend && env -u GOROOT go get golang.org/x/oauth2 github.com/coreos/go-oidc/v3/oidc`
Expected: `go.mod`/`go.sum` updated.

- [ ] **Step 2: Google provider**

```go
package google

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application"
)

type Provider struct {
	clientID, clientSecret string
	verifier               *oidc.IDTokenVerifier
	endpoint               oauth2.Endpoint
}

func New(ctx context.Context, clientID, clientSecret string) (*Provider, error) {
	p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil { return nil, err }
	return &Provider{
		clientID: clientID, clientSecret: clientSecret,
		verifier: p.Verifier(&oidc.Config{ClientID: clientID}),
		endpoint: p.Endpoint(),
	}, nil
}

func (p *Provider) Name() string { return "google" }

func (p *Provider) oauth(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: p.clientID, ClientSecret: p.clientSecret,
		Endpoint: p.endpoint, RedirectURL: redirectURL,
		Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

func (p *Provider) AuthURL(state, challenge, redirectURL string) string {
	return p.oauth(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (p *Provider) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.SSOProfile, error) {
	tok, err := p.oauth(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil { return application.SSOProfile{}, err }
	raw, _ := tok.Extra("id_token").(string)
	idt, err := p.verifier.Verify(ctx, raw)
	if err != nil { return application.SSOProfile{}, err }
	var c struct{ Email, Name, Picture, Sub string }
	if err := idt.Claims(&c); err != nil { return application.SSOProfile{}, err }
	return application.SSOProfile{Email: c.Email, Name: c.Name, AvatarURL: c.Picture, Subject: c.Sub, Provider: "google"}, nil
}
```

- [ ] **Step 3: Microsoft provider**

Same shape; use issuer `https://login.microsoftonline.com/common/v2.0` (personal + work accounts). Scopes `openid email profile`. Claims: `email` (fallback `preferred_username`), `name`, `sub`. `Name()` returns `"microsoft"`, `Provider: "microsoft"`.

```go
package microsoft
// ... identical structure to google, with:
//   p, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/common/v2.0")
//   verifier := p.Verifier(&oidc.Config{ClientID: clientID, SkipIssuerCheck: true}) // /common multiplexes issuers
// In Exchange, claim email may be empty → fall back to preferred_username.
```

- [ ] **Step 4: Build**

Run: `cd backend && env -u GOROOT go build ./... && make lint` Expected: compiles, lint clean.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/oauth backend/go.mod backend/go.sum
git commit -m "feat(oauth): Google + Microsoft OIDC providers (PKCE)"
```

---

## Task 7: SMTP `EmailSender` + magic-link issue/verify (application)

Magic-link token logic is pure-ish (hashing + expiry) → unit-tested via a fake repo; SMTP is I/O → build-verified.

**Files:**
- Create: `backend/internal/infrastructure/email/smtp/sender.go`
- Create: `backend/internal/application/web_auth.go`, `backend/internal/application/web_auth_test.go`
- Create: `backend/internal/infrastructure/persistence/postgres/magic_link_repo.go`

- [ ] **Step 1: SMTP sender (build-verified)**

```go
package smtp

import (
	"context"
	"fmt"
	"net/smtp"
)

type Sender struct{ host, port, user, pass, from string }

func New(host, port, user, pass, from string) *Sender {
	return &Sender{host, port, user, pass, from}
}

func (s *Sender) Send(_ context.Context, to, subject, htmlBody string) error {
	addr := s.host + ":" + s.port
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, htmlBody)
	var auth smtp.Auth
	if s.user != "" { auth = smtp.PlainAuth("", s.user, s.pass, s.host) }
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
```

- [ ] **Step 2: Magic-link repo (build-verified)**

In `magic_link_repo.go`: `InsertMagicLink(ctx, email string, tokenHash []byte, expiresAt time.Time) error`, and `ConsumeMagicLink(ctx, tokenHash []byte, now time.Time) (email string, ok bool, err error)` — a single `UPDATE magic_link_tokens SET consumed_at=now() WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at > $2 RETURNING email`.

- [ ] **Step 3: Write failing test for magic-link issue/verify**

```go
package application

import (
	"context"
	"testing"
	"time"
)

type fakeMagicRepo struct {
	inserted map[string][]byte // email -> hash
	expires  time.Time
}
func (f *fakeMagicRepo) InsertMagicLink(_ context.Context, email string, hash []byte, exp time.Time) error {
	f.inserted[email] = hash; f.expires = exp; return nil
}
func (f *fakeMagicRepo) ConsumeMagicLink(_ context.Context, hash []byte, _ time.Time) (string, bool, error) {
	for email, h := range f.inserted { if string(h) == string(hash) { return email, true, nil } }
	return "", false, nil
}
type fakeMailer struct{ lastTo, lastBody string }
func (m *fakeMailer) Send(_ context.Context, to, _ , body string) error { m.lastTo, m.lastBody = to, body; return nil }

func TestRequestMagicLinkSendsTokenAndStoresHash(t *testing.T) {
	repo := &fakeMagicRepo{inserted: map[string][]byte{}}
	mail := &fakeMailer{}
	svc := newMagicLinkService(repo, mail, "https://app.example.com", 15*time.Minute, fixedClock)
	if err := svc.RequestMagicLink(context.Background(), "u@yandex.ru"); err != nil { t.Fatal(err) }
	if mail.lastTo != "u@yandex.ru" { t.Fatalf("wrong recipient %q", mail.lastTo) }
	if repo.inserted["u@yandex.ru"] == nil { t.Fatal("hash not stored") }
	// raw token in the email must NOT equal the stored hash
}

func TestVerifyMagicLinkReturnsEmailForValidToken(t *testing.T) {
	// issue, capture raw token from the link in mail.lastBody, then Verify it
}
```

- [ ] **Step 4: Run — fails**

Run: `cd backend && env -u GOROOT go test ./internal/application/ -run MagicLink` Expected: FAIL (undefined).

- [ ] **Step 5: Implement `web_auth.go` magic-link service**

Use `authweb.NewState` to mint a raw token, `authweb.HashToken` to store/compare, build the link `APP_BASE_URL + "/api/auth/web/magic/verify?token=" + raw`, send via `EmailSender`. `Verify` hashes the incoming raw token and calls `ConsumeMagicLink`. Inject a `clock func() time.Time` for tests (`fixedClock` returns a constant; production passes `time.Now`).

- [ ] **Step 6: Run — passes; build; lint**

Run: `cd backend && env -u GOROOT go test ./internal/application/ -run MagicLink && go build ./... && cd .. && make lint`
Expected: PASS, compiles, clean.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/infrastructure/email backend/internal/infrastructure/persistence/postgres/magic_link_repo.go backend/internal/application/web_auth.go backend/internal/application/web_auth_test.go
git commit -m "feat(auth): SMTP sender + magic-link issue/verify service"
```

---

## Task 8: `web_sessions` repo + session create/resolve/revoke

I/O → build-verified; the only pure unit here is sliding-renewal decision (test it).

**Files:**
- Create: `backend/internal/infrastructure/persistence/postgres/web_session_repo.go`
- Create: `backend/internal/application/web_session.go`, `backend/internal/application/web_session_test.go`

- [ ] **Step 1: Repo methods (build-verified)**

```go
// CreateWebSession stores a hashed session token; returns the row.
func (s *Store) CreateWebSession(ctx context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time, ua, ip string) (WebSession, error)
// ResolveWebSession returns the live (unrevoked, unexpired) session by token hash.
func (s *Store) ResolveWebSession(ctx context.Context, tokenHash []byte, now time.Time) (WebSession, bool, error)
// TouchWebSession bumps last_seen_at and (optionally) extends expires_at.
func (s *Store) TouchWebSession(ctx context.Context, id uuid.UUID, lastSeen, expiresAt time.Time) error
// RevokeWebSession sets revoked_at.
func (s *Store) RevokeWebSession(ctx context.Context, tokenHash []byte, now time.Time) error
```

- [ ] **Step 2: Failing test for sliding-renewal policy**

```go
func TestShouldSlideWhenPastHalfLife(t *testing.T) {
	ttl := 720 * time.Hour
	created := mustTime("2026-06-01T00:00:00Z")
	// not past half-life -> no slide
	if shouldSlide(created, created.Add(ttl), created.Add(100*time.Hour), ttl) { t.Fatal("slid too early") }
	// past half-life -> slide
	if !shouldSlide(created, created.Add(ttl), created.Add(400*time.Hour), ttl) { t.Fatal("should slide") }
}
```

- [ ] **Step 3: Run — fails; implement `shouldSlide` + `WebSessionService` (Create/Resolve/Revoke wrapping repo + `authweb`); run — passes**

`Create` mints a raw token via `authweb.NewState`, returns it to the caller (for the cookie) and stores only `HashToken(raw)`. `Resolve` hashes the cookie value, calls repo, and slides via `TouchWebSession` when `shouldSlide`.

Run: `cd backend && env -u GOROOT go test ./internal/application/ -run Session && go build ./...` Expected: PASS, compiles.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/web_session_repo.go backend/internal/application/web_session.go backend/internal/application/web_session_test.go
git commit -m "feat(auth): server-side web session store + sliding renewal"
```

---

## Task 9: `WebAuth` middleware + CSRF

**Files:**
- Create: `backend/internal/delivery/http/middleware/web_auth.go`, `backend/internal/delivery/http/middleware/web_auth_test.go`

- [ ] **Step 1: Failing test — CSRF compare + missing cookie → 401**

```go
func TestCSRFTokenMatch(t *testing.T) {
	if !csrfMatches("abc", "abc") { t.Fatal("equal should match") }
	if csrfMatches("abc", "abd") { t.Fatal("different must not match") }
	if csrfMatches("", "") { t.Fatal("empty must not match") }
}
```

- [ ] **Step 2: Run — fails; implement**

`WebAuth(sessions WebSessionResolver)` returns a Fiber handler: read `lc_session` cookie → `sessions.Resolve` → load `platform_user` → `c.Locals("web_user", user)`; 401 if missing/invalid. On non-GET, require `X-CSRF-Token` header == `lc_csrf` cookie via constant-time `csrfMatches` (use `crypto/subtle.ConstantTimeCompare`, returns false on empty). Define a small `WebSessionResolver` interface locally (satisfied by `*application.Services`).

- [ ] **Step 3: Run — passes; build**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/middleware/ -run "CSRF|WebAuth" && go build ./...` Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/middleware/web_auth.go backend/internal/delivery/http/middleware/web_auth_test.go
git commit -m "feat(http): WebAuth cookie middleware + CSRF double-submit"
```

---

## Task 10: Org application logic (create, slug, membership, last-owner guard)

**Files:**
- Create: `backend/internal/application/org.go`, `backend/internal/application/org_test.go`
- Create: `backend/internal/infrastructure/persistence/postgres/organization_repo.go` additions (create org + members + invites repo methods), `org_invite_repo.go`

- [ ] **Step 1: Failing tests (pure logic)**

```go
func TestSlugify(t *testing.T) {
	cases := map[string]string{"Acme Inc.": "acme-inc", "  Hello  World ": "hello-world", "Ünïcødé": "unicode"}
	for in, want := range cases { if got := slugify(in); got != want { t.Fatalf("%q -> %q want %q", in, got, want) } }
}

func TestCanRemoveMemberBlocksLastOwner(t *testing.T) {
	members := []OrgMemberView{{Role: "owner"}, {Role: "member"}}
	if err := canDemoteOrRemove(members, 0, "remove"); err == nil { t.Fatal("removing the only owner must fail") }
	members = append(members, OrgMemberView{Role: "owner"})
	if err := canDemoteOrRemove(members, 0, "remove"); err != nil { t.Fatal("two owners -> removal ok") }
}

func TestResolveRolePrecedence(t *testing.T) {
	if !roleAtLeast("owner", "admin") { t.Fatal("owner >= admin") }
	if roleAtLeast("member", "admin") { t.Fatal("member < admin") }
}
```

- [ ] **Step 2: Run — fails; implement `slugify`, `canDemoteOrRemove`, `roleAtLeast`, `OrgMemberView` in `org.go`; run — passes**

Run: `cd backend && env -u GOROOT go test ./internal/application/ -run "Slug|Member|Role"` Expected: PASS.

- [ ] **Step 3: Repo methods (build-verified)**

`CreateOrganization(ctx, name, slug string, ownerUserID uuid.UUID) (Organization, error)` (insert org with `created_by_user_id`, then insert owner `organization_members` in one tx); `ListOrganizationsForUser(ctx, userID) ([]Organization, error)`; `GetOrgMember(ctx, orgID, userID) (OrganizationMember, bool, error)`; `ListOrgMembers(ctx, orgID) ([]OrganizationMember, error)`; `UpdateMemberRole`, `RemoveMember`. In `org_invite_repo.go`: `CreateInvite`, `ListInvites`, `DeleteInvite`, `AcceptInvitesForEmail(ctx, email string, userID uuid.UUID) (int, error)` (matches pending invites by `lower(email)`, inserts members, marks accepted — in a tx).

- [ ] **Step 4: Build + lint**

Run: `cd backend && env -u GOROOT go build ./... && cd .. && make lint` Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/application/org.go backend/internal/application/org_test.go backend/internal/infrastructure/persistence/postgres/organization_repo.go backend/internal/infrastructure/persistence/postgres/org_invite_repo.go
git commit -m "feat(org): create/slug/membership logic + invites repo"
```

---

## Task 11: `RequireOrgMember` middleware

**Files:**
- Create: `backend/internal/delivery/http/middleware/require_org_member.go`, `backend/internal/delivery/http/middleware/require_org_member_test.go`

- [ ] **Step 1: Failing test — role gate helper**

```go
func TestRequireRoleGate(t *testing.T) {
	if !memberMeetsRole("admin", "admin") { t.Fatal("admin meets admin") }
	if memberMeetsRole("member", "admin") { t.Fatal("member fails admin") }
}
```

- [ ] **Step 2: Run — fails; implement**

`RequireOrgMember(resolver)` reads `c.Locals("web_user")`, the `X-Org-Id` header (fallback: the user's single org), resolves membership via `resolver.GetOrgMember`; 403 if not a member; stores `c.Locals("org_member", m)`. Provide `RequireOrgRole(role)` that additionally checks `memberMeetsRole(m.Role, role)`.

- [ ] **Step 3: Run — passes; build**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/middleware/ -run OrgMember && go build ./...` Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/middleware/require_org_member.go backend/internal/delivery/http/middleware/require_org_member_test.go
git commit -m "feat(http): RequireOrgMember + role gate middleware"
```

---

## Task 12: Config keys + Services wiring

**Files:**
- Modify: `backend/internal/platform/config/config.go`, `backend/internal/application/services.go`, `backend/cmd/server/main.go` (provider/sender construction)

- [ ] **Step 1: Add config fields + parsing**

Add to `Config`: `AppBaseURL`, `WebCookieDomain`, `WebSessionTTL time.Duration`, `MagicLinkTTL time.Duration`, `GoogleOAuth {ClientID,Secret,RedirectURL}`, `MicrosoftOAuth {...}`, `SMTP {Host,Port,Username,Password,From}`. Parse via `envOr`; `WEB_SESSION_TTL_HOURS` default 720, `MAGIC_LINK_TTL_MINUTES` default 15. None are hard-required (web auth simply disabled if unset — log `web_auth_disabled_missing_config` at startup).

- [ ] **Step 2: Construct providers/sender in `main.go`**

Build `google.New`, `microsoft.New` (skip if client id empty), `smtp.New`; pass into `Services` (add fields `SSO map[string]SSOProvider`, `Email EmailSender`, plus the magic-link/session/org services). Guard: if a provider's config is empty, omit it from the map (the route returns 404/400 for that provider).

- [ ] **Step 3: Build + lint**

Run: `cd backend && env -u GOROOT go build ./... && cd .. && make lint` Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/platform/config/config.go backend/internal/application/services.go backend/cmd/server/main.go
git commit -m "feat(config): web auth + SSO + SMTP config and wiring"
```

---

## Task 13: Web auth HTTP handlers + routes (+ integration test)

**Files:**
- Create: `backend/internal/delivery/http/handlers/web_auth.go`, `backend/internal/delivery/http/handlers/web_auth_test.go`
- Modify: `backend/internal/delivery/http/app.go`

- [ ] **Step 1: Handlers**

`WebAuthStart` (`/api/auth/web/:provider/start`): mint state+PKCE, set `lc_oauth_state`+`lc_pkce` httpOnly short-lived cookies, 302 to `provider.AuthURL`. `WebAuthCallback`: validate `state` vs cookie, `Exchange`, `UpsertWebIdentity`, `AcceptInvitesForEmail`, create web session (set `lc_session`+`lc_csrf` cookies), 302 to `APP_BASE_URL + (hasOrg ? "/" : "/onboarding")`. `WebMagicRequest` (POST): `svc.RequestMagicLink`, always 204. `WebMagicVerify` (GET): `svc.VerifyMagicLink` → provision + session, 302. `WebLogout` (POST): revoke + clear cookies, 204. `WebMe` (GET, behind `WebAuth`): return account + `ListOrganizationsForUser`.

- [ ] **Step 2: Wire routes in `app.go`**

```go
webAuth := middleware.NewWebAuth(services)
web := app.Group("/api/auth/web")
web.Get("/:provider/start", api.WebAuthStart)
web.Get("/:provider/callback", api.WebAuthCallback)
web.Post("/magic/request", api.WebMagicRequest)
web.Get("/magic/verify", api.WebMagicVerify)
web.Post("/logout", webAuth.Middleware, api.WebLogout)
web.Get("/me", webAuth.Middleware, api.WebMe)
```

Update CORS `AllowHeaders` to include `X-CSRF-Token, X-Org-Id` and set `AllowCredentials: true` (required for cookies); ensure `AllowOrigins` is the explicit `APP_BASE_URL` (cannot be `*` with credentials).

- [ ] **Step 3: Integration test (≥1, per spec)**

```go
// Spins up the Fiber app with a test Store (scratch DB) + fake EmailSender.
// Flow: POST /api/auth/web/magic/request {email} -> 204; capture raw token from
// the fake mailer; GET /api/auth/web/magic/verify?token=... -> 302 + Set-Cookie lc_session;
// GET /api/auth/web/me with that cookie -> 200 with the email.
func TestMagicLinkRoundTripIssuesSession(t *testing.T) { /* ... */ }
```

Gate behind a build tag or `testing.Short()` skip if no `TEST_DATABASE_URL`, matching repo convention (check how existing integration-ish tests guard DB).

- [ ] **Step 4: Run + build + lint**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/... && go build ./... && cd .. && make lint` Expected: PASS/clean (integration test skips without DB).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/handlers/web_auth.go backend/internal/delivery/http/handlers/web_auth_test.go backend/internal/delivery/http/app.go
git commit -m "feat(http): web auth routes (SSO + magic-link + session) + integration test"
```

---

## Task 14: Org management HTTP handlers + routes

**Files:**
- Create: `backend/internal/delivery/http/handlers/orgs.go`
- Modify: `backend/internal/delivery/http/app.go`

- [ ] **Step 1: Handlers**

`CreateOrg` (POST `/api/orgs`, behind `WebAuth`): body `{name}` → `slugify` → `CreateOrganization(owner=web_user)` → 201 with org. `ListMyOrgs` (GET `/api/orgs`): `ListOrganizationsForUser`. `ListOrgMembers` (GET `/api/orgs/:id/members`, `RequireOrgMember`). `InviteMember` (POST `/api/orgs/:id/invites`, `RequireOrgRole("admin")`): create invite + email link. `ListInvites`/`DeleteInvite`. `UpdateMemberRole` (PATCH `/api/orgs/:id/members/:uid/role`), `RemoveMember` (DELETE) — both `RequireOrgRole("admin")` and guarded by `canDemoteOrRemove` (last-owner). Audit each mutation via the existing `Audit` helper (extend actor to web user — Task 15).

- [ ] **Step 2: Wire routes**

```go
orgs := app.Group("/api/orgs", webAuth.Middleware)
orgs.Post("", api.CreateOrg)
orgs.Get("", api.ListMyOrgs)
scoped := orgs.Group("/:id", middleware.RequireOrgMember(services))
scoped.Get("/members", api.ListOrgMembers)
scoped.Patch("/members/:uid/role", middleware.RequireOrgRole(services, "admin"), api.UpdateMemberRole)
scoped.Delete("/members/:uid", middleware.RequireOrgRole(services, "admin"), api.RemoveMember)
scoped.Get("/invites", middleware.RequireOrgRole(services, "admin"), api.ListInvites)
scoped.Post("/invites", middleware.RequireOrgRole(services, "admin"), api.InviteMember)
scoped.Delete("/invites/:iid", middleware.RequireOrgRole(services, "admin"), api.DeleteInvite)
```

- [ ] **Step 3: Build + lint**

Run: `cd backend && env -u GOROOT go build ./... && cd .. && make lint` Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/orgs.go backend/internal/delivery/http/app.go
git commit -m "feat(http): org create/list + members + invites routes"
```

---

## Task 15: Extend audit log actor to web users

**Files:**
- Modify: `backend/internal/application/audit.go` (or wherever `Audit`/`AuditContext` live — confirm with grep), `backend/internal/delivery/http/handlers/web_auth.go`/`orgs.go`

- [ ] **Step 1: Allow a web `platform_user` actor**

`AuditContext` currently keys on bot user (UserID/TelegramID/Email). Add an actor-kind so a web user can be the actor (TelegramID optional/nullable). Confirm the `admin_audit_log.actor_user_id` FK references `bot_users(id)` — if so, this phase records web-actor events with a NULL/separate path: either relax the FK (migration: drop FK or repoint to a generic actor) OR store web actor email + a sentinel. Choose the smallest change: **drop the `actor_user_id` FK constraint** (keep the column nullable) so any actor uuid (bot or platform user) can be recorded; add `actor_kind TEXT NOT NULL DEFAULT 'bot'`.

Migration `20260610170000_audit_actor_web.sql`:
```sql
-- +goose Up
ALTER TABLE admin_audit_log DROP CONSTRAINT IF EXISTS admin_audit_log_actor_user_id_fkey;
ALTER TABLE admin_audit_log ALTER COLUMN actor_user_id DROP NOT NULL;
ALTER TABLE admin_audit_log ADD COLUMN actor_kind TEXT NOT NULL DEFAULT 'bot';
-- +goose Down
ALTER TABLE admin_audit_log DROP COLUMN actor_kind;
ALTER TABLE admin_audit_log ALTER COLUMN actor_user_id SET NOT NULL;
ALTER TABLE admin_audit_log ADD CONSTRAINT admin_audit_log_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES bot_users(id);
```

- [ ] **Step 2: Add `WithWebAuditActor` helper + call it from web handlers**

Mirror the existing `WithAuditActor`/`withAuditActor` pattern but populate from `web_user` with `actor_kind="web"`.

- [ ] **Step 3: Build + lint + migrate**

Run: `make migrate && cd backend && env -u GOROOT go build ./... && cd .. && make lint` Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/20260610170000_audit_actor_web.sql backend/internal/application backend/internal/delivery/http/handlers
git commit -m "feat(audit): record web platform-user actors"
```

---

## Task 16: Frontend — surface detection + web auth client/context

**Files:**
- Create: `frontend/src/shared/lib/surface.ts`, `frontend/src/shared/web-auth/api.ts`, `frontend/src/shared/web-auth/queries.ts`, `frontend/src/shared/web-auth/context.tsx`
- Test: `frontend/src/shared/lib/surface.test.ts`

- [ ] **Step 1: Failing Vitest for surface detection**

```ts
import { describe, it, expect } from "vitest"
import { detectSurface } from "./surface"

describe("detectSurface", () => {
  it("returns telegram when initData present", () => {
    expect(detectSurface({ initData: "x" } as any)).toBe("telegram")
  })
  it("returns web otherwise", () => {
    expect(detectSurface(undefined)).toBe("web")
    expect(detectSurface({ initData: "" } as any)).toBe("web")
  })
})
```

- [ ] **Step 2: Run — fails; implement `detectSurface` + web-auth client**

`detectSurface(webApp)` returns `"telegram" | "web"`. `api.ts`: cookie-based fetchers using `fetch(url, {credentials:"include", headers:{"X-CSRF-Token": readCsrfCookie()}})` — `getMe()`, `logout()`, `requestMagicLink(email)`, `createOrg(name)`, `listMyOrgs()`. `context.tsx`: `WebAuthProvider` calling `getMe()`; states `loading|authed|anonymous|error`; exposes `user`, `orgs`, `activeOrgId` (+ setter persisted to `localStorage` and sent as `X-Org-Id`).

- [ ] **Step 3: Run — passes; build**

Run: `cd frontend && npm run test -- surface && npm run build` Expected: PASS, builds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/shared/lib/surface.ts frontend/src/shared/lib/surface.test.ts frontend/src/shared/web-auth
git commit -m "feat(web): surface detection + cookie web-auth client/context"
```

---

## Task 17: Frontend — /login, /onboarding, web shell + routing branch

**Files:**
- Create: `frontend/src/features/web-login/login-page.tsx`, `frontend/src/features/onboarding/onboarding-page.tsx`, `frontend/src/components/web-shell/web-layout.tsx`, `frontend/src/components/web-shell/org-switcher.tsx`, `frontend/src/routes/login.tsx`, `frontend/src/routes/onboarding.tsx`, `frontend/src/routes/_web.tsx`
- Modify: `frontend/src/app/app-content.tsx`

- [ ] **Step 1: Login page**

Buttons that link to `/api/auth/web/google/start` and `/api/auth/web/microsoft/start` (full-page nav, not fetch — these 302 to the provider) + an email field calling `requestMagicLink` then showing a "check your email" state.

- [ ] **Step 2: Onboarding + shell + org switcher**

`/onboarding`: name field → `createOrg` → navigate to `/`. `web-layout.tsx`: shell with `org-switcher` (reads `orgs`, sets `activeOrgId`), account menu, logout. `_web.tsx`: route that gates on `WebAuthProvider` authed state, redirects to `/login` if anonymous and to `/onboarding` if authed with zero orgs.

- [ ] **Step 3: Branch the bootstrap**

In `app-content.tsx`, compute `detectSurface(getTelegramWebApp())`. If `web`, render the web router tree (login/onboarding/_web) wrapped in `WebAuthProvider`; if `telegram`, render the existing MiniApp tree (parked — may be non-functional, acceptable this phase). Keep both route trees registered; the generated `routeTree.gen.ts` updates automatically.

- [ ] **Step 4: Build + typecheck**

Run: `cd frontend && npm run build` Expected: builds; routeTree regenerated.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/features/web-login frontend/src/features/onboarding frontend/src/components/web-shell frontend/src/routes/login.tsx frontend/src/routes/onboarding.tsx frontend/src/routes/_web.tsx frontend/src/app/app-content.tsx frontend/src/routeTree.gen.ts
git commit -m "feat(web): login, onboarding, web shell + surface routing branch"
```

---

## Task 18: Local dev story (Mailpit) + config docs + OpenAPI

**Files:**
- Modify: `docker-compose.yml` (or the project's local compose — confirm path), `.env.example`/`docs/SETUP.md`, `backend/openapi/openapi.json` + `docs/openapi.json`, `docs/MEETINGS.md` (status note)

- [ ] **Step 1: Add Mailpit to local compose + dev SMTP defaults**

Add a `mailpit` service (ports 1025 SMTP / 8025 UI); document `SMTP_HOST=localhost SMTP_PORT=1025 SMTP_FROM=dev@lead-cat.local` for `make dev`. Magic-link emails land in the Mailpit UI.

- [ ] **Step 2: Document new env keys**

In `docs/SETUP.md` (or `.env.example`): list all Task-12 config keys with example values and which are required for which auth method (Google/MS optional; SMTP required for magic-link).

- [ ] **Step 3: Update OpenAPI + status docs**

Add the `/api/auth/web/*` and `/api/orgs*` paths to `backend/openapi/openapi.json`, mirror to `docs/openapi.json`. Add a short "SaaS Phase 0" note to `docs/MEETINGS.md` (or a new `docs/SAAS.md` pointer) describing the new web auth surface and that Telegram is parked.

- [ ] **Step 4: Verify served OpenAPI**

Run: `cd backend && env -u GOROOT go build ./...` (embedded OpenAPI), then `make build`. Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml docs/SETUP.md backend/openapi/openapi.json docs/openapi.json docs/MEETINGS.md
git commit -m "docs(saas): Mailpit dev SMTP, env keys, OpenAPI + status for Phase 0"
```

---

## Task 19: Whole-phase verification

- [ ] **Step 1: Full backend gate**

Run: `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && cd .. && make lint && make build`
Expected: all green; gofmt clean (run `gofmt -l backend` → empty).

- [ ] **Step 2: Migrations round-trip**

Run on scratch DB: `make migrate` (up), goose `down` to the pre-Phase-0 version, then `up` again. Expected: reversible, no errors.

- [ ] **Step 3: Frontend gate**

Run: `cd frontend && npm run test && npm run build` Expected: green.

- [ ] **Step 4: Manual smoke (documented, not automated)**

With Mailpit + `AUTH_DEV_MODE`/SMTP dev config: start app, open web `/login`, request a magic link, click it from Mailpit → land on `/onboarding` → create org → see dashboard + org switcher → logout. Record result in the PR description.

- [ ] **Step 5: Final commit (if any doc/cleanup remains)**

```bash
git add -- <explicit changed paths>
git commit -m "chore(saas): Phase 0 verification fixes"
```

---

## Self-review notes (coverage vs spec)

- Spec §1 data model → Tasks 1–3, 10, 15 (audit). §2 auth layer (ports/routes/provisioning) → Tasks 5,6,7,13. §3 web session → Tasks 8,9. §4 tenant context → Tasks 10,11,14. §5 onboarding & invites → Tasks 10,14,17. §6 frontend shell → Tasks 16,17. §7 testing → unit tasks throughout + integration in Task 13 + Task 19. §8 breaking/migration → Tasks 1,15 (Telegram parked: Task 17 branch). §9 open items resolved: rename = single migration (Task 1) with explicit down; Mailpit dev SMTP (Task 18); config keys (Task 12); org switch via `X-Org-Id` header (Tasks 11,16). 
- Telegram `MiniAppAuth`/`RequireBotAdmin` left intact but parked at the frontend (Task 17) — acceptable per decision #8; backend stays compiling.
- No password auth, calendar, availability, notifications, billing — out of scope, confirmed absent from tasks.
