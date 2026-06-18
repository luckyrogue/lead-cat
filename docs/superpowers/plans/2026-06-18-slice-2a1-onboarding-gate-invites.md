# Slice 2a-1 — Onboarding Gate Fix + Explicit Invite Accept/Decline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Invited users land on an Accept/Decline screen (not forced to create-org), and a user with an org can't accidentally re-create one via Back.

**Architecture:** Drop silent invite auto-accept at login; add email-validated `/api/auth/web/me/invites/*` endpoints (list/accept/decline) backed by new repo methods + a `declined_at` column. Fix the admin gate to use fresh membership and guard `/onboarding` pre-render; the onboarding screen lists pending invites with Accept/Decline.

**Tech Stack:** Go 1.26 (Fiber, pgx, goose, testcontainers), React Router v7 / shadcn / TanStack Query admin SPA.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-18-slice-2a1-onboarding-gate-invites-design.md`.
- depguard: `application` imports zero `internal/infrastructure`; error sentinels in `internal/application/model` (leaf). No code comments in new Go/TS files.
- Invite accept/decline is **web-session auth, email-validated** (the invitee isn't an org member) — under `/api/auth/web/me/invites/*`, NOT `/api/orgs/:id/...`.
- Pending invite = `accepted_at IS NULL AND declined_at IS NULL AND expires_at > now()`.
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; i18n keys in ALL THREE dicts (en/ru/kk), admin **formal** ("вы"); never run repo-wide prettier (keep edits additive); pnpm filter is `admin`.
- gofmt all Go; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green.
- Work on `main`; never `git add -A` (stage explicit paths); **verify HEAD before each commit** (the user commits in parallel — commit your staged files on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference — existing (verified):**
- `organization_invites` cols: `id, organization_id, email, role, token_hash, expires_at, accepted_at, created_by_user_id, created_at` (migration `20260610150000_org_auth_tables.sql`). `OrganizationInvite` model + `postgres.OrganizationInvite` alias exist.
- `org_invite_repo.go` has `CreateInvite`/`ListInvites`/`DeleteInvite`/`AcceptInvitesForEmail` (the last is called at login `web_auth.go:135` `WebAuthCallback` + `:187` `WebMagicVerify` — both to be removed). `s.pool` is the pgx pool.
- `model/errors.go` has `ErrMeetingNotEditable` + `IsNotFound(err) = errors.Is(err, sql.ErrNoRows)`.
- Web-auth group in `app.go:81-89`: `web := app.Group("/api/auth/web")`; routes mounted `web.Get("/me", webAuth.Middleware, api.WebMe)` etc. `api` is the `*handlers.API`.
- Handler caller identity: `c.Locals("web_user").(model.PlatformUser)` (`.Email`, `.ID`).
- Admin: `routes/_app.tsx:32-34` gate (`Navigate to /onboarding` if 0 orgs); `routes/onboarding.tsx` (full file — `useEffect` redirect + `useCreateOrg`); `shared/auth/use-me.ts` (`meQueryKey`, `staleTime: 60_000`); `entities/org/queries.ts` (`useCreateOrg` invalidates `meQueryKey`); `shared/api/active-org` `setActiveOrgId`; `shared/auth/types.ts` `Me` (has `organizations`). Admin API client: `~/shared/api/client` `api.get/post`.

---

### Task 1: Migration + repo methods + model

**Files:**
- Create: `apps/backend/migrations/20260618130000_invite_declined_at.sql`
- Modify: `apps/backend/internal/application/model/errors.go` — add `ErrInviteEmailMismatch`
- Modify: `apps/backend/internal/application/model/` — add `InviteView` (e.g. in `model.go` or a new `invite.go`)
- Create: `apps/backend/internal/infrastructure/persistence/postgres/invite_accept_repo.go`
- Test: `apps/backend/internal/infrastructure/persistence/postgres/invite_accept_repo_test.go`

**Interfaces:**
- Produces:
  - `model.InviteView{ InviteID, OrganizationID uuid.UUID; OrgName, Role string }`
  - `model.ErrInviteEmailMismatch` (sentinel `error`)
  - `(*Store).ListPendingInvitesForEmail(ctx, email string) ([]model.InviteView, error)`
  - `(*Store).AcceptInvite(ctx, inviteID, userID uuid.UUID, email string) error` — `model.IsNotFound`-error when no pending invite; `model.ErrInviteEmailMismatch` when the invite's email ≠ `email`.
  - `(*Store).DeclineInvite(ctx, inviteID uuid.UUID, email string) error` — same not-found / mismatch semantics.

- [ ] **Step 1: Migration** — `20260618130000_invite_declined_at.sql`:
```sql
-- +goose Up
ALTER TABLE organization_invites ADD COLUMN declined_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE organization_invites DROP COLUMN declined_at;
```

- [ ] **Step 2: Model + error** — append to `model/errors.go`:
```go
var ErrInviteEmailMismatch = errors.New("invite email does not match user")
```
Add `InviteView` (new file `model/invite.go`):
```go
package model

import "github.com/google/uuid"

type InviteView struct {
	InviteID       uuid.UUID `json:"invite_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Role           string    `json:"role"`
}
```
The JSON tags are required — this struct is returned directly by the `GET /me/invites` handler (Task 2) and the admin client expects `invite_id`/`organization_id`/`org_name`/`role`. Add the `postgres.InviteView = model.InviteView` alias if the package aliases model types.

- [ ] **Step 3: Write the failing repo test** — `invite_accept_repo_test.go` (`package postgres_test`, reuse the dir's testcontainers harness — `newStore()`/`testDB.Truncate(t)` as the other repo tests do). Seed an org + a platform user + an invite, then:
```go
func TestInviteAccept_ListAcceptDecline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	// seed: create org (owner), create a platform user "bob@x.com", create a pending invite for bob to the org as "member"
	orgID, ownerID := seedOrg(t, store)          // helper: create user+org, returns ids
	bobID := seedUser(t, store, "bob@x.com")
	invID := seedInvite(t, store, orgID, "bob@x.com", "member", ownerID) // INSERT organization_invites, expires in 24h

	views, err := store.ListPendingInvitesForEmail(ctx, "BOB@x.com")
	if err != nil || len(views) != 1 || views[0].InviteID != invID || views[0].OrgName == "" {
		t.Fatalf("list: %v %+v", err, views)
	}

	// email mismatch -> ErrInviteEmailMismatch
	if err := store.AcceptInvite(ctx, invID, bobID, "eve@x.com"); !errors.Is(err, model.ErrInviteEmailMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
	// accept -> membership exists, invite no longer pending
	if err := store.AcceptInvite(ctx, invID, bobID, "bob@x.com"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	views, _ = store.ListPendingInvitesForEmail(ctx, "bob@x.com")
	if len(views) != 0 {
		t.Fatalf("expected no pending after accept, got %v", views)
	}
	// second accept -> not found (already accepted)
	if err := store.AcceptInvite(ctx, invID, bobID, "bob@x.com"); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound on re-accept, got %v", err)
	}
}

func TestInviteDecline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	_ = seedUser(t, store, "carol@x.com")
	invID := seedInvite(t, store, orgID, "carol@x.com", "member", ownerID)
	if err := store.DeclineInvite(ctx, invID, "carol@x.com"); err != nil {
		t.Fatalf("decline: %v", err)
	}
	views, _ := store.ListPendingInvitesForEmail(ctx, "carol@x.com")
	if len(views) != 0 {
		t.Fatalf("declined invite should not be pending, got %v", views)
	}
}
```
Write the `seedOrg`/`seedUser`/`seedInvite` helpers in the test file using the existing store methods where available (e.g. `CreateOrganization`/`UpsertWebIdentity`/`CreateInvite`) — inspect the store for the real method names; fall back to direct `store`-pool inserts only if no method exists. Keep helpers minimal.

- [ ] **Step 4: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestInvite -v`

- [ ] **Step 5: Implement** — `invite_accept_repo.go`:
```go
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.organization_id, o.name, i.role
		FROM organization_invites i
		JOIN organizations o ON o.id = i.organization_id
		WHERE lower(i.email) = lower($1)
		  AND i.accepted_at IS NULL AND i.declined_at IS NULL AND i.expires_at > now()
		ORDER BY i.created_at`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.InviteView{}
	for rows.Next() {
		var v model.InviteView
		if err := rows.Scan(&v.InviteID, &v.OrganizationID, &v.OrgName, &v.Role); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID uuid.UUID
	var inviteEmail, role string
	err = tx.QueryRow(ctx, `
		SELECT organization_id, email, role
		FROM organization_invites
		WHERE id = $1 AND accepted_at IS NULL AND declined_at IS NULL AND expires_at > now()
		FOR UPDATE`, inviteID).Scan(&orgID, &inviteEmail, &role)
	if err != nil {
		return err
	}
	if !strings.EqualFold(inviteEmail, email) {
		return model.ErrInviteEmailMismatch
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3) ON CONFLICT (organization_id, user_id) DO NOTHING`,
		orgID, userID, role); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE organization_invites SET accepted_at = now() WHERE id = $1`, inviteID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	var inviteEmail string
	err := s.pool.QueryRow(ctx, `
		SELECT email FROM organization_invites
		WHERE id = $1 AND accepted_at IS NULL AND declined_at IS NULL AND expires_at > now()`,
		inviteID).Scan(&inviteEmail)
	if err != nil {
		return err
	}
	if !strings.EqualFold(inviteEmail, email) {
		return model.ErrInviteEmailMismatch
	}
	_, err = s.pool.Exec(ctx, `UPDATE organization_invites SET declined_at = now() WHERE id = $1`, inviteID)
	return err
}
```
(Add `"strings"` import. The `QueryRow ... Scan` returning `pgx.ErrNoRows` wraps `sql.ErrNoRows`, so `model.IsNotFound` is true for missing/expired/decided invites — no extra branch.)

- [ ] **Step 6: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestInvite -v`

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/model/errors.go internal/application/model/invite.go internal/infrastructure/persistence/postgres/invite_accept_repo.go internal/infrastructure/persistence/postgres/invite_accept_repo_test.go
git add apps/backend/migrations/20260618130000_invite_declined_at.sql apps/backend/internal/application/model/errors.go apps/backend/internal/application/model/invite.go apps/backend/internal/infrastructure/persistence/postgres/invite_accept_repo.go apps/backend/internal/infrastructure/persistence/postgres/invite_accept_repo_test.go
# include models.go if you added the alias
git commit -m "feat(orgs): declined_at + pending-invite list/accept/decline repo"
```

---

### Task 2: App methods + remove auto-accept + HTTP endpoints

**Files:**
- Create: `apps/backend/internal/application/invites.go` — `ListMyInvites`/`AcceptInvite`/`DeclineInvite`
- Modify: `apps/backend/internal/application/repository.go` — add the 3 repo methods to the `Repository` interface
- Modify: `apps/backend/internal/delivery/http/handlers/web_auth.go` — REMOVE the two `AcceptInvitesForEmail` calls (`WebAuthCallback`, `WebMagicVerify`)
- Create: `apps/backend/internal/delivery/http/handlers/web_invites.go` — the 3 handlers
- Modify: `apps/backend/internal/delivery/http/app.go` — register the 3 routes under the `web` group
- Test: `apps/backend/internal/delivery/http/handlers/web_invites_test.go`

**Interfaces:**
- Consumes (Task 1): `ListPendingInvitesForEmail`, `AcceptInvite`, `DeclineInvite`, `model.InviteView`, `model.ErrInviteEmailMismatch`, `model.IsNotFound`.
- Produces:
  - `(s *Services) ListMyInvites(ctx, email string) ([]model.InviteView, error)`
  - `(s *Services) AcceptInvite(ctx, inviteID, userID uuid.UUID, email string) error`
  - `(s *Services) DeclineInvite(ctx, inviteID uuid.UUID, email string) error`
  - Endpoints: `GET /api/auth/web/me/invites`, `POST /api/auth/web/me/invites/:iid/accept`, `POST /api/auth/web/me/invites/:iid/decline`.

- [ ] **Step 1: Repository port** — add to `internal/application/repository.go`'s `Repository` interface:
```go
ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error)
AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error
DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error
```

- [ ] **Step 2: Application methods** — `internal/application/invites.go`:
```go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) ListMyInvites(ctx context.Context, email string) ([]model.InviteView, error) {
	return s.Store.ListPendingInvitesForEmail(ctx, email)
}

func (s *Services) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	return s.Store.AcceptInvite(ctx, inviteID, userID, email)
}

func (s *Services) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	return s.Store.DeclineInvite(ctx, inviteID, email)
}
```

- [ ] **Step 3: Remove auto-accept at login** — in `web_auth.go`, delete the `AcceptInvitesForEmail` blocks in BOTH `WebAuthCallback` (~`:135`) and `WebMagicVerify` (~`:187`). Concretely, remove:
```go
if _, err := a.App.AcceptInvitesForEmail(ctx, profile.Email, user.ID); err != nil {
	a.Log.Warn("web_accept_invites_failed", zap.Error(err))
}
```
(and the equivalent in the magic-verify handler — match its exact variable names). Leave everything else (identity upsert, session, `postLoginDest`) intact. If `zap` becomes unused in the file, the build will flag it — keep it if other calls use it (they do).

- [ ] **Step 4: Write the failing handler test** — `web_invites_test.go`: build a Fiber app, inject `c.Locals("web_user", model.PlatformUser{ID: bobID, Email: "bob@x.com"})` via a tiny test middleware, register the 3 routes against a `Services` over the testcontainers store (or a fake `Repository` — match the pattern `calendar_connect_test.go` used: in-memory fake embedding `application.Repository`, overriding the 3 invite methods). Assert:
  - `GET /api/auth/web/me/invites` → 200 with the seeded pending invite.
  - `POST /api/auth/web/me/invites/<iid>/accept` → 204.
  - accept with a caller email ≠ invite email → 403.
  - `POST .../<iid>/decline` → 204.
  - accept of an unknown invite id → 404.

- [ ] **Step 5: Handlers** — `web_invites.go`:
```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) WebMyInvites(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	views, err := a.App.ListMyInvites(c.UserContext(), user.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) WebAcceptInvite(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	iid, err := uuid.Parse(c.Params("iid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if err := a.App.AcceptInvite(c.UserContext(), iid, user.ID, user.Email); err != nil {
		return inviteError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) WebDeclineInvite(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	iid, err := uuid.Parse(c.Params("iid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_id")
	}
	if err := a.App.DeclineInvite(c.UserContext(), iid, user.Email); err != nil {
		return inviteError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func inviteError(err error) error {
	if errors.Is(err, model.ErrInviteEmailMismatch) {
		return fiber.NewError(fiber.StatusForbidden, "email_mismatch")
	}
	if model.IsNotFound(err) {
		return fiber.NewError(fiber.StatusNotFound, "invite_not_found")
	}
	return fiber.NewError(fiber.StatusInternalServerError, "invite_failed")
}
```
(The `JSON(views)` returns `InviteView` with default field names — add JSON tags to `InviteView` in Task 1 if a specific shape is wanted; the admin client expects `invite_id`/`organization_id`/`org_name`/`role` — ADD those json tags to `model.InviteView` in Task 1: `\`json:"invite_id"\`` etc. **Update Task 1's struct accordingly.**)

- [ ] **Step 6: Register routes** — in `app.go` after `web.Patch("/me/settings", ...)`:
```go
web.Get("/me/invites", webAuth.Middleware, api.WebMyInvites)
web.Post("/me/invites/:iid/accept", webAuth.Middleware, api.WebAcceptInvite)
web.Post("/me/invites/:iid/decline", webAuth.Middleware, api.WebDeclineInvite)
```

- [ ] **Step 7: Run + build/vet/lint** — `env -u GOROOT go test ./internal/delivery/http/... -run Invite -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green.

- [ ] **Step 8: gofmt + commit**
```bash
gofmt -w internal/application/invites.go internal/application/repository.go internal/delivery/http/handlers/web_auth.go internal/delivery/http/handlers/web_invites.go internal/delivery/http/app.go internal/delivery/http/handlers/web_invites_test.go
git add apps/backend/internal/application/invites.go apps/backend/internal/application/repository.go \
        apps/backend/internal/delivery/http/handlers/web_auth.go apps/backend/internal/delivery/http/handlers/web_invites.go \
        apps/backend/internal/delivery/http/app.go apps/backend/internal/delivery/http/handlers/web_invites_test.go
git commit -m "feat(orgs): explicit invite accept/decline endpoints; drop login auto-accept"
```

---

### Task 3: Admin frontend — gate freshness + onboarding guard + invites UI + i18n

**Files:**
- Modify: `apps/admin/app/shared/auth/use-me.ts` — fresh membership at gate
- Modify: `apps/admin/app/routes/onboarding.tsx` — pre-render guard + invites section
- Create: `apps/admin/app/entities/invite/{types.ts,api.ts,queries.ts}`
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — `onboarding.invites.*`

**Interfaces:**
- Consumes: `GET /api/auth/web/me/invites`, `POST /api/auth/web/me/invites/:iid/accept`, `POST .../:iid/decline`; `meQueryKey`, `setActiveOrgId`.
- Produces: `useMyInvites()` + `useAcceptInvite()`/`useDeclineInvite()` hooks; an invites section in onboarding.

- [ ] **Step 1: Gate freshness** — in `use-me.ts`, add `refetchOnMount: "always"` to the `useQuery` options (keep `staleTime`/`retry`). This ensures `_app.tsx` and `onboarding.tsx` re-read membership on mount, so an invited/just-joined user is never gated on stale "0 orgs".
```ts
return useQuery({
  queryKey: meQueryKey,
  queryFn: fetchMe,
  retry: false,
  staleTime: 60_000,
  refetchOnMount: "always",
})
```

- [ ] **Step 2: Onboarding pre-render guard** — in `onboarding.tsx` `OnboardingBody`, REPLACE the `useEffect` redirect with a render-time early return (so the create form is never shown to a user who already has an org, killing the Back-reentry create). Remove the `useEffect`/`useNavigate`-for-redirect usage for this purpose; keep `useNavigate` for post-create. After the `if (!me) return <Navigate to="/login" replace />`:
```tsx
if (me.organizations.length > 0) {
  return <Navigate to="/" replace />
}
```
(Place it right after the `!me` guard; delete the now-dead `useEffect`.)

- [ ] **Step 3: `entities/invite`** — `types.ts`:
```ts
export type MyInvite = {
  invite_id: string
  organization_id: string
  org_name: string
  role: string
}
```
`api.ts` (mirror `entities/org/api.ts` client usage):
```ts
import { api } from "~/shared/api/client"
import type { MyInvite } from "./types"

export async function listMyInvites(): Promise<MyInvite[]> {
  const { data } = await api.get<MyInvite[]>("/api/auth/web/me/invites")
  return data
}
export async function acceptInvite(iid: string): Promise<void> {
  await api.post(`/api/auth/web/me/invites/${iid}/accept`, {})
}
export async function declineInvite(iid: string): Promise<void> {
  await api.post(`/api/auth/web/me/invites/${iid}/decline`, {})
}
```
`queries.ts`:
```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { acceptInvite, declineInvite, listMyInvites } from "./api"
import { meQueryKey } from "~/shared/auth/use-me"

export const myInvitesKey = ["invites", "mine"] as const

export function useMyInvites() {
  return useQuery({ queryKey: myInvitesKey, queryFn: listMyInvites, retry: false })
}
export function useAcceptInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: acceptInvite,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meQueryKey })
      void qc.invalidateQueries({ queryKey: myInvitesKey })
    },
  })
}
export function useDeclineInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: declineInvite,
    onSuccess: () => void qc.invalidateQueries({ queryKey: myInvitesKey }),
  })
}
```
(Match the real `api` client return shape from `entities/org/api.ts` — adjust `{ data }` destructuring if that client returns data directly.)

- [ ] **Step 4: Invites section in onboarding** — in `onboarding.tsx`, above the create-org `Card`, render pending invites when present. Add `const { data: invites = [] } = useMyInvites()`, `const accept = useAcceptInvite()`, `const decline = useDeclineInvite()`. For each invite, a row with org name + Accept (on success: `setActiveOrgId(invite.organization_id)` then `navigate("/", { replace: true })`) + Decline buttons, all strings via `t()`. Keep the file ≤300 lines, no comments. Example block:
```tsx
{invites.length > 0 ? (
  <Card>
    <CardHeader>
      <CardTitle>{t("onboarding.invites.title")}</CardTitle>
      <CardDescription>{t("onboarding.invites.description")}</CardDescription>
    </CardHeader>
    <CardContent className="space-y-3">
      {invites.map((inv) => (
        <div key={inv.invite_id} className="flex items-center justify-between gap-3">
          <span className="font-medium">{inv.org_name}</span>
          <div className="flex gap-2">
            <Button
              size="sm"
              disabled={accept.isPending}
              onClick={() =>
                accept.mutate(inv.invite_id, {
                  onSuccess: () => {
                    setActiveOrgId(inv.organization_id)
                    navigate("/", { replace: true })
                  },
                  onError: (e) => toastError(e, t("onboarding.invites.acceptFailed")),
                })
              }
            >
              {t("onboarding.invites.accept")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={decline.isPending}
              onClick={() => decline.mutate(inv.invite_id)}
            >
              {t("onboarding.invites.decline")}
            </Button>
          </div>
        </div>
      ))}
    </CardContent>
  </Card>
) : null}
```

- [ ] **Step 5: i18n** — add to en/ru/kk under `onboarding.invites`: `title`, `description`, `accept`, `decline`, `acceptFailed`. EN: "You've been invited" / "Accept an invitation to join an existing organization, or create your own below." / "Accept" / "Decline" / "Couldn't accept the invitation". Provide formal RU + KK for all five. (Key parity compile-enforced — all three dicts.)

- [ ] **Step 6: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/shared/auth/use-me.ts apps/admin/app/routes/onboarding.tsx apps/admin/app/entities/invite apps/admin/app/shared/i18n/dictionaries
git commit -m "feat(admin): onboarding invites accept/decline + gate freshness + onboarding guard"
```

---

### Task 4: Whole-slice verification

**Files:** none (verification only)

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green; invite repo + handler tests pass.
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green; i18n parity en/ru/kk.
- [ ] **Step 3: Behavior check (documented)** — confirm in code: login handlers no longer call `AcceptInvitesForEmail`; `/onboarding` returns `<Navigate to="/">` pre-render when `organizations.length > 0`; the gate's `me` refetches on mount.
- [ ] **Step 4: Tree clean** — verify HEAD; `git status` shows no stray staged files; user parallel WIP untouched.

---

## Notes for the executor

- **The two bugs map to:** (1) invited-user-forced-to-create → drop auto-accept + surface invites + gate freshness; (2) Back→re-create → the pre-render `/onboarding` guard (`Navigate to="/"` when the user has an org).
- **Email match is case-insensitive** (`strings.EqualFold` / `lower()`); never join on a mismatched email (403).
- **Invitee isn't a member** → endpoints live under `/api/auth/web/me/invites/*` (web-session auth), not `/api/orgs/:id/...`.
- **`InviteView` JSON tags** must be `invite_id`/`organization_id`/`org_name`/`role` (Task 1 struct + Task 3 type agree).
- **Deferred:** join-by-slug + admin review (2a-2); activation checklist (2a-3).
