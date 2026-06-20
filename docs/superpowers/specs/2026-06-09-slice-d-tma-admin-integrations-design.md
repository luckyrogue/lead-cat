# Slice D — TMA admin integrations: design

**Status:** approved (brainstorm), ready for writing-plans.
**Date:** 2026-06-09
**Goal:** Move setup operations (Google Calendar integration, chat link / members, scenarios) from curl + platform JWT into the Telegram Mini App under an admin-gated route group, with a persistent audit log.
**Topic:** ТЗ §6 admin functions slice 1 of N. Roadmap reference: [`2026-06-06-roadmap-to-beta-design.md`](2026-06-06-roadmap-to-beta-design.md) → Slice D.

## Locked decisions

The five brainstorm questions resolved as follows:

| # | Decision |
|---|----------|
| 1 | **Scope** — phases 1 + 2 + 3 (Google integration + chat/members + scenarios); supersedes the pre-monorepo TMA setup cutover draft (removed) which was written before `PatchIntegrations` landed Google support |
| 2 | **Google verify depth** — _real_: parse SA JSON → impersonate `subject` → `Calendars.Get(calendarID)`. No probe-insert / no side-effect events |
| 3 | **SA upload UX** — paste-only `<textarea>` with client-side JSON validation. No file input (TMA WebView edge cases) |
| 4 | **Audit log** — persistent `admin_audit_log` table + `GET /api/tma/admin/audit` endpoint. Whitelisted `details` keys per action to prevent secret leaks |
| 5 | **Workspace model** — single workspace. `EnsureSingleWorkspace` returns first by `created_at`, or creates one with defaults (`name='Lead Cat'`, `tz='Asia/Almaty'`, `meet_link=''`). Multi-workspace picker out of scope |

## Reality check vs prior draft

The pre-monorepo TMA setup cutover draft was authored before Google SA support landed in `PatchIntegrations`. Current `main` (HEAD `be3c330`) has:

| Surface | Status |
|---------|--------|
| `PatchIntegrations` HTTP endpoint | Accepts `google_sa_json`, `google_subject`, `google_calendar_id` via platform JWT (handlers.go:158-181) |
| `Services.SetGoogleConfig` | Encrypts SA JSON via `crypto.TokenCipher` (AES-256-GCM); default calendar `"primary"` |
| `crypto.TokenCipher` | Driven by `MASTER_ENCRYPTION_KEY` env var; single key per instance |
| `VerifyIntegrations` | Verifies VCS only (gitlab/github). **Google verification does not exist.** |
| `Provider.For()` cache | Hashes `enc \| subject \| calendar_id` → SA-key change auto-invalidates |
| `/api/tma/admin/*` route group | Does not exist |
| `admin` role | Set by `botreg` when `BOT_ADMIN_TELEGRAM_IDS` env contains user's TG ID |

This spec supersedes the 2026-06-05 doc; the older spec stays in the repo as historical record.

## 1. Surface (routes + middleware + audit)

### Route group

Mounted under `/api/tma/admin/*`. Two-layer middleware:

1. Existing TMA JWT middleware (sets `c.Locals("bot_user")`)
2. New `requireBotAdmin` — re-reads `c.Locals("bot_user").(postgres.BotUser).Role == "admin"` per request → `403 {"error":"forbidden"}` otherwise

Per-request re-check (not from JWT claims) ensures `role` revocation takes effect immediately.

### Phase 1 — Google integration

| Method | Path | Maps to |
|--------|------|---------|
| `GET` | `/api/tma/admin/workspace` | `EnsureSingleWorkspace` + `GetIntegrations` — status summary |
| `POST` | `/api/tma/admin/workspace` | `EnsureSingleWorkspace` — idempotent create |
| `GET` | `/api/tma/admin/integrations` | `GetIntegrations` — returns `IntegrationsView` without secrets (`has_google`, `google_subject`, `google_calendar_id` only) |
| `PATCH` | `/api/tma/admin/integrations` | `SetGoogleConfig` + `UpdateWorkspace(meet_link, tz)` — accepts `google_sa_json` (paste), `google_subject`, `google_calendar_id`, `meet_link`, `tz`. All optional, partial update |
| `POST` | `/api/tma/admin/integrations/verify` | **New** `VerifyGoogleIntegration` — parse → impersonate → `Calendars.Get` |

### Phase 2 — chat + members

| Method | Path | Maps to |
|--------|------|---------|
| `GET` | `/api/tma/admin/chat/status` | `ChatStatus` |
| `POST` | `/api/tma/admin/chat/link` | `LinkChat` |
| `GET` | `/api/tma/admin/members` | `ListMembers` |
| `POST` | `/api/tma/admin/members/sync-chat` | `SyncChatMembers` |

### Phase 3 — scenarios (toggle + run only)

| Method | Path | Maps to |
|--------|------|---------|
| `GET` | `/api/tma/admin/scenarios` | `ListScenarios` |
| `PATCH` | `/api/tma/admin/scenarios/:id` | `UpdateScenario` (**only `enabled`** — definition/name editing stays platform JWT) |
| `POST` | `/api/tma/admin/scenarios/:id/run` | `RunScenario` |
| `GET` | `/api/tma/admin/scenarios/:id/runs` | `ListRuns` |

### Audit

| Method | Path | Notes |
|--------|------|-------|
| `GET` | `/api/tma/admin/audit?limit=&action=&actor=` | `ListAuditEntries`; pagination by `created_at DESC`, max `limit=200` |

### Standardized error codes

`forbidden`, `unauthorized`, `validation_failed`, `google_sa_invalid`, `google_subject_invalid`, `google_calendar_not_accessible`, `google_api_disabled`, `workspace_not_found`, `audit_query_invalid`.

### Out of scope for Slice D

- Workspace create/delete (picker)
- Full scenario CRUD via TMA (only `enabled` + `run`)
- Members add/delete via TMA (only read + sync)
- Master encryption key rotation flow
- Audit retention/TTL automation (deferred to Slice H)

## 2. Backend changes

### New files

| File | Purpose |
|------|---------|
| `backend/internal/delivery/http/middleware/require_bot_admin.go` | `RequireBotAdmin(c *fiber.Ctx) error` — reads `c.Locals("bot_user")`, asserts `Role == "admin"` |
| `backend/internal/delivery/http/handlers/tma_admin.go` | Fourteen handlers (5 phase 1 + 4 phase 2 + 4 phase 3 + 1 audit GET) |
| `backend/internal/application/admin_workspace.go` | `EnsureSingleWorkspace(ctx) (uuid.UUID, error)` — SELECT-existing-or-INSERT-default with idempotency |
| `backend/internal/application/google_verify.go` | `VerifyGoogleIntegration(ctx, workspaceID) (*GoogleVerifyResult, error)` |
| `backend/internal/application/google_verify_test.go` | Error-code mapping unit tests (mock probe via interface) |
| `backend/internal/application/audit.go` | `Audit(ctx, action, targetKind, targetID, details)` helper with whitelist enforcement |
| `backend/internal/application/audit_test.go` | Details redaction test |
| `backend/internal/infrastructure/persistence/postgres/audit_repo.go` | `InsertAuditEntry`, `ListAuditEntries(filters, limit)` |
| `backend/internal/infrastructure/calendar/google/probe.go` | `Probe(ctx, saJSON, subject, calendarID) (*calendar.Calendar, error)` with sentinel errors |

### Modified files

| File | Change |
|------|--------|
| `backend/internal/delivery/http/app.go` | Register `tmaAdmin := tma.Group("/admin", middleware.RequireBotAdmin)` and all fourteen handlers |
| `backend/internal/application/services.go` | Hook `s.Audit(...)` into `SetGoogleConfig`, `LinkChat`, `SyncChatMembers`, `UpdateScenario`, `RunScenario`. Audit write failure must not roll back the mutation — log warn + continue. Add `audit auditWriter interface{InsertAuditEntry(ctx, entry)}` field via Store injection |

### Google verify — implementation sketch

```go
// application/google_verify.go
type GoogleVerifyResult struct {
    OK              bool   `json:"ok"`
    CalendarSummary string `json:"calendar_summary,omitempty"`
    TimeZone        string `json:"time_zone,omitempty"`
    AccessRole      string `json:"access_role,omitempty"`
}

func (s *Services) VerifyGoogleIntegration(ctx context.Context, workspaceID uuid.UUID) (*GoogleVerifyResult, error) {
    enc, subject, calendarID, err := s.Store.GetGoogleConfig(ctx, workspaceID)
    if err != nil { return nil, err }
    if len(enc) == 0 || subject == "" {
        return nil, ErrGoogleNotConfigured
    }
    saJSON, err := s.Cipher.Decrypt(enc)
    if err != nil { return nil, ErrGoogleSAInvalid }
    cal, err := googleprobe.Probe(ctx, saJSON, subject, calendarID)
    switch {
    case errors.Is(err, googleprobe.ErrJSONParse):
        return nil, ErrGoogleSAInvalid
    case errors.Is(err, googleprobe.ErrAPIDisabled):
        return nil, ErrGoogleAPIDisabled
    case errors.Is(err, googleprobe.ErrSubject):
        return nil, ErrGoogleSubjectInvalid
    case errors.Is(err, googleprobe.ErrCalendar):
        return nil, ErrGoogleCalendarNotAccessible
    case err != nil:
        return nil, err
    }
    return &GoogleVerifyResult{
        OK: true,
        CalendarSummary: cal.Summary,
        TimeZone: cal.TimeZone,
        AccessRole: cal.AccessRole,
    }, nil
}
```

```go
// infrastructure/calendar/google/probe.go
var (
    ErrJSONParse   = errors.New("sa_json_parse")
    ErrAPIDisabled = errors.New("calendar_api_disabled")
    ErrSubject     = errors.New("subject_impersonation")
    ErrCalendar    = errors.New("calendar_not_accessible")
)

func Probe(ctx context.Context, saJSON, subject, calendarID string) (*calendar.Calendar, error) {
    cfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
    if err != nil { return nil, fmt.Errorf("%w: %v", ErrJSONParse, err) }
    cfg.Subject = subject
    svc, err := calendar.NewService(ctx, option.WithHTTPClient(cfg.Client(ctx)))
    if err != nil { return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err) }
    cal, err := svc.Calendars.Get(calendarID).Context(ctx).Do()
    if err != nil {
        if isGoogleAPIDisabled(err)  { return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err) }
        if isImpersonationFail(err)  { return nil, fmt.Errorf("%w: %v", ErrSubject, err) }
        return nil, fmt.Errorf("%w: %v", ErrCalendar, err)
    }
    return cal, nil
}
```

`isGoogleAPIDisabled` / `isImpersonationFail` discriminate via the `*googleapi.Error` reason field (`accessNotConfigured`, `forbidden`+reason `domainPolicy`, etc.).

### Audit details whitelist

`Audit(ctx, action, targetKind, targetID, details map[string]any)` enforces whitelist per action:

| action | targetKind | Allowed `details` keys |
|--------|-----------|------------------------|
| `google_config_updated` | `workspace` | `subject`, `calendar_id`, `has_new_sa_json` (bool, **never** the raw value) |
| `google_verified` | `workspace` | `ok`, `calendar_summary`, `time_zone`, `error_code` |
| `chat_linked` | `workspace` | `chat_id`, `chat_title` |
| `members_synced` | `workspace` | `added`, `removed`, `unchanged` (counts) |
| `scenario_toggled` | `scenario` | `name`, `enabled` |
| `scenario_run_started` | `scenario` | `name`, `manual_run_id` |

Any key not in the whitelist is dropped + `log.Warn("audit_unexpected_key", ...)`. Protects against accidental secret leaks when new actions get added.

## 3. Frontend overlay

### New FSD slices

```
frontend/src/entities/admin/
├── api.ts              GET workspace, integrations, chat status, members, scenarios, audit
├── write-api.ts        POST workspace, PATCH integrations, POST verify, POST chat link, POST members sync, PATCH scenario, POST scenario run
├── mutations.ts        useUpdateIntegrations, useVerifyIntegrations, useLinkChat, useSyncMembers, useToggleScenario, useRunScenario
├── queries.ts          useWorkspaceStatus, useIntegrations, useChatStatus, useMembers, useScenarios, useAuditLog
├── types.ts            WorkspaceStatus, IntegrationsView, ChatStatus, Member, Scenario, AuditEntry, GoogleVerifyResult
└── constants.ts        ADMIN_AUDIT_PAGE_SIZE = 50, AUDIT_ACTION_LABELS

frontend/src/features/admin-setup/
├── pages/
│   └── admin-setup-page.tsx          // single overlay with all groups
├── components/
│   ├── integrations-section.tsx       // Phase 1
│   ├── sa-paste-input.tsx
│   ├── verify-result-card.tsx
│   ├── chat-link-section.tsx          // Phase 2
│   ├── members-section.tsx
│   ├── scenarios-section.tsx          // Phase 3
│   └── audit-log-section.tsx
└── lib/
    ├── sa-validate.ts
    └── sa-validate.test.ts
```

Route: existing `/profile/admin` page gains a "Setup" link → new route `/profile/admin/setup` rendering `admin-setup-page` with sections per phase. Existing `canAccessTmaAdmin` guard reused on the route loader.

### SA paste — heaviest UX piece

```tsx
// sa-paste-input.tsx
<Field label={t("googleSAJson")}>
  <textarea
    value={saText}
    onChange={(e) => setSaText(e.target.value)}
    spellCheck={false}
    autoCapitalize="off"
    rows={8}
    className="font-mono ... whitespace-pre overflow-auto"
    placeholder={'{\n  "type": "service_account",\n  "project_id": "...",\n  "private_key": "-----BEGIN PRIVATE KEY-----\\n...",\n  ...\n}'}
  />
  {validation.error && (
    <p className="text-tma-danger text-xs">{t(validation.errorKey)}</p>
  )}
</Field>
```

```ts
// lib/sa-validate.ts (pure, testable)
export type SAValidation =
  | { ok: true; clientEmail: string; projectID: string }
  | { ok: false; errorKey: I18nKey }   // saInvalidJson | saMissingFields | saNotServiceAccount

export function validateSAJson(text: string): SAValidation {
  if (!text.trim()) return { ok: false, errorKey: "saInvalidJson" }
  let obj: unknown
  try { obj = JSON.parse(text) } catch { return { ok: false, errorKey: "saInvalidJson" } }
  if (typeof obj !== "object" || obj === null) return { ok: false, errorKey: "saInvalidJson" }
  const o = obj as Record<string, unknown>
  if (o.type !== "service_account") return { ok: false, errorKey: "saNotServiceAccount" }
  const required = ["project_id", "private_key", "client_email", "token_uri"]
  const missing = required.filter((k) => typeof o[k] !== "string" || !o[k])
  if (missing.length) return { ok: false, errorKey: "saMissingFields" }
  return { ok: true, clientEmail: o.client_email as string, projectID: o.project_id as string }
}
```

Save button disabled until `validation.ok === true`. After save → auto-trigger verify → `<VerifyResultCard>` shows result with error-code-specific copy:

| Error code | RU copy (sample) |
|-----------|-------------------|
| `google_sa_invalid` | "Файл сервис-аккаунта повреждён или не от Google" |
| `google_subject_invalid` | "Не удалось имперсонировать `{subject}`. Проверь domain-wide delegation в Google Workspace" |
| `google_calendar_not_accessible` | "Календарь `{calendar_id}` недоступен для этого сервис-аккаунта" |
| `google_api_disabled` | "Calendar API выключен в GCP-проекте. Включи через console.cloud.google.com" |

### Phase 2/3 — thin tables

- **Chat** — card showing `linked / not linked` + paste chat_id form
- **Members** — read-only table (full_name, telegram_username, role) + "Sync from chat" button
- **Scenarios** — list with toggle per row + "Run now" button (triggers `useRunScenario`, toast on success)

### Audit log

`components/audit-log-section.tsx`: list of cards by `created_at DESC`, action/actor dropdown filters, page-based pagination (50 entries/page). Per-action icon and color.

### i18n keys

~90 new keys × 3 langs (ru/kk/en) ≈ 270 entries: `adminSetup`, `adminIntegrations`, `googleSAJson`, `googleSubject`, `googleCalendarID`, `meetLinkLabel`, `timezoneLabel`, `saInvalidJson`, `saNotServiceAccount`, `saMissingFields`, `verifyButton`, `verifying`, `verifyOK`, `googleSaInvalidErr`, `googleSubjectInvalidErr`, `googleCalendarNotAccessibleErr`, `googleApiDisabledErr`, `chatLink`, `chatLinked`, `chatNotLinked`, `chatIDLabel`, `linkChat`, `members`, `syncFromChat`, `synced`, `scenarios`, `scenarioRun`, `auditLog`, `auditAction`, `auditActor`, `auditTime`, + 6 action-name labels.

### Tests

- `lib/sa-validate.ts` — vitest unit (valid JSON, missing fields, wrong type, malformed JSON)
- Other components — no unit tests (project convention is `pnpm typecheck` + `pnpm build`)

## 4. Migration + OpenAPI

### Migration

`backend/migrations/20260609120000_admin_audit_log.sql`:

```sql
-- +goose Up
CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NOT NULL REFERENCES bot_users(id),
    actor_telegram_id BIGINT NOT NULL,
    actor_email TEXT NOT NULL,
    action TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_log_created_at_idx ON admin_audit_log (created_at DESC);
CREATE INDEX admin_audit_log_actor_idx ON admin_audit_log (actor_user_id, created_at DESC);
CREATE INDEX admin_audit_log_action_idx ON admin_audit_log (action, created_at DESC);

CREATE UNIQUE INDEX workspaces_singleton_idx ON workspaces ((true)) WHERE name = 'Lead Cat';

-- +goose Down
DROP INDEX IF EXISTS workspaces_singleton_idx;
DROP TABLE IF EXISTS admin_audit_log;
```

Denormalized `actor_email` + `actor_telegram_id` are immutable audit history (preserves what was true at action time; Slice E may mutate `bot_users.email`).

`workspaces_singleton_idx` enforces "one Lead Cat workspace" at the DB layer — see §5.3.

### OpenAPI

Both `backend/openapi/openapi.json` and `backend/docs/openapi.json` (byte-identical mirror, slice A/B convention).

**New paths (fourteen endpoints):**

```
/api/tma/admin/workspace                  GET, POST
/api/tma/admin/integrations               GET, PATCH
/api/tma/admin/integrations/verify        POST
/api/tma/admin/chat/status                GET
/api/tma/admin/chat/link                  POST
/api/tma/admin/members                    GET
/api/tma/admin/members/sync-chat          POST
/api/tma/admin/scenarios                  GET
/api/tma/admin/scenarios/{id}             PATCH
/api/tma/admin/scenarios/{id}/run         POST
/api/tma/admin/scenarios/{id}/runs        GET
/api/tma/admin/audit                      GET
```

**New component schemas:**

| Schema | Fields |
|--------|--------|
| `TmaAdminWorkspaceStatus` | `id, name, tz, meet_link, has_google, google_subject, google_calendar_id, has_chat, chat_id?, chat_title?` |
| `TmaAdminIntegrationsPatchRequest` | `google_sa_json?, google_subject?, google_calendar_id?, meet_link?, tz?` (all optional) |
| `TmaAdminGoogleVerifyResult` | `ok, calendar_summary?, time_zone?, access_role?` |
| `TmaAdminChatStatus` | `linked, chat_id?, chat_title?` |
| `TmaAdminChatLinkRequest` | `chat_id, chat_title?` |
| `TmaAdminMember` | `id, full_name, telegram_username, role, github_login?, gitlab_login?` |
| `TmaAdminScenario` | `id, name, enabled, schedule, last_run_at?` |
| `TmaAdminScenarioPatchRequest` | `enabled` (only) |
| `TmaAdminScenarioRun` | `id, scenario_id, status, started_at, finished_at?, error?` |
| `TmaAdminAuditEntry` | `id, actor_email, actor_telegram_id, action, target_kind, target_id, details, created_at` |
| `TmaAdminAuditListResponse` | `entries: TmaAdminAuditEntry[], next_cursor?` |

All endpoints carry `security: [{ bearerAuth: [] }]` + tag `tma-admin`.

**Error responses (every endpoint):** `400 validation_failed`, `401 unauthorized`, `403 forbidden`, `500 internal_error`. Phase 1 additions: `409 workspace_not_found`; `/verify` also: `400 google_sa_invalid | google_subject_invalid | google_calendar_not_accessible | google_api_disabled`.

### Frontend schema regen

`pnpm openapi:generate` → regenerates `frontend/src/shared/api/generated/schema.ts`. Pattern from slice A/B: `entities/admin/types.ts` defines domain types, `api.ts` uses generated DTOs.

## 5. Operational concerns

### 5.1. SA JSON encryption lifecycle

Current: `MASTER_ENCRYPTION_KEY` env var → `crypto.NewTokenCipher` → AES-256-GCM. One key per instance. Per-encrypt 12-byte nonce stored inline.

**Slice D does NOT introduce:**
- Key rotation (two versions concurrent)
- KMS-backed storage
- Bulk re-encryption

**Slice D documents in `docs/OPERATIONS.md`:**

| Scenario | Current behavior | Operator action |
|----------|------------------|-----------------|
| `MASTER_ENCRYPTION_KEY` leak | All SA JSON in DB compromised | Generate new key → revoke SA private key in GCP → re-upload SA JSON via TMA admin → swap env var → redeploy |
| Key loss (env wiped) | All SA JSON unrecoverable | Re-upload SA JSON via TMA admin; GCP SA itself remains valid |
| Rolling SA key in GCP | Provider cache auto-invalidates on hash change | Upload new SA JSON via admin → Verify |

Key rotation with downtime is the documented recipe; zero-downtime rotation is post-beta backlog.

### 5.2. Audit log retention

Slice D writes without automatic GC. Estimate: ~10 admin actions/week per active tenant → ~100 KB/year. Slice H may add an asynq cron `purge_audit_log` with retention = 365 days if data grows beyond 1 GB.

### 5.3. Multi-instance safety

The service runs single-instance under Dokploy. Endpoints are still designed multi-instance-safe:

| Operation | Concurrency posture |
|-----------|---------------------|
| `EnsureSingleWorkspace` | Race possible (two concurrent POST → two workspaces). Mitigation: `workspaces_singleton_idx` partial unique on `name='Lead Cat'` + `INSERT ... ON CONFLICT DO NOTHING` followed by SELECT |
| `SetGoogleConfig` | Last-write-wins (single-row UPDATE). Fine for admin flow |
| `LinkChat`, `SyncChatMembers` | Same pattern — last-write-wins |
| `VerifyGoogleIntegration` | Read-only |
| `UpdateScenario` (enabled) | Last-write-wins |
| `RunScenario` | Already serialized via asynq with job-id dedup |
| `Audit` insert | INSERT-only, no conflict |

### 5.4. Observability — logs

| Level | When |
|-------|------|
| `Info` | `admin_action` (action, actor_email_hashed, target_id, has_diff) — parallel to audit table |
| `Info` | `google_verify_attempt` (workspace_id, subject_hash, calendar_id, result_code) |
| `Warn` | `audit_write_failed` (action, error) — never blocks the main operation |
| `Warn` | `google_verify_unexpected_error` (workspace_id, error) — probe returned non-sentinel |
| `Error` | `admin_handler_panic` — recovered |

`actor_email_hashed = SHA-256(email)[:8]` — greppable per admin without PII in logs. Plaintext `actor_email` lives only in the audit table.

### 5.5. Metrics

Two new counters in existing `metrics.go`:

- `admin_action_total{action="...",result="ok|error"}`
- `google_verify_total{result="ok|sa_invalid|subject_invalid|calendar_not_accessible|api_disabled"}`

Dashboard work deferred to Slice H.

### 5.6. Failure modes

| Failure | User visible | Logs | Operator action |
|---------|-------------|------|-----------------|
| Google API down | `verify` → 503 google_api_unreachable | Warn + GoogleAPIError | Retry |
| `Cipher.Decrypt` fails (key rotated) | `verify` → 400 google_sa_invalid | Error `decrypt_failed` | Re-upload SA via admin UI |
| Postgres down | 500 on all admin endpoints | Error (existing) | Existing infra alerting |
| Audit INSERT fails | Mutation passes, audit skipped | Warn `audit_write_failed` | Manual log review |
| Telegram bot not running | `chat/*` → 500 | Existing | Restart bot worker |

## 6. Risks + open questions

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Verify false-positive (impersonation passes; event create later fails) | Med | Verify uses same `Calendars.Get` path as `Provider.For()`. If that passes, `Events.Insert` near-certainly passes. We explicitly chose no probe-insert in §Q2 |
| SA JSON paste → OS clipboard → cache leak | Low-Med | Notice above field "Очистите буфер обмена после сохранения". Textarea has no `name`/`autocomplete` to suppress form-history |
| Partial unique index unsupported by DB driver | Low | Postgres ≥ 9.2 supports them; Lead Cat uses pg 15 ✓ |
| Audit table unbounded growth | Low | Estimate 100 KB/year per tenant; Slice H will add TTL if needed |
| Old curl clients keep hitting `/api/workspaces/:id/integrations` after D ships | Med | Intentional. Mark deprecated in OpenAPI (Slice E or G `"deprecated": true`); remove after first beta release |
| Overcrowded overlay (3 phases on one screen) | Med | Use existing `SettingsGroup` as visual separator per phase. If beta testers complain → trivial split into sub-routes |
| `scenario_executor` breaks on manual run with corrupt scenario | Low | Existing; Slice D adds nothing new. Fiber recovers handler panics by default |
| Race: admin revokes own role mid-session | Low | Middleware re-checks role per request; next API call returns 403; frontend redirects to `/profile` |

### Open questions (resolved here)

1. **Non-admin opens `/profile/admin/setup`?** → API returns 403 → routing-level redirect to `/profile` + toast "только админ" via existing `canAccessTmaAdmin` route loader guard
2. **DB already has multiple workspaces (from curl history)?** → `EnsureSingleWorkspace` returns first by `created_at`; UI shows its name. Slice E adds picker if needed
3. **Test coverage?**
   - Pure unit: `validateSAJson`, `audit` redaction, `google_verify` error-code mapping (no network)
   - Backend integration: none (project convention)
   - Frontend: only `validateSAJson` unit
4. **Kazakh (`kk`) copies for error codes?** → Yes, all 90 keys × 3 langs (slice A/B convention)
5. **Audit details — diff snapshot ("before → after")?** → No, only final-value (`{calendar_id: "new-id"}`). Diff over-engineered; reconstructible from git + audit timestamp

## 7. File structure (final)

```
backend/
├── migrations/
│   └── 20260609120000_admin_audit_log.sql               [NEW]
├── internal/
│   ├── delivery/http/
│   │   ├── app.go                                       [MODIFY: register /api/tma/admin/* group]
│   │   ├── middleware/
│   │   │   └── require_bot_admin.go                     [NEW]
│   │   └── handlers/
│   │       └── tma_admin.go                             [NEW: 14 handlers]
│   ├── application/
│   │   ├── services.go                                  [MODIFY: hook audit into 5 mutations]
│   │   ├── admin_workspace.go                           [NEW: EnsureSingleWorkspace]
│   │   ├── google_verify.go                             [NEW: VerifyGoogleIntegration]
│   │   ├── google_verify_test.go                        [NEW: error-code mapping]
│   │   ├── audit.go                                     [NEW: Audit helper + whitelist]
│   │   └── audit_test.go                                [NEW: details redaction]
│   └── infrastructure/
│       ├── calendar/google/
│       │   └── probe.go                                 [NEW: Probe with sentinel errors]
│       └── persistence/postgres/
│           └── audit_repo.go                            [NEW: InsertAuditEntry, ListAuditEntries]
├── openapi/openapi.json                                 [MODIFY: 14 paths + 11 schemas]
└── docs/openapi.json                                    [MODIFY: byte-identical mirror]

frontend/
├── src/
│   ├── shared/api/generated/schema.ts                   [REGEN]
│   ├── shared/tma/i18n.ts                               [MODIFY: ~90 keys × 3 langs]
│   ├── entities/admin/                                  [NEW slice]
│   │   ├── api.ts
│   │   ├── write-api.ts
│   │   ├── mutations.ts
│   │   ├── queries.ts
│   │   ├── types.ts
│   │   └── constants.ts
│   ├── features/admin-setup/                            [NEW slice]
│   │   ├── pages/admin-setup-page.tsx
│   │   ├── components/
│   │   │   ├── integrations-section.tsx
│   │   │   ├── sa-paste-input.tsx
│   │   │   ├── verify-result-card.tsx
│   │   │   ├── chat-link-section.tsx
│   │   │   ├── members-section.tsx
│   │   │   ├── scenarios-section.tsx
│   │   │   └── audit-log-section.tsx
│   │   └── lib/
│   │       ├── sa-validate.ts
│   │       └── sa-validate.test.ts
│   ├── features/profile/pages/admin-panel-page.tsx      [MODIFY: link to admin-setup]
│   └── routes/_tma/profile.admin.setup.tsx              [NEW: TanStack route]

docs/
├── API.md                                               [MODIFY: 14 new TMA paths + deprecation note]
├── MEETINGS.md                                          [MODIFY: Done row for admin setup]
└── OPERATIONS.md                                        [MODIFY: key rotation recipe]
```

## 8. Effort estimate

| Segment | New LoC | Time |
|---------|---------|------|
| Backend phase 1 (Google + admin middleware + audit) | ~700 | 2 days |
| Backend phase 2 (chat + members thin layer) | ~150 | 0.5 day |
| Backend phase 3 (scenarios thin layer) | ~120 | 0.5 day |
| Migrations + OpenAPI | ~250 | 0.5 day |
| Frontend entities + features + i18n | ~1100 | 2 days |
| Docs + smoke | ~150 | 0.5 day |
| **Total** | **~2500** | **6 days (~1.5 weeks)** |

## 9. Self-review

| Check | Status |
|-------|--------|
| Placeholder scan (TBD/TODO/vague reqs) | None |
| Internal consistency (architecture matches feature descriptions) | OK — routes, handlers, services, persistence all line up |
| Scope check (single implementation plan ≠ decomposition) | OK — three phases, one branch, ~2500 LoC; same shape as Slice A and B |
| Ambiguity check (any req with two readings) | Resolved inline; see open questions §6 |

## 10. Next step

Hand off to `writing-plans` skill to produce the task-by-task implementation plan at `docs/superpowers/plans/2026-06-09-slice-d-tma-admin-integrations.md`.
