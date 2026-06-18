# Slice 2a-1 — Onboarding Gate Fix + Explicit Invite Accept/Decline (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 2a (activation/onboarding)**, sub-slice **1**.

## Epic context

Track 1 (cross-calendar wedge: 1a Google OAuth + 1b Microsoft/Teams + 1c unified availability)
is complete. Track 2a is onboarding/activation, decomposed into:
- **2a-1 (this)** — fix the onboarding-gate bug + replace silent invite auto-accept with an
  explicit accept/decline screen.
- **2a-2** — join-by-slug requests + admin accept/decline (new join-request subsystem).
- **2a-3** — first-run activation checklist (connect calendar → invite → first meeting).

## Goal

An invited user reliably lands on a screen offering to **Accept or Decline** their pending
invite(s) (instead of being wrongly forced to create an organization), and a user who already
has an organization can no longer accidentally **re-create** one by navigating Back.

## The bug — root cause (verified from the code map)

1. **Invited user still sees "create org".** The gate `apps/admin/app/routes/_app.tsx:32-34`
   decides purely on `me.organizations.length === 0`. `useMe`
   (`apps/admin/app/shared/auth/use-me.ts:22-28`) caches with `staleTime: 60_000` and is **not
   invalidated on login**, so a stale "0 orgs" result routes a freshly-invited user to
   `/onboarding`. There is also **no pending-invite UI**, and `AcceptInvitesForEmail` failures
   at login are only `Warn`-logged (`web_auth.go:135,187`).
2. **Back → create again.** `/onboarding` is a **top-level route** (`routes.ts:17`), NOT nested
   under the `_app` membership gate; it only redirects via a *post-render* `useEffect`
   (`onboarding.tsx:63-67`), so the create form flashes / is reachable, and `POST /api/orgs`
   (`orgs.go:30-48`) has no "already has an org" guard.

## Decisions (from brainstorming)

- **Drop silent auto-accept** at login: invited users explicitly Accept/Decline. This both
  matches the desired UX and fixes the "wrongly forced to create-org" path.
- **Invite accept/decline lives under `/api/auth/web/me/invites/*`** (web-session auth,
  email-validated) — NOT the admin-gated `/api/orgs/:id/...` group, because the invitee is not
  yet an org member and cannot pass `RequireOrgMember`.
- **Server-side multi-org stays allowed** — the fix for accidental duplicates is a frontend
  route guard (a user owning multiple orgs may be legitimate; we only stop the *accidental*
  Back-reentry create).
- Invite state gains a **`declined_at`** column (pending = `accepted_at IS NULL AND declined_at
  IS NULL`).

## Background — verified current state

- `organization_invites` (`migrations/20260610150000_org_auth_tables.sql:6-17`): `id`,
  `organization_id`, `email`, `role`, `token_hash`, `expires_at`, `accepted_at` (NULL=pending),
  `created_by_user_id`, `created_at`. Partial index on `lower(email) WHERE accepted_at IS NULL`.
  **No `declined_at`.**
- `AcceptInvitesForEmail` (`application/services.go:80` → `postgres/org_invite_repo.go:67-127`):
  in a tx, for each pending invite matching `lower(email)`, `INSERT ... organization_members ON
  CONFLICT DO NOTHING` + `UPDATE ... accepted_at = now()`. Called at login in `WebAuthCallback`
  (`web_auth.go:135`) and `WebMagicVerify` (`:187`), errors `Warn`-logged.
- `WebMe` (`web_auth.go:212-238`) → `ListOrganizationsForUser` (JOIN members) — drives `/me`.
- `postLoginDest` (`web_auth.go:278-284`) redirects to `/` if the user has orgs, else (its
  current fallback). With auto-accept removed, an invited user has 0 orgs at login → lands on
  the onboarding route → sees pending invites.
- Existing admin invite endpoints under `scoped` (`/api/orgs/:id`, admin-gated): `GET/POST
  /invites`, `DELETE /invites/:iid`.
- Admin frontend: `_app.tsx` gate; `onboarding.tsx` (create form, `useCreateOrg`
  `entities/org/queries.ts:36-44` invalidates `meQueryKey`); `routes.ts:17` mounts onboarding
  top-level; `use-me.ts` `staleTime: 60_000`.
- Web-auth-protected routes (no org membership required) are registered near `app.go:86-89`
  (`/api/auth/web/me`, `/me/settings`) — the new invite endpoints mount here.

## Design

### A. Migration + repo

- Migration `add_invite_declined_at`: `ALTER TABLE organization_invites ADD COLUMN declined_at
  TIMESTAMPTZ;`. (Leave the existing partial index; "pending" queries add `AND declined_at IS
  NULL`.)
- Repo methods on `*postgres.Store`:
  - `ListPendingInvitesForEmail(ctx, email string) ([]model.InviteView, error)` —
    `organization_invites i JOIN organizations o ON o.id = i.organization_id WHERE
    lower(i.email)=lower($1) AND i.accepted_at IS NULL AND i.declined_at IS NULL AND
    i.expires_at > now()` → `[]InviteView{InviteID, OrganizationID, OrgName, Role}`.
  - `AcceptInvite(ctx, inviteID uuid.UUID, userID uuid.UUID, email string) error` — tx: load
    the invite `FOR UPDATE`; verify `lower(email) == lower(invite.email)`, still pending, not
    expired (else a typed error: `ErrInviteNotFound` / `ErrInviteEmailMismatch`); `INSERT
    organization_members ON CONFLICT DO NOTHING`; `UPDATE accepted_at = now()`.
  - `DeclineInvite(ctx, inviteID uuid.UUID, email string) error` — verify email + pending; set
    `declined_at = now()`.
- `model.InviteView{ InviteID, OrganizationID uuid.UUID; OrgName, Role string }`.

### B. Application layer

- `(s *Services) ListMyInvites(ctx, email string) ([]InviteView, error)` → repo.
- `(s *Services) AcceptInvite(ctx, inviteID, userID uuid.UUID, email string) error` → repo
  (returns the typed errors so the handler maps 404/403).
- `(s *Services) DeclineInvite(ctx, inviteID uuid.UUID, email string) error` → repo.
- **Remove** the `AcceptInvitesForEmail` calls from `WebAuthCallback` + `WebMagicVerify` (login
  no longer auto-joins). Keep identity upsert + session creation + `postLoginDest`. (Leave
  `AcceptInvitesForEmail` defined for now; it simply becomes unused — or delete if trivially
  unreferenced.)

### C. HTTP endpoints (web-session auth, email-validated)

Mounted in the web-auth-protected group (alongside `/api/auth/web/me`):
- `GET /api/auth/web/me/invites` → `200 [{invite_id, organization_id, org_name, role}]` (the
  caller's pending invites; email from `c.Locals("web_user").(model.PlatformUser)`).
- `POST /api/auth/web/me/invites/:iid/accept` → `204`; `ErrInviteNotFound`→404,
  `ErrInviteEmailMismatch`→403. On success the membership exists.
- `POST /api/auth/web/me/invites/:iid/decline` → `204` (same error mapping).
- (`POST /api/orgs` unchanged — multi-org allowed.)
- OpenAPI: these are consumed via the admin's axios client (not the generated client), so
  regen is optional; note in the plan.

### D. Frontend (admin)

- **Gate freshness:** make membership non-stale at the gate. Set `useMe` to
  `refetchOnMount: "always"` (or `staleTime: 0`) — or invalidate `meQueryKey` on entering the
  authed tree — so `_app.tsx` never acts on a stale "0 orgs". (Pick the minimal change;
  `refetchOnMount: "always"` on the `me` query is simplest.)
- **`/onboarding` route guard:** add a guard that redirects to `/` **before render** when the
  user already has an org — convert the `useEffect` redirect into a render-time early return
  (`if (me && me.organizations.length > 0) return <Navigate to="/" replace />`) so the create
  form is never shown / submittable to a user who already has an org. (Optionally nest
  `/onboarding` under a wrapper that enforces this; the early-return is sufficient.)
- **`entities/invite` (admin):** `types.ts` (`MyInvite{inviteId, organizationId, orgName,
  role}`), `api.ts` (`listMyInvites`, `acceptInvite(iid)`, `declineInvite(iid)`), `queries.ts`
  (`useMyInvites` + accept/decline mutations that invalidate `meQueryKey` + the invites query;
  on accept success → `setActiveOrgId(organizationId)` + navigate `/`).
- **Onboarding screen** (`onboarding.tsx`): above the create-org form, render a "You've been
  invited" section listing `useMyInvites()` results, each with **Accept** / **Decline**
  buttons. Empty invites → just the create form (today's UI). Accept → join + go to dashboard;
  Decline → remove from list.
- i18n: new keys (`onboarding.invites.*`, accept/decline, "invited to {org}") in en/ru/kk
  (admin formal).

### E. Error handling

- Email mismatch (invite belongs to a different address than the logged-in user) → 403, never
  joins. Expired/already-decided invite → 404 (filtered out of the list anyway). Accept is
  idempotent (`ON CONFLICT DO NOTHING` on membership).

## Testing / verification

- **Repo** (testcontainers): `ListPendingInvitesForEmail` (excludes accepted/declined/expired,
  joins org name); `AcceptInvite` (creates membership + sets accepted_at; email-mismatch →
  error; idempotent); `DeclineInvite` (sets declined_at; no membership).
- **Handlers** (httptest + fake/real services): `GET /me/invites` lists; accept → 204 +
  membership; accept with mismatched email → 403; decline → 204.
- **Login no longer auto-joins:** a test (or assertion) that `WebAuthCallback`/`WebMagicVerify`
  no longer create membership from invites (the auto-accept call is gone).
- **Frontend:** admin `typecheck`/`lint`/`build` green; new i18n keys parity en/ru/kk.
- `go test -race ./...` + `golangci-lint` clean.

## Risks & mitigations

- **Behavior change (no auto-accept).** Existing pending invites now require an explicit
  Accept. *Mitigation:* the onboarding screen surfaces them prominently; this is the intended
  UX. Users already members are unaffected.
- **Gate freshness vs. extra fetches.** `refetchOnMount: "always"` on `me` adds a refetch.
  *Mitigation:* `me` is small; correctness > a tiny request. Acceptable.
- **Email canonicalization.** Invite match is `lower(email)`; SSO/magic emails must compare
  case-insensitively. *Mitigation:* all comparisons use `lower()`; covered by repo tests.
- **Invitee not a member → can't use `/api/orgs/:id`.** *Mitigation:* endpoints live under
  `/api/auth/web/me/invites/*` (web-session auth only).

## Done criteria

- `declined_at` migration + the three repo methods + `InviteView` model.
- `/api/auth/web/me/invites` (list) + `:iid/accept` + `:iid/decline`; login no longer
  auto-joins; email-validated; correct 204/403/404.
- Admin gate uses fresh membership; `/onboarding` redirects pre-render when the user has an org
  (no Back-reentry create); onboarding screen shows pending invites with Accept/Decline.
- i18n en/ru/kk; backend `-race`+lint green; admin typecheck/lint/build green.
- Join-by-slug + admin review (2a-2) and the activation checklist (2a-3) explicitly deferred.
