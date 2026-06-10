# SaaS Phase 0 — Multi-tenancy + Web SSO foundation: design

**Status:** approved (brainstorm), ready for writing-plans.
**Date:** 2026-06-10
**Goal:** Lay the foundation for pivoting lead-cat from a single-tenant Telegram tool into a multi-tenant SaaS sold to other teams: reintroduce real multi-tenancy, add a web dashboard with SSO + magic-link sign-in, self-serve organization onboarding, and tenant-scoped access control.
**Topic:** First sub-project of the SaaS pivot. This is the **foundation** phase — it ships no calendar sync, availability, notifications routing, analytics, or billing. Those are later phases (see Decomposition below). Everything here exists so that calendars, members, and notifications have a tenant and an authenticated web user to attach to.

## Product pivot context

lead-cat is being repurposed (a deliberate pivot of the existing codebase, not a greenfield product) into a **unified, calendar-agnostic meeting-control SaaS for external teams**. The agreed product vision and its decomposition:

**Wedge (the reason a team buys):** a unified cross-calendar availability layer that actually sees everyone's busyness across Google Calendar, Microsoft 365/Teams, and internal meetings — solving "slots are often not free" independent of any single calendar.

**Surface:** web dashboard (primary) + Telegram (notification channel / quick actions).

**Decomposition into sub-projects (each its own brainstorm → spec → plan → execute cycle):**

| Phase | Ships | Depends on |
|---|---|---|
| **0. Tenancy + web auth (THIS SPEC)** | Multi-tenant org model, web SSO (Google + Microsoft) + magic-link, self-serve org onboarding + invites, web shell, tenant context. | — |
| 1. Calendar connection (hybrid, 2 providers) | `CalendarProvider` port; per-user OAuth connect for Google + MS Graph; domain-wide delegation path; encrypted token storage + refresh; "connect calendar" UI. | 0 |
| 2. Unified availability engine (wedge payoff) | Aggregate freebusy across all sources + internal meetings into one view; provider-agnostic; reuse pure `FreeSlots`/`Overlaps` core; web "find common free time" UI. | 1 |
| 3. Notification hub (Slack-like routing) | Configurable event→channel→recipient routing; reuse asynq dispatcher; Telegram + email channels. | 0 |
| 4. Analytics / ROI | Reports: meeting load, time/cost, ROI. | 1–3 |
| 5. Billing + SaaS packaging | Plans, subscription, tenant limits. | all |

Serialized order to a proven wedge: **0 → 1 → 2**, then 3, 4, 5.

## Locked decisions

| # | Decision |
|---|----------|
| 1 | **Scope = foundation only.** No calendar sync, availability, notification routing, analytics, billing. Phase 0 ends when a person can sign in on the web (3 methods), create or join an organization, and the app enforces per-org membership + roles. |
| 2 | **Tenant onboarding = self-serve signup.** Anyone signs in → if they belong to no org, they are sent to `/onboarding` to create one and become its `owner`. No domain-based auto-join (avoids public-domain and access-control problems; mixed/personal emails are first-class). |
| 3 | **Auth methods (3):** Google SSO, Microsoft SSO (both allow personal/consumer accounts — OIDC `common`/`consumers`, no tenant restriction), and **magic-link for any email** (yandex, mail.ru, corporate without Google/MS). Magic-link users can be members and receive notifications; they simply have no calendar to connect in Phase 1. No password auth. |
| 4 | **Provider order within the phase:** Google SSO lands first; the `SSOProvider` port is multi-provider from the start; Microsoft is the second task; magic-link is independent. |
| 5 | **Web session = server-side (`web_sessions` table).** httpOnly + Secure + SameSite cookie carries an opaque session id; the row holds `user_id`, `expires_at`, `revoked_at`, sliding renewal. Gives real logout/revocation and survives tab close (unlike the TMA `sessionStorage`). CSRF via double-submit cookie on mutations. |
| 6 | **Tenant table renamed `workspaces` → `organizations`** (and `workspace_members` → `organization_members`). Now is the cheapest moment to align names with product language. Migration renames tables + FKs + indexes; all repos/queries updated. |
| 7 | **`platform_users` = canonical account.** One account (keyed by email/`auth_sub`) belongs to many orgs via `organization_members`. Identity carries `auth_method` and `avatar_url`; `telegram_id` stays nullable for later Telegram unification. |
| 8 | **Telegram path may break.** Current single-tenant TMA/bot auth is allowed to temporarily diverge from the new model (user authorized this). The existing singleton workspace is migrated into "organization #1". Telegram is re-converged into multi-tenancy in a later phase. |
| 9 | **Email sender is a new dependency.** Magic-link requires an `EmailSender` port with a default SMTP implementation (provider/credentials via config). This is the only new external infra in Phase 0. |
| 10 | **Out of scope (Phase 0):** calendar OAuth, availability, notification routing, billing, password auth, SCIM/SAML enterprise SSO, org deletion/transfer, audit-log UI, Telegram↔web identity merge UI. |

## Reality check vs current `main` (HEAD `6ea054e`)

Mapped during brainstorm (Explore report). Key reusable assets vs build-new:

| Surface | Status | Phase 0 action |
|---------|--------|----------------|
| `workspaces` table (`id`, `slug`, `name`, `owner_user_id`, `tz`, `meet_link`, `notify_chat_id`, …) | Multi-workspace-shaped, **singleton-enforced** by `workspaces_singleton_idx` (partial unique index `WHERE name='Lead Cat'`). | Rename → `organizations`, **drop singleton index**, add `created_by_user_id`, `plan`. |
| `workspace_members` (`workspace_id`, `user_id`→platform_users, `telegram_username`, `role`) | Exists; roles `developer/admin/owner`. | Rename → `organization_members`; `user_id` NOT NULL; roles → `owner/admin/member`; add `invited_email` (pending). |
| `platform_users` (`id`, `auth_sub`, `email`, `telegram_id`, `avatar_url`) | Alive; used as meeting **organizer** identity via `EnsureMiniAppOrganizer`; `auth_sub` = `email:<addr>`. | Canonical web account. Add `auth_method` (`google`/`microsoft`/`magiclink`); keep organizer role. |
| `UserCanAccessWorkspace` (owner OR member) | `application/workspace_access.go`. | Basis for `RequireOrgMember` middleware. |
| `MiniAppToken` (HS256, claims, TTL, `tok_typ:"miniapp"`, `JWT_SECRET`) | `platform/auth/miniapp_token.go`. | **Not reused for web** (web uses opaque DB sessions). Stays for TMA. |
| Retired platform auth (`/api/auth/*`, `/api/me`, `/api/workspaces` → 410 `PlatformGone`) | `app.go:73-77`. | Web auth lives under **new** `/api/auth/web/*`; the 410 routes stay retired. |
| OAuth/OTP/passkey scaffolding | Mostly removed (passkeys dropped); `auth_sub` builder + `UpsertUserIdentity` remain. | Reuse `UpsertUserIdentity`/`sub.go` for SSO provisioning. |
| `admin_audit_log` (actor = bot_user + telegram_id + email) | `audit_repo.go`. | Extend actor to web `platform_user` (Phase 0 records auth + org/membership events). |
| Frontend: everything gated behind Telegram `initData`; routes under `/_miniapp`; `MiniAppAuthProvider` + sessionStorage | `frontend/src/...`. | **New web entry**: `/login`, `/onboarding`, web dashboard shell with org switcher + logout. TMA tree parked behind a surface detector. |
| Email sending | None (Telegram-only notifications). | New `EmailSender` port + SMTP impl. |
| Config (`JWT_SECRET`, `JWT_ISSUER`, encryption key) | `platform/config/config.go`. | Add: OAuth client id/secret + redirect (Google, MS), SMTP creds, web cookie domain, app base URL. |

## 1. Data model

New/changed migrations (timestamps assigned at writing-plans time):

- **Rename** `workspaces` → `organizations`, `workspace_members` → `organization_members`; rename FK columns `workspace_id` → `organization_id` across `meetings`, `employees`, `scenarios`, `scenario_runs`, `admin_audit_log`, etc.; rename indexes/constraints accordingly. **Drop** `workspaces_singleton_idx`.
- **`organizations`** add: `created_by_user_id UUID REFERENCES platform_users(id)`, `plan TEXT NOT NULL DEFAULT 'free'`. Keep `slug` UNIQUE but now generated per-org (not hardcoded).
- **`organization_members`**: `organization_id`, `user_id UUID NOT NULL REFERENCES platform_users(id)`, `role TEXT NOT NULL` (`owner`/`admin`/`member`), `invited_email TEXT NULL`, `created_at`. UNIQUE(`organization_id`, `user_id`).
- **`platform_users`** add: `auth_method TEXT` (`google`/`microsoft`/`magiclink`), keep `email` UNIQUE, `telegram_id` nullable.
- **`organization_invites`** (new): `id`, `organization_id`, `email`, `role`, `token` (random, hashed at rest), `expires_at`, `accepted_at NULL`, `created_by_user_id`, `created_at`.
- **`magic_link_tokens`** (new): `id`, `email`, `token_hash`, `purpose` (`login`), `expires_at`, `consumed_at NULL`, `created_at`. Short TTL (e.g. 15 min), single-use.
- **`web_sessions`** (new): `id` (the opaque cookie value, random; stored hashed), `user_id`, `created_at`, `last_seen_at`, `expires_at`, `revoked_at NULL`, `user_agent`, `ip`.

Data migration: the existing singleton workspace row becomes "organization #1"; its existing `workspace_members` become `organization_members` with mapped roles. No data loss.

## 2. Auth layer

### Ports (application-defined, infra-implemented)

- `SSOProvider`: `AuthURL(state, pkceChallenge) string`, `Exchange(ctx, code, pkceVerifier) (Profile, error)` where `Profile = {Email, Name, AvatarURL, Subject, Provider}`. Implementations in `infrastructure/oauth/google` and `infrastructure/oauth/microsoft` (OIDC, authorization-code + PKCE, `state` for CSRF). Microsoft uses the `common`/`consumers` authority so personal accounts work.
- `EmailSender`: `Send(ctx, to, subject, body) error`. Default SMTP implementation in `infrastructure/email/smtp`. Used for magic-link delivery.

### Routes (new group `/api/auth/web`)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/auth/web/:provider/start` | provider ∈ {google, microsoft}: set PKCE + state cookies, 302 to provider. |
| `GET` | `/api/auth/web/:provider/callback` | validate state, exchange code, provision identity, create session, 302 to `/onboarding` or dashboard. |
| `POST` | `/api/auth/web/magic/request` | body `{email}`: create single-use `magic_link_tokens` row, email the link. Always 204 (no account enumeration). |
| `GET` | `/api/auth/web/magic/verify` | query `token`: consume token, provision identity, create session, 302. |
| `POST` | `/api/auth/web/logout` | revoke current `web_sessions` row, clear cookie. |
| `GET` | `/api/auth/web/me` | current account + org memberships (for bootstrap). |

### Identity provisioning (shared by all 3 methods)

On successful auth, upsert `platform_users` by email (`auth_sub` = `email:<addr>`), set/refresh `auth_method`, `name`, `avatar_url`. Then create a `web_sessions` row and set the cookie. If the email matches a pending `organization_invites`, auto-accept (create `organization_members`, mark invite accepted). If the account belongs to ≥1 org → dashboard; else → `/onboarding`.

## 3. Web session

httpOnly + Secure + `SameSite=Lax` cookie holds the opaque session id (random; only the hash is stored in `web_sessions`). `WebAuth` middleware resolves the cookie → unrevoked, unexpired session → loads `platform_user` into `c.Locals("web_user")`; slides `expires_at`/`last_seen_at` on activity. `logout` sets `revoked_at`. CSRF: double-submit cookie token validated on all state-changing web routes (non-GET).

## 4. Tenant context

- The active org is carried per request via `X-Org-Id` header (SPA sets it from the org switcher); fallback to the user's only/last org.
- `RequireOrgMember` middleware (mounted after `WebAuth`) resolves `(web_user, org_id)` → `organization_members` row; 403 if not a member; stores membership (incl. role) in locals.
- Role gates: `owner` > `admin` > `member`. Existing admin endpoints migrate from `bot_user.role=="admin"` to the org-scoped membership role. **All org-scoped queries take `organization_id` explicitly** and are filtered by it (schema already FK'd; this phase makes the scoping explicit + enforced).

## 5. Onboarding & invites

- **Create org:** `POST /api/orgs` (authed, any web user) → name → creates `organizations` (slug derived) + `organization_members(owner)`; sets `created_by_user_id`.
- **Switch org:** client-side; `GET /api/auth/web/me` returns memberships; SPA sets `X-Org-Id`.
- **Invite:** `POST /api/orgs/:id/invites` (owner/admin) → email + role → `organization_invites` row + email with accept link. Acceptance happens implicitly on the invitee's next sign-in (email match) or explicitly via the link. `GET`/`DELETE` invites for management.
- Minimal member list: `GET /api/orgs/:id/members`, `PATCH .../members/:uid/role`, `DELETE .../members/:uid` (owner/admin; cannot remove/demote self as last owner).

## 6. Frontend (web shell)

- **Surface detection** on bootstrap: Telegram `initData` present → TMA path (parked, may be non-functional this phase); else → web path.
- **New routes:** `/login` (Google / Microsoft buttons + email field for magic-link), `/onboarding` (create org), authenticated dashboard shell with **org switcher**, account menu, logout. Routing reads `/api/auth/web/me`.
- **Web auth context** (separate from `MiniAppAuthProvider`): cookie-based (no bearer token in JS); 401 → redirect to `/login`. TanStack Query hooks for `me`, orgs, members, invites.
- The existing meetings/profile screens are **not** rewired in Phase 0 (no tenant-scoped reads shipped yet beyond auth/org). They re-converge starting Phase 1.

## 7. Testing

Repo convention: pure logic unit-tested; I/O build-verified; ≥1 HTTP integration test on a key auth route.

- **Unit:** PKCE challenge/verifier + `state` generation/validation; magic-link token issue/verify/consume + expiry/single-use; invite parse + email-match acceptance; org membership/role resolution + last-owner guard; slug derivation.
- **Build-verified:** `SSOProvider` Google/MS implementations (OAuth exchange), SMTP `EmailSender`, migrations, repos.
- **Integration (≥1):** magic-link `request → verify → session cookie → /me` round-trip against a test DB + fake `EmailSender`; and a `WebAuth` + `RequireOrgMember` 403/200 path test.

## 8. What breaks / migration

- `workspaces_singleton_idx` dropped → multiple orgs allowed. Existing row → "org #1"; its members mapped.
- Telegram auth (`/api/auth/miniapp`, `RequireBotAdmin`, TMA frontend gate) temporarily diverges from the new org model — acceptable this phase; re-converged later. Build stays green; TMA may be non-functional at runtime until a later phase.
- `admin_audit_log` actor extended to web users.

## 9. Open items for writing-plans

- Exact migration sequencing for the rename (single migration vs split) to keep `make migrate` reversible.
- SMTP provider choice + local dev story (e.g. Mailpit/Mailhog) for magic-link in `make dev`.
- New config keys (OAuth client id/secret/redirect ×2 providers, SMTP, `APP_BASE_URL`, cookie domain) and their dev defaults.
- Whether org switching uses `X-Org-Id` header (spec default) or a path segment `/orgs/:slug/...` — finalize at plan time; header chosen here for minimal routing churn.
