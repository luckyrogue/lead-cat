# Slice 2a-2 — Join-by-Slug Requests + Admin Accept/Decline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A user requests to join an org by slug; the org's admins accept (→ membership, role `member`) or decline.

**Architecture:** New `organization_join_requests` table + repo; user-side request/list under `/api/auth/web/me/join-requests` (web-session auth); admin list/accept/decline under `/api/orgs/:id/join-requests` (admin-gated). Onboarding gets a "Join by slug" card; the admin Invites page gets a "Join requests" card.

**Tech Stack:** Go 1.26 (Fiber, pgx, goose, testcontainers), React Router v7 / shadcn / TanStack Query admin SPA.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-18-slice-2a2-join-by-slug-design.md`.
- depguard: `application` imports zero `internal/infrastructure`; model types/sentinels in `internal/application/model` (leaf). No code comments in new Go/TS files.
- **User-side endpoints are web-session auth** (requester isn't a member) under `/api/auth/web/me/join-requests`; **admin endpoints** under `/api/orgs/:id/join-requests` + `RequireOrgRole("admin")`.
- **Accept membership insert is by `user_id`** (mirror `AcceptInvite`'s `INSERT INTO organization_members (organization_id, user_id, role) ... ON CONFLICT DO NOTHING`), NOT the username-based `AddMember`.
- Accepted role = `"member"`. One pending request per `(org,user)` (partial unique index); idempotent while pending.
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; i18n keys in en/ru/kk (admin **formal**); never repo-wide prettier (additive edits); pnpm filter `admin`.
- gofmt all Go; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green (testcontainers repo tests skip if Docker absent — verify SQL by inspection then).
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference (verified):**
- `gen_random_uuid()` is used by existing tables (`organization_invites` etc.) — safe for the new table's `id` default.
- `GetOrganizationBySlug(ctx, slug) (Organization, error)` on `*Store` (`organization_repo.go:122`) — NOT yet on `Repository`. `GetOrgMember(ctx, orgID, userID) (Member, bool, error)` (`:58`). `AcceptInvite` membership-insert pattern in `invite_accept_repo.go`.
- Org routes (`app.go:100-117`): `orgs := app.Group("/api/orgs", webAuth.Middleware)`; `scoped := orgs.Group("/:id", middleware.RequireOrgMember(store))`; admin writes add `middleware.RequireOrgRole("admin")`. Web group: `web := app.Group("/api/auth/web")` with `web.Get("/me/invites", webAuth.Middleware, api.WebMyInvites)` (2a-1). Handler caller: `c.Locals("web_user").(model.PlatformUser)` (.ID, .Email).
- Admin entity `entities/org/{types,api,queries}.ts` (axios `{ data }` client `api.get/post`); invites page `features/invites/pages/invites-page.tsx`; onboarding `routes/onboarding.tsx` (post-2a-1: invites card + create-org card, `useMyInvites`, `setActiveOrgId`, guarded by `me.organizations.length > 0`).

---

### Task 1: Migration + join-request repo + model

**Files:**
- Create: `apps/backend/migrations/20260618140000_org_join_requests.sql`
- Create: `apps/backend/internal/application/model/join_request.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/join_request_repo.go`
- Test: `apps/backend/internal/infrastructure/persistence/postgres/join_request_repo_test.go`

**Interfaces:**
- Produces:
  - `model.JoinRequestView{ OrganizationID uuid.UUID \`json:"organization_id"\`; OrgName string \`json:"org_name"\`; Status string \`json:"status"\` }`
  - `model.JoinRequestAdminView{ RequestID, UserID uuid.UUID; Name, Email string; CreatedAt time.Time }` with json tags `request_id`/`user_id`/`name`/`email`/`created_at`.
  - `(*Store).CreateJoinRequest(ctx, orgID, userID uuid.UUID) error`
  - `(*Store).ListJoinRequestsForUser(ctx, userID uuid.UUID) ([]model.JoinRequestView, error)`
  - `(*Store).ListPendingJoinRequests(ctx, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)`
  - `(*Store).AcceptJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error` — not-found → `model.IsNotFound`.
  - `(*Store).DeclineJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error` — not-found → `model.IsNotFound`.

- [ ] **Step 1: Migration** — `20260618140000_org_join_requests.sql`:
```sql
-- +goose Up
CREATE TABLE organization_join_requests (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    status             TEXT NOT NULL DEFAULT 'pending',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at         TIMESTAMPTZ,
    decided_by_user_id UUID REFERENCES platform_users(id)
);
CREATE UNIQUE INDEX organization_join_requests_pending_idx
    ON organization_join_requests (organization_id, user_id) WHERE status = 'pending';

-- +goose Down
DROP TABLE organization_join_requests;
```

- [ ] **Step 2: Model** — `model/join_request.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type JoinRequestView struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Status         string    `json:"status"`
}

type JoinRequestAdminView struct {
	RequestID uuid.UUID `json:"request_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
```
Add `postgres.JoinRequestView`/`JoinRequestAdminView` aliases if the package aliases model types.

- [ ] **Step 3: Write the failing repo test** — `join_request_repo_test.go` (`package postgres_test`, reuse `newStore()`/`testDB.Truncate(t)` + the `seedOrg`/`seedUser` helpers from the 2a-1 invite test if accessible in-package, else replicate minimal seeds):
```go
func TestJoinRequest_Lifecycle(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	daveID := seedUser(t, store, "dave@x.com")

	if err := store.CreateJoinRequest(ctx, orgID, daveID); err != nil {
		t.Fatalf("create: %v", err)
	}
	// idempotent while pending
	if err := store.CreateJoinRequest(ctx, orgID, daveID); err != nil {
		t.Fatalf("create idempotent: %v", err)
	}
	mine, err := store.ListJoinRequestsForUser(ctx, daveID)
	if err != nil || len(mine) != 1 || mine[0].Status != "pending" || mine[0].OrgName == "" {
		t.Fatalf("list-mine: %v %+v", err, mine)
	}
	pend, err := store.ListPendingJoinRequests(ctx, orgID)
	if err != nil || len(pend) != 1 || pend[0].Email != "dave@x.com" {
		t.Fatalf("list-pending: %v %+v", err, pend)
	}
	if err := store.AcceptJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	// membership now exists
	if _, ok, _ := store.GetOrgMember(ctx, orgID, daveID); !ok {
		t.Fatal("expected membership after accept")
	}
	// no longer pending
	if pend, _ := store.ListPendingJoinRequests(ctx, orgID); len(pend) != 0 {
		t.Fatalf("expected 0 pending after accept, got %d", len(pend))
	}
	// accept again -> not found
	if err := store.AcceptJoinRequest(ctx, orgID, pend0(mine), ownerID); !model.IsNotFound(err) {
		_ = err // see note: re-accept of a non-pending id -> IsNotFound
	}
}

func TestJoinRequest_Decline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	eveID := seedUser(t, store, "eve@x.com")
	_ = store.CreateJoinRequest(ctx, orgID, eveID)
	pend, _ := store.ListPendingJoinRequests(ctx, orgID)
	if err := store.DeclineJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, ok, _ := store.GetOrgMember(ctx, orgID, eveID); ok {
		t.Fatal("declined request must not create membership")
	}
	if p, _ := store.ListPendingJoinRequests(ctx, orgID); len(p) != 0 {
		t.Fatalf("declined request should not be pending")
	}
	// re-request after decline is allowed (partial unique index only blocks pending)
	if err := store.CreateJoinRequest(ctx, orgID, eveID); err != nil {
		t.Fatalf("re-request after decline: %v", err)
	}
}
```
(Drop the `pend0(mine)` re-accept assertion if it complicates — the decline test + the post-accept "0 pending" already cover the pending transitions. Keep the test clean; remove any helper you don't define.)

- [ ] **Step 4: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestJoinRequest -v`

- [ ] **Step 5: Implement** — `join_request_repo.go`:
```go
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organization_join_requests (organization_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (organization_id, user_id) WHERE status = 'pending' DO NOTHING`,
		orgID, userID)
	return err
}

func (s *Store) ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.organization_id, o.name, r.status
		FROM organization_join_requests r
		JOIN organizations o ON o.id = r.organization_id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.JoinRequestView{}
	for rows.Next() {
		var v model.JoinRequestView
		if err := rows.Scan(&v.OrganizationID, &v.OrgName, &v.Status); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.user_id, COALESCE(u.name, ''), u.email, r.created_at
		FROM organization_join_requests r
		JOIN platform_users u ON u.id = r.user_id
		WHERE r.organization_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.JoinRequestAdminView{}
	for rows.Next() {
		var v model.JoinRequestAdminView
		if err := rows.Scan(&v.RequestID, &v.UserID, &v.Name, &v.Email, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM organization_join_requests
		WHERE id = $1 AND organization_id = $2 AND status = 'pending'
		FOR UPDATE`, requestID, orgID).Scan(&userID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'member') ON CONFLICT (organization_id, user_id) DO NOTHING`,
		orgID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE organization_join_requests
		SET status = 'accepted', decided_at = now(), decided_by_user_id = $2
		WHERE id = $1`, requestID, deciderID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE organization_join_requests
		SET status = 'declined', decided_at = now(), decided_by_user_id = $3
		WHERE id = $1 AND organization_id = $2 AND status = 'pending'`,
		requestID, orgID, deciderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
```
Add the `"github.com/jackc/pgx/v5"` import for `pgx.ErrNoRows` (confirm the module's pgx version path matches the rest of the package). `pgx.ErrNoRows` wraps `sql.ErrNoRows` so `model.IsNotFound` is true.

- [ ] **Step 6: Run; expect PASS** (Docker present) or SKIP — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestJoinRequest -v`

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/model/join_request.go internal/infrastructure/persistence/postgres/join_request_repo.go internal/infrastructure/persistence/postgres/join_request_repo_test.go
git add apps/backend/migrations/20260618140000_org_join_requests.sql apps/backend/internal/application/model/join_request.go apps/backend/internal/infrastructure/persistence/postgres/join_request_repo.go apps/backend/internal/infrastructure/persistence/postgres/join_request_repo_test.go
# include models.go if aliased
git commit -m "feat(orgs): organization_join_requests table + repo (create/list/accept/decline)"
```

---

### Task 2: Application + HTTP endpoints (user + admin)

**Files:**
- Create: `apps/backend/internal/application/join_requests.go`
- Modify: `apps/backend/internal/application/repository.go` — add the new repo methods + `GetOrganizationBySlug`
- Create: `apps/backend/internal/delivery/http/handlers/join_requests.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — register user + admin routes
- Test: `apps/backend/internal/delivery/http/handlers/join_requests_test.go`

**Interfaces:**
- Produces:
  - `application.JoinResult{ AlreadyMember bool; OrganizationID uuid.UUID }`
  - `(s *Services) RequestToJoinBySlug(ctx, userID uuid.UUID, slug string) (JoinResult, error)` — slug not found → `model.IsNotFound`-error.
  - `(s *Services) ListMyJoinRequests(ctx, userID uuid.UUID) ([]model.JoinRequestView, error)`
  - `(s *Services) ListOrgJoinRequests(ctx, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)`
  - `(s *Services) AcceptJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error`
  - `(s *Services) DeclineJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error`
  - Endpoints: `POST /api/auth/web/me/join-requests`, `GET /api/auth/web/me/join-requests`, `GET /api/orgs/:id/join-requests`, `POST /api/orgs/:id/join-requests/:rid/accept`, `POST /api/orgs/:id/join-requests/:rid/decline`.

- [ ] **Step 1: Repository port** — add to `internal/application/repository.go`'s `Repository` interface:
```go
GetOrganizationBySlug(ctx context.Context, slug string) (model.Organization, error)
CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error
ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error)
ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)
AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
```
(If `GetOrgMember` / `GetOrganizationBySlug` are already on the interface — check first — don't duplicate. `*postgres.Store` already implements all of these; the postgres return types are aliases of `model.X`.)

- [ ] **Step 2: Application methods** — `internal/application/join_requests.go`:
```go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type JoinResult struct {
	AlreadyMember  bool      `json:"already_member"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

func (s *Services) RequestToJoinBySlug(ctx context.Context, userID uuid.UUID, slug string) (JoinResult, error) {
	org, err := s.Store.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return JoinResult{}, err
	}
	if _, ok, err := s.Store.GetOrgMember(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	} else if ok {
		return JoinResult{AlreadyMember: true, OrganizationID: org.ID}, nil
	}
	if err := s.Store.CreateJoinRequest(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{OrganizationID: org.ID}, nil
}

func (s *Services) ListMyJoinRequests(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	return s.Store.ListJoinRequestsForUser(ctx, userID)
}

func (s *Services) ListOrgJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return s.Store.ListPendingJoinRequests(ctx, orgID)
}

func (s *Services) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.Store.AcceptJoinRequest(ctx, orgID, requestID, deciderID)
}

func (s *Services) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.Store.DeclineJoinRequest(ctx, orgID, requestID, deciderID)
}
```

- [ ] **Step 3: Handlers** — `internal/delivery/http/handlers/join_requests.go`:
```go
package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) WebRequestToJoin(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	var body struct {
		Slug string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Slug) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing_slug")
	}
	res, err := a.App.RequestToJoinBySlug(c.UserContext(), user.ID, strings.TrimSpace(strings.ToLower(body.Slug)))
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "org_not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "join_failed")
	}
	if res.AlreadyMember {
		return c.JSON(fiber.Map{"already_member": true, "organization_id": res.OrganizationID})
	}
	return c.JSON(fiber.Map{"status": "pending", "organization_id": res.OrganizationID})
}

func (a *API) WebMyJoinRequests(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	views, err := a.App.ListMyJoinRequests(c.UserContext(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) OrgJoinRequests(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	views, err := a.App.ListOrgJoinRequests(c.UserContext(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) AcceptJoinRequest(c *fiber.Ctx) error {
	return a.decideJoinRequest(c, true)
}

func (a *API) DeclineJoinRequest(c *fiber.Ctx) error {
	return a.decideJoinRequest(c, false)
}

func (a *API) decideJoinRequest(c *fiber.Ctx, accept bool) error {
	user := c.Locals("web_user").(model.PlatformUser)
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	rid, err := uuid.Parse(c.Params("rid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if accept {
		err = a.App.AcceptJoinRequest(c.UserContext(), orgID, rid, user.ID)
	} else {
		err = a.App.DeclineJoinRequest(c.UserContext(), orgID, rid, user.ID)
	}
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "request_not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "decide_failed")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 4: Register routes** — in `app.go`:
  - Web group (with the other `/me/*`): `web.Post("/me/join-requests", webAuth.Middleware, api.WebRequestToJoin)` and `web.Get("/me/join-requests", webAuth.Middleware, api.WebMyJoinRequests)`.
  - Admin (on `scoped`, admin-gated): `scoped.Get("/join-requests", middleware.RequireOrgRole("admin"), api.OrgJoinRequests)`; `scoped.Post("/join-requests/:rid/accept", middleware.RequireOrgRole("admin"), api.AcceptJoinRequest)`; `scoped.Post("/join-requests/:rid/decline", middleware.RequireOrgRole("admin"), api.DeclineJoinRequest)`. (Match the exact `RequireOrgRole` call style used by the existing invite routes.)

- [ ] **Step 5: Handler test** — `join_requests_test.go`: fake `Repository` (embed `application.Repository`, override the join-request methods + `GetOrganizationBySlug` + `GetOrgMember`), real `Services`, test middleware injecting `web_user`. Assert: `POST /me/join-requests {slug}` → `{status:"pending"}` when org found + not member; → `{already_member:true}` when GetOrgMember returns true; → 404 when GetOrganizationBySlug returns `sql.ErrNoRows`; → 400 when slug missing. `GET /me/join-requests` lists. Admin `GET /:id/join-requests` lists; accept→204; decline→204; not-found→404. (Pure fakes; no Docker.)

- [ ] **Step 6: Run + build/vet/lint** — `env -u GOROOT go test ./internal/delivery/http/... -run Join -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green.

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/join_requests.go internal/application/repository.go internal/delivery/http/handlers/join_requests.go internal/delivery/http/app.go internal/delivery/http/handlers/join_requests_test.go
git add apps/backend/internal/application/join_requests.go apps/backend/internal/application/repository.go apps/backend/internal/delivery/http/handlers/join_requests.go apps/backend/internal/delivery/http/app.go apps/backend/internal/delivery/http/handlers/join_requests_test.go
git commit -m "feat(orgs): join-request endpoints (request/list + admin accept/decline)"
```

---

### Task 3: Frontend — join-by-slug card + admin join-requests card + i18n

**Files:**
- Create: `apps/admin/app/entities/join-request/{types.ts,api.ts,queries.ts}`
- Modify: `apps/admin/app/routes/onboarding.tsx` — "Join by slug" card
- Modify: `apps/admin/app/features/invites/pages/invites-page.tsx` — "Join requests" admin card
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts`

**Interfaces:**
- Consumes: `POST /api/auth/web/me/join-requests`, `GET /api/auth/web/me/join-requests`, `GET /api/orgs/:id/join-requests`, `POST /api/orgs/:id/join-requests/:rid/accept|decline`.
- Produces: `useMyJoinRequests`, `useRequestToJoin`; `useOrgJoinRequests(orgId)`, `useAcceptJoinRequest`, `useDeclineJoinRequest`.

- [ ] **Step 1: entity** — `types.ts`:
```ts
export type MyJoinRequest = { organization_id: string; org_name: string; status: string }
export type JoinResult = { already_member?: boolean; status?: string; organization_id: string }
export type OrgJoinRequest = { request_id: string; user_id: string; name: string; email: string; created_at: string }
```
`api.ts` (axios `{ data }` client, mirror `entities/org/api.ts`):
```ts
import { api } from "~/shared/api/client"
import type { JoinResult, MyJoinRequest, OrgJoinRequest } from "./types"

export async function requestToJoin(slug: string): Promise<JoinResult> {
  const { data } = await api.post<JoinResult>("/api/auth/web/me/join-requests", { slug })
  return data
}
export async function listMyJoinRequests(): Promise<MyJoinRequest[]> {
  const { data } = await api.get<MyJoinRequest[]>("/api/auth/web/me/join-requests")
  return data
}
export async function listOrgJoinRequests(orgId: string): Promise<OrgJoinRequest[]> {
  const { data } = await api.get<OrgJoinRequest[]>(`/api/orgs/${orgId}/join-requests`)
  return data
}
export async function acceptJoinRequest(orgId: string, rid: string): Promise<void> {
  await api.post(`/api/orgs/${orgId}/join-requests/${rid}/accept`, {})
}
export async function declineJoinRequest(orgId: string, rid: string): Promise<void> {
  await api.post(`/api/orgs/${orgId}/join-requests/${rid}/decline`, {})
}
```
`queries.ts`: `useMyJoinRequests` (query); `useRequestToJoin` (mutation); `useOrgJoinRequests(orgId)` (query, `enabled: !!orgId`); `useAcceptJoinRequest`/`useDeclineJoinRequest` (mutations invalidating the org-join-requests key + the members key `["org","members",orgId]` — match the real members query key from `entities/org/queries.ts`). Define `orgJoinRequestsKey(orgId)` + `myJoinRequestsKey`.

- [ ] **Step 2: Onboarding "Join by slug" card** — in `onboarding.tsx`, between the invites card and the create-org card, add a card: a slug `<Input>` + Submit calling `useRequestToJoin`. On success: `data.already_member` → `setActiveOrgId(data.organization_id)` + `navigate("/", {replace:true})`; else show a "request sent" inline state. Render existing pending requests from `useMyJoinRequests()` ("Pending approval — {org_name}"). On 404 (toApiError status 404) → inline `t("onboarding.join.notFound")`. ≤300 lines total, no comments.

- [ ] **Step 3: Admin "Join requests" card** — in `invites-page.tsx`, add a third Card (after the invites table) shown when the caller can manage (admin) — `useOrgJoinRequests(activeOrgId)` listing name/email with Accept / Decline buttons (`useAcceptJoinRequest`/`useDeclineJoinRequest`). Hidden/empty when none. Match the page's existing card/table style.

- [ ] **Step 4: i18n** — add to en/ru/kk: `onboarding.join.{title,description,slugLabel,slugPlaceholder,submit,sent,pending,notFound}` and `invites.requests.{title,description,accept,decline,empty}`. Provide real formal-RU + KK for every key (parity compile-enforced).

- [ ] **Step 5: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/entities/join-request apps/admin/app/routes/onboarding.tsx apps/admin/app/features/invites/pages/invites-page.tsx apps/admin/app/shared/i18n/dictionaries
git commit -m "feat(admin): join-by-slug onboarding card + admin join-requests review + i18n"
```

---

### Task 4: Whole-slice verification

**Files:** none

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green (join-request repo tests run if Docker present, else skip — note it).
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green; i18n parity.
- [ ] **Step 3: Route gating check (documented)** — confirm admin join-request routes are under `RequireOrgRole("admin")` and the user request/list routes are under `webAuth.Middleware` (not member-gated).
- [ ] **Step 4: Tree clean** — verify HEAD; `git status` no stray staged files; user parallel WIP untouched.

---

## Notes for the executor

- **Membership insert is by `user_id`** (mirror `AcceptInvite`), NOT the username-based `AddMember`.
- **`ON CONFLICT ... WHERE status='pending'`** targets the partial unique index — keep the predicate in the upsert.
- **User vs admin auth:** request/list-mine = `webAuth.Middleware` only (requester isn't a member); admin list/accept/decline = `scoped` + `RequireOrgRole("admin")`.
- **404 mapping:** unknown slug → 404; unknown/decided request id → 404 (repo returns `pgx.ErrNoRows` → `model.IsNotFound`).
- **Docker** may be unavailable → repo tests skip; the handler/app tests use fakes and run regardless. Verify SQL by inspection in review when tests skip.
- **Deferred:** admin notification of new requests (2a-2 is in-app list only); the activation checklist (2a-3).
