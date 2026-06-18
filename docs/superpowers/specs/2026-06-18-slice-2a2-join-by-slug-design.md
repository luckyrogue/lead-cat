# Slice 2a-2 — Join-by-Slug Requests + Admin Accept/Decline (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 2a (activation/onboarding)**, sub-slice **2**. Follows 2a-1.

## Epic context

2a-1 fixed the onboarding gate + added explicit invite accept/decline. 2a-2 adds the inverse
flow: a user can **request to join an organization by its slug**, and the org's admins
**accept or decline** the request. 2a-3 (the first-run activation checklist) follows.

## Goal

A logged-in user without (or with) an org can submit a join request by entering an org's slug;
the org's admins see pending requests and accept (→ membership, role `member`) or decline.

## Decisions (from brainstorming)

- **Request → admin accept/decline** (not open-join): a join request is created `pending`; an
  org admin must approve.
- **Accepted role = `member`.**
- **Already-a-member** on join-by-slug → idempotent success returning `{already_member,
  organization_id}` so the UI routes to the dashboard.
- **Admin review lives as a card on the existing Invites page** (`features/invites`), not a new
  nav item.
- **No admin notification** in 2a-2 (in-app list only; email/Telegram/badge deferred).
- **Dedupe:** one `pending` request per `(organization_id, user_id)` (partial unique index);
  re-request allowed after a decline.

## Background — verified current state

- Org routes (`app.go:100-117`): `orgs := app.Group("/api/orgs", webAuth.Middleware)`; `scoped
  := orgs.Group("/:id", RequireOrgMember(store))`; admin writes add
  `RequireOrgRole("admin")`. `RequireOrgMember` sets `org_member` local; `RequireOrgRole` checks
  `application.RoleAtLeast` (owner=3/admin=2/member=1, `application/org.go:34`).
- `GetOrganizationBySlug(ctx, slug) (Organization, error)` exists on `*Store`
  (`organization_repo.go:122-129`) but is NOT on the `Repository` interface and has no route.
- `GetOrgMember(ctx, orgID, userID) (Member, bool, error)` (`organization_repo.go:58-72`) — the
  canonical membership check (also on `Repository` via the middleware resolver).
- `AddMember(ctx, ...)` (`organization_repo.go:173`) inserts into `organization_members`.
- `CreateOrganizationForOwner` slugifies + appends a 6-hex suffix (`services.go:100-111`,
  `org.go:62-88`). `Organization` model has `Slug` (`model/model.go:29-37`).
- Admin members/invites UI: `features/members/pages/members-page.tsx`,
  `features/invites/pages/invites-page.tsx`; entity `entities/org/{types,api,queries}.ts`
  (e.g. `useInvites`/`useCreateInvite`). A "Join requests" admin card fits on the invites page.
- Onboarding `routes/onboarding.tsx` (post-2a-1): invites card + create-org card, guarded by
  `if (me.organizations.length > 0) return <Navigate to="/">`. A third "Join by slug" card fits
  between them.
- 2a-1 established the pattern for non-org-gated, web-session, self-service endpoints under
  `/api/auth/web/me/*` (e.g. `GET /me/invites`).

## Design

### A. Migration + repo

Migration `organization_join_requests`:
```sql
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
```
(`gen_random_uuid()` — confirm pgcrypto/pg13+ is available, matching existing tables; else
generate the UUID in Go like other repos.)

Repo methods on `*Store`:
- `CreateJoinRequest(ctx, orgID, userID uuid.UUID) error` — `INSERT ... ON CONFLICT DO NOTHING`
  against the partial unique index (idempotent while pending).
- `ListJoinRequestsForUser(ctx, userID uuid.UUID) ([]model.JoinRequestView, error)` — JOIN org
  for name; `JoinRequestView{OrganizationID, OrgName, Status}`.
- `ListPendingJoinRequests(ctx, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)` — JOIN
  platform_users for requester name/email; `JoinRequestAdminView{RequestID, UserID, Name,
  Email, CreatedAt}`; `WHERE status='pending'`.
- `AcceptJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error` — tx: load the pending
  request for this org `FOR UPDATE` (not-found → `model.IsNotFound`); `AddMember`-equivalent
  insert `ON CONFLICT DO NOTHING` (role `member`); `UPDATE status='accepted', decided_at,
  decided_by`.
- `DeclineJoinRequest(ctx, orgID, requestID, deciderID uuid.UUID) error` — `UPDATE
  status='declined', decided_at, decided_by WHERE id=$ AND organization_id=$ AND
  status='pending'`; 0 rows → `model.IsNotFound`.
- `GetOrganizationBySlug` promoted onto the `Repository` interface.

### B. Application

- `RequestToJoinBySlug(ctx, userID uuid.UUID, slug string) (JoinResult, error)`: trim/lower the
  slug; `GetOrganizationBySlug` (not found → `ErrOrgNotFound`); `GetOrgMember` — if already a
  member → `JoinResult{AlreadyMember:true, OrganizationID:org.ID}`; else `CreateJoinRequest` →
  `JoinResult{Pending:true, OrganizationID:org.ID}`.
- `ListMyJoinRequests(ctx, userID)`; `ListOrgJoinRequests(ctx, orgID)`;
  `AcceptJoinRequest(ctx, orgID, requestID, deciderID)`; `DeclineJoinRequest(...)`.
- `ErrOrgNotFound` sentinel in `model` (or reuse IsNotFound from the slug lookup → 404).
- New `Repository` interface methods for all of the above.

### C. HTTP endpoints

User-side (web-session auth, under the `web` group like 2a-1's `/me/invites`):
- `POST /api/orgs/join-requests` `{ "slug": "..." }` — body-parsed; caller from `web_user`.
  `200 {already_member:true, organization_id}` | `200 {status:"pending", organization_id}` |
  `404` (slug not found) | `400` (missing slug). *(Mounted on the `orgs` group's root, which is
  `webAuth.Middleware` but NOT `:id`/RequireOrgMember — a sibling of `POST /api/orgs`.)*
- `GET /api/auth/web/me/join-requests` → `[{organization_id, org_name, status}]`.

Admin-side (under `scoped` = `/api/orgs/:id` + `RequireOrgRole("admin")`):
- `GET /api/orgs/:id/join-requests` → `[{request_id, user_id, name, email, created_at}]`
  (pending only).
- `POST /api/orgs/:id/join-requests/:rid/accept` → `204` (membership created).
- `POST /api/orgs/:id/join-requests/:rid/decline` → `204`.
- Error mapping: not-found request → 404.

`JoinRequestView`/`JoinRequestAdminView`/`JoinResult` carry JSON tags matching the FE
(`organization_id`/`org_name`/`status`; `request_id`/`user_id`/`name`/`email`/`created_at`;
`already_member`/`status`/`organization_id`).

### D. Frontend

- **`entities/join-request`** (`types`/`api`/`queries`): user `requestToJoin(slug)` →
  `POST /api/orgs/join-requests`; `useMyJoinRequests` → `GET /me/join-requests`; admin
  `listOrgJoinRequests(orgId)`, `acceptJoinRequest(orgId, rid)`, `declineJoinRequest(orgId,
  rid)`.
- **Onboarding** (`routes/onboarding.tsx`): a third "Join by slug" card — slug input + submit.
  On `already_member` → `setActiveOrgId` + navigate `/`; on `pending` → show "Request sent,
  waiting for an admin" + the user's pending requests (`useMyJoinRequests`). On 404 → inline
  "organization not found".
- **Admin Invites page** (`features/invites/pages/invites-page.tsx`): a "Join requests" card
  listing pending requests (name/email) with Accept / Decline; accept/decline invalidate the
  requests query + members query.
- i18n keys (`onboarding.join.*`, `invites.requests.*`) in en/ru/kk (admin formal).

### E. Error handling / abuse

- Slug reveals an org exists — acceptable (slug-gated; admin must approve; harmless request).
- Re-submitting while pending is idempotent (partial unique index `ON CONFLICT DO NOTHING`).
- Accept is idempotent on membership (`ON CONFLICT DO NOTHING`).
- Admin endpoints are org-admin-gated; the user-side request endpoint validates the caller via
  session, creates only the caller's own request.

## Testing / verification

- **Repo** (testcontainers): create (idempotent while pending; re-request after decline ok);
  list-for-user (status + org name); list-pending-for-org (name/email); accept (membership +
  status); decline (status, no membership); GetOrganizationBySlug.
- **Application:** `RequestToJoinBySlug` (not found → ErrOrgNotFound; already-member →
  AlreadyMember; else pending).
- **Handlers** (httptest + fake repo): `POST /api/orgs/join-requests` (pending / already_member
  / 404 / 400); `GET /me/join-requests`; admin list/accept(204)/decline(204); admin endpoints
  reject non-admins (middleware-gated — assert the route is under `RequireOrgRole`).
- **Frontend:** admin typecheck/lint/build green; i18n parity en/ru/kk.
- `go test -race ./...` + `golangci-lint` clean (note: testcontainers repo tests skip if Docker
  is unavailable — verify SQL by inspection in review when so).

## Risks & mitigations

- **Repo tests need Docker** (may be unavailable). *Mitigation:* SQL reviewed by inspection;
  handler/app tests use fakes (run without Docker).
- **`gen_random_uuid()` availability.** *Mitigation:* confirm against existing migrations; else
  generate UUID in Go (match the existing repo pattern).
- **Mount path for the user request endpoint.** It must be a sibling of `POST /api/orgs` (web
  auth, not `:id`/member-gated). *Mitigation:* register on the `orgs` group root before/with the
  `scoped` subgroup.
- **No notification** means admins must visit the page. *Mitigation:* accepted for 2a-2; a
  badge/notification is a later concern.

## Done criteria

- `organization_join_requests` table + repo methods + `GetOrganizationBySlug` on `Repository`.
- User endpoints (`POST /api/orgs/join-requests`, `GET /me/join-requests`) + admin endpoints
  (list/accept/decline under `/api/orgs/:id/join-requests`, admin-gated).
- Onboarding "Join by slug" card (pending / already-member / not-found states); admin "Join
  requests" card with Accept/Decline on the Invites page; i18n en/ru/kk.
- `-race` + lint green (repo tests run when Docker present); admin typecheck/lint/build green.
- Admin notification + activation checklist (2a-3) explicitly deferred.
