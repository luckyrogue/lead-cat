> **Superseded paths:** implemented under `frontend/src/features/*`, `shared/api`, `features/auth`. See `frontend/README.md`.

# TMA Write Paths — Design (frontend integration, sub-project 3)

**Status:** approved (brainstorm), ready for implementation plan.
**Part of:** "wire the meetings Mini App to the backend" — (1) TMA auth & identity [done] → (2) read paths [done] → **(3) write paths [this spec]** → (4) auto/profile tabs.
**Spec source (ТЗ):** `docs/NEW-FEATURES.md` §4 (meeting create/edit/delete), §4.7 (conflict warning). Feature status: `docs/MEETINGS.md`. Prior slice: `docs/superpowers/specs/2026-06-01-tma-read-paths-design.md`.

## Goal

Let the authenticated Telegram user **create**, **edit**, and **delete** their own meetings from the Mini App, with a **real cross-participant conflict warning** on create — replacing the optimistic-only React Query cache writes left in place by sub-project 2. Writes reuse the existing application commands (`CreateMeeting` / `UpdateMeeting` / `CancelMeeting`) unchanged, bridging the two identity worlds (TMA `bot_users` → meeting-owning `platform_users`) with a lazy find-or-create link.

## Decisions (locked during brainstorming)

1. **Organizer = lazily linked `platform_users`.** TMA users live in `bot_users` (telegram_id ↔ email ↔ role), but every meeting command takes a `platform_users` UUID as organizer and authorizes via `ownerOrOrganizer`. A new application helper `EnsureTMAOrganizer(ctx, email, telegramID) (uuid.UUID, error)` find-or-creates the `platform_users` row by email using the existing `auth_sub="email:<email>"` convention (`UpsertUserIdentity`) and sets `telegram_id` (`LinkTelegram`). It is **idempotent** and **unifies** with native email-OTP login (same `auth_sub`), so a user who also logs in on the web resolves to the same row — and a meeting they organized on the web is editable from the Mini App, and vice-versa. Every write command is then reused **unchanged**.

2. **Create target workspace = the single Google-configured workspace.** Reuse `Store.ListWorkspacesWithGoogle(ctx)` (= `google_sa_json_enc IS NOT NULL`, the same selector CSV seeding uses): exactly one → use it; zero → `400 meetings_not_configured`; more than one → use the first (documented limitation; multi-workspace TMA targeting is out of scope). Edit/delete take the workspace **from the meeting itself** (via `ListEditableMeetings`, which returns `WorkspaceID`) — no global resolution needed.

3. **Scope narrowed to non-recurring create + single-meeting edit/delete.** The wizard collects `rec`/`recDays` but **no `recurrence_until`**, which `CreateMeeting` requires for `rec != once`; and the wizard's `custom`/`recDays` option has no backend recurrence equivalent. So this slice ships **once-only create**. Edit maps to `UpdateMeeting` (single occurrence — the Mini App has no this/whole-series scope selector). Delete maps to `CancelMeeting` (single). **Deferred:** recurring create, whole-series edit/delete, reminder settings (→ sub-project 4).

4. **Edit threads the meeting id through the wizard.** Today "edit" calls `openCreate(detailToDraft(detail))` and `completeCreate` always **creates a new** meeting (a mock limitation). Real edit must carry the source meeting `id` so `onComplete` chooses `PATCH` vs `POST`. The `OverlayState.create` variant gains an optional `editId`, threaded into `CreateWizard` and back out via the complete callback.

5. **Conflict warning uses the real backend.** The wizard's review step currently runs a client-side overlap check against only the user's own loaded meetings. Replace it with `POST /api/tma/conflicts` (global-by-email, the `MeetingConflicts` core from §4.7). Non-blocking: conflicts are shown but the user may still confirm (matches §4.7.3 bot behavior and the existing wizard's "proceed anyway").

6. **Writes invalidate, not optimistically patch.** Create/edit/delete become React Query `useMutation`s that, on success, `invalidateQueries(["tma","meetings","all"])` (and the colleague-schedule keys are left alone — they're other people's data). This replaces the `queryClient.setQueryData` cache surgery in `tma-app.tsx`. The success animation (`SuccessView` / paw burst / toast) is preserved.

## Codebase facts (verified)

- **Module path:** `github.com/luckyrogue/lead-cat`.
- **TMA route group:** `tma := app.Group("/api/tma", tmaAuth.Middleware)` (`delivery/http/app.go:149`), sets `c.Locals("bot_user").(postgres.BotUser)` (`TelegramID int64, FullName, Email, Role string`). New write routes register here alongside the read ones (`app.go:150-154`).
- **Read-handler conventions** (`handlers/tma_read.go`): receiver `*API`; `a.App` is `*application.Services`; `botUserEmail(c) (string, bool)` reads the email from `c.Locals("bot_user")`; `almatyLoc()` helper; `a.toMeetingDTO(ctx, m)` already maps a `postgres.Meeting` → `tmaMeetingDTO`. Reuse all of these.
- **Write commands (reused unchanged):**
  - `CreateMeeting(ctx, workspaceID, organizerID uuid.UUID, in CreateMeetingInput) (postgres.Meeting, error)` (`meeting_service.go:58`). `CreateMeetingInput{Dept, Type, Host, Date "YYYY-MM-DD", Start/End "HH:MM", Recurrence, RecurrenceUntil "YYYY-MM-DD" (required when Recurrence != once), Description, Participants []postgres.MeetingParticipant}`. Loads workspace TZ, builds the Google event, persists, enqueues `meeting:created`. Returns `ErrInvalidInput` on bad input.
  - `UpdateMeeting(ctx, workspaceID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (postgres.Meeting, error)` (`meeting_service.go:241`). `UpdateMeetingInput` = all-pointer overrides (`Dept,Type,Host,Date,Start,End,Recurrence,Description *string`; date/start/end are a unit). Authorizes via `ownerOrOrganizer(w, cur.OrganizerUserID, userID)` → `ErrForbidden`.
  - `CancelMeeting(ctx, workspaceID, userID, id uuid.UUID) error` (`meeting_service.go:297`). Authorizes via `ownerOrOrganizer`; idempotent; emits `meeting:cancelled` DM. (REST delete already routes through it.)
  - `ListEditableMeetings(ctx, telegramID int64) ([]postgres.MeetingWithTZ, error)` (`meeting_service.go:293`) — `WHERE pu.telegram_id=$1 AND status='scheduled' AND starts_at>now()`; each row carries `WorkspaceID`. Used to resolve a meeting's workspace + assert the caller may edit it before calling Update/Cancel.
  - `MeetingConflicts(ctx, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)` (`conflict.go:46`) — global-by-email; `excludeMeetingID = uuid.Nil` for create.
- **Identity bridge primitives:**
  - `Store.UpsertUserIdentity(ctx, authSub, email, phone string) (User, error)` (`auth_repo.go:19`).
  - `platformauth.SubEmail(email) string` → `"email:" + lower(trim(email))` (`platform/auth/otp.go:99`).
  - `(*Services).LinkTelegram(ctx, userID uuid.UUID, telegramID int64, username string) error` (`services.go:33`) → `Store.LinkTelegram` + member backfill. (Pass `username=""` — TMA has no username here.)
  - `Store.ListWorkspacesWithGoogle(ctx) ([]uuid.UUID, error)` (`employee_repo.go:73`).
- **Frontend write surface** (`frontend/src/`):
  - `features/tma/tma-app.tsx` — `TmaContent` holds `meetings = useMyMeetings("all")` + `useQueryClient()`. `completeCreate` (line 160) does `draftToMeeting` + `setQueryData(["tma","meetings","all"], …)`; `deleteMeeting` (183) filters the cache; edit reopens the wizard via `openCreate(detailToDraft(detail))` (290). `CreateWizard` is rendered at line 260 with `initial`, `meetings`, `onComplete`.
  - `features/tma/screens/create-wizard.tsx` — `STEPS = [what, when, who, review]`; `onComplete({...draft, end: endTime})` on the final step (line 304); uses mock `ME` for host default (271) + `finalMeeting.organizer = ME.email` (346); the review step computes `conflictPeople` client-side (310-328) and renders the warning (867).
  - `shared/tma/types.ts` — `MeetingDraft{dept,type,host,date,start,dur,rec,recDays,participants:Employee[],desc,end?}`; `OverlayState` create variant `{type:"create"; initial?:Partial<MeetingDraft>}`.
  - `shared/tma/meeting-utils.ts` — `draftToMeeting(draft,id)` (sets `organizer: ME.email`), `detailToDraft(m)`.
  - `shared/tma/api.ts` + `queries.ts` — existing fetchers/hooks + DTO type `MeetingDTO`; `useMyMeetings(scope)` key `["tma","meetings",scope]`.
  - `shared/tma/auth-context.tsx` — `useTmaAuth().user` (the real `{telegramId,name,email,role}`) is the replacement for mock `ME`.
- Conventions: backend pure logic unit-tested, handlers/wiring build-verified (no HTTP/DB harness); Go run as `env -u GOROOT go ...` from `backend/`; gate `make test && make lint && make build` from repo root (`make lint` = golangci-lint incl. gofmt). Frontend: no test runner — `pnpm -C frontend typecheck` + `pnpm -C frontend build` + `pnpm -C frontend format`. No secrets/initData/JWT/PII in logs. Never touch `frontend/vite.config.ts`.

## Architecture

Thin TMA write handlers that (a) bridge identity once via `EnsureTMAOrganizer`, (b) resolve the workspace, (c) delegate to the existing command, (d) return the same `tmaMeetingDTO` the read path uses. The frontend swaps optimistic cache writes for real mutations + invalidation, threads an `editId` through the wizard, and calls a real conflicts endpoint on the review step.

### Backend — new endpoints (under the `/api/tma` group, TMA-auth)

| Method & path                  | Reuses                                                     | Returns                             |
| ------------------------------ | ---------------------------------------------------------- | ----------------------------------- |
| `POST /api/tma/meetings`       | `EnsureTMAOrganizer` + workspace-resolve + `CreateMeeting` | `201 {meeting: tmaMeetingDTO}`      |
| `PATCH /api/tma/meetings/:id`  | `EnsureTMAOrganizer` + meeting→workspace + `UpdateMeeting` | `200 {meeting: tmaMeetingDTO}`      |
| `DELETE /api/tma/meetings/:id` | `EnsureTMAOrganizer` + meeting→workspace + `CancelMeeting` | `204` (no body)                     |
| `POST /api/tma/conflicts`      | `MeetingConflicts`                                         | `200 {conflicts: tmaConflictDTO[]}` |

All read the authed `bot_user` from `c.Locals("bot_user")`.

**`EnsureTMAOrganizer` (new, `application/`):**

```go
// EnsureTMAOrganizer find-or-creates the platform_users row backing a TMA user
// (by email, via the email:<email> auth_sub convention) and links the telegram id.
// Idempotent; returns the platform_users UUID used as meeting organizer.
func (s *Services) EnsureTMAOrganizer(ctx context.Context, email string, telegramID int64) (uuid.UUID, error) {
    u, err := s.Store.UpsertUserIdentity(ctx, platformauth.SubEmail(email), email, "")
    if err != nil { return uuid.Nil, err }
    if err := s.LinkTelegram(ctx, u.ID, telegramID, ""); err != nil { return uuid.Nil, err }
    return u.ID, nil
}
```

**Create handler flow:**

1. Read `bot_user` (email, telegram id). 2. Parse body (`tmaCreateRequest`, below); reject `recurrence != "" && != "once"` → `400 meetings_recurring_unsupported`. 3. `EnsureTMAOrganizer`. 4. Resolve workspace via `ListWorkspacesWithGoogle` (zero → `400 meetings_not_configured`; else first). 5. Map body → `CreateMeetingInput` (host defaults to `bot_user.FullName` when empty; participants → `[]postgres.MeetingParticipant{{Email: …}}`). 6. `CreateMeeting`; map `ErrInvalidInput` → `400`. 7. `a.toMeetingDTO` → `201`.

**Edit / delete handler flow:** resolve `:id` (parse UUID → `400`); look up the meeting's workspace from `ListEditableMeetings(telegramID)` (the meeting must be in the caller's editable set, else `403`/`404`); `EnsureTMAOrganizer`; call `UpdateMeeting` / `CancelMeeting`; map `ErrForbidden`→`403`, `ErrInvalidInput`→`400`, not-found→`404`. Edit returns the refreshed DTO; delete returns `204`. _(Using `ListEditableMeetings` both finds the workspace and provides the membership/recency guard for free; it already filters to `telegram_id`-owned scheduled future meetings.)_

**Conflicts handler:** body `{participants:[email…], date "YYYY-MM-DD", start "HH:MM", end "HH:MM", exclude_id?}`; parse start/end in Almaty; `MeetingConflicts(emails, start, end, excludeID)`; map to `tmaConflictDTO`.

**Request / response DTOs** (new, in `handlers/tma_write.go`; reuse `tmaMeetingDTO` from `tma_read.go`):

```go
type tmaCreateRequest struct {
    Dept, Type, Host, Date, Start, End, Recurrence, Desc string
    Participants []string // emails
}
type tmaUpdateRequest struct { // all optional; mapped to *string overrides
    Dept, Type, Host, Date, Start, End, Desc *string
}
type tmaConflictRequest struct {
    Participants []string `json:"participants"`
    Date, Start, End string
    ExcludeID string `json:"exclude_id"`
}
type tmaConflictDTO struct {
    Email string `json:"email"`
    Name  string `json:"name"`  // Conflict.PersonName (the wizard warning shows names)
    Title string `json:"title"` // Conflict.MeetingName
    Start string `json:"start"` // HH:MM Almaty
    End   string `json:"end"`
}
```

(`MeetingConflicts` returns `Conflict{Email, PersonName, MeetingName, Start, End (UTC)}` → map `Start/End.In(almaty)` to `HH:MM`.)

### Frontend

- **`shared/tma/api.ts`** — add write fetchers: `createMeeting(input)` (`POST /tma/meetings` → maps `{meeting}` via existing `toMeeting`), `updateMeeting(id, patch)` (`PATCH /tma/meetings/:id`), `deleteMeeting(id)` (`DELETE`), `fetchConflicts({participants,date,start,end,excludeId?})` (`POST /tma/conflicts` → `Conflict[]`). A `MeetingInput` type carries the create payload (snake_case mapping: `duration→end` is computed in the wizard already; send `start`/`end`).
- **`shared/tma/queries.ts`** — `useCreateMeeting`, `useUpdateMeeting`, `useDeleteMeeting` (each `useMutation` with `onSuccess: invalidateQueries(["tma","meetings"])` so every scope refetches), and `useConflicts` (a `useMutation` triggered when the wizard reaches the review step).
- **`shared/tma/meeting-utils.ts`** — `draftToMeeting` no longer hardcodes `ME.email`; the organizer comes from the server response, so the mock-only `draftToMeeting` is dropped from the create path (the server returns the canonical meeting). `detailToDraft` stays (pre-fills the edit wizard).
- **`create-wizard.tsx`** — replace mock `ME` host default with the real user (`useTmaAuth().user.name`); drop the client-side `conflictPeople` computation and instead call `useConflicts` on entering the review step (debounced/once), rendering the returned conflicts in the existing warning UI; when `rec !== "once"`, show a localized note on the review step and disable confirm (recurring deferred). Accept an optional `editId` prop; `onComplete` passes it through.
- **`tma-app.tsx`** — `completeCreate` calls `useCreateMeeting`/`useUpdateMeeting` (branch on `editId`) instead of `setQueryData`; on success keep the paw-burst + `SuccessView` (for create) / toast (for edit). `deleteMeeting` calls `useDeleteMeeting`. The `OverlayState` create variant carries `editId`; edit sets it (`openCreate(detailToDraft(detail), detail.id)`). Show the edit/delete actions in `MeetingDetail` only when `detail.organizer === user.email` (backend still enforces `403`). Mutation errors surface a localized toast; `meetings_not_configured` / `meetings_recurring_unsupported` get specific copy.

## Data flow & error handling

```
wizard review → useConflicts(participants,date,start,end) → POST /api/tma/conflicts
   → MeetingConflicts(emails, start, end, Nil) → render warning (non-blocking)
confirm → useCreateMeeting(input) → POST /api/tma/meetings
   → EnsureTMAOrganizer(email,tgID) → resolve Google workspace → CreateMeeting → toMeetingDTO
   → 201 → invalidate ["tma","meetings"] → SuccessView
edit  → useUpdateMeeting(id,patch) → PATCH → (ListEditableMeetings→ws) UpdateMeeting → 200 → invalidate
delete→ useDeleteMeeting(id)       → DELETE → CancelMeeting → 204 → invalidate
```

| Case                        | Backend                              | Frontend                        |
| --------------------------- | ------------------------------------ | ------------------------------- |
| OK create / edit            | `201` / `200` `{meeting}`            | invalidate + success UI         |
| OK delete                   | `204`                                | invalidate + toast              |
| No Google workspace         | `400 meetings_not_configured`        | specific toast                  |
| Recurring create attempted  | `400 meetings_recurring_unsupported` | confirm disabled + note (guard) |
| Bad input (time/date)       | `400` (`ErrInvalidInput`)            | error toast                     |
| Edit/delete others' meeting | `403` (`ErrForbidden`) / `404`       | "not your meeting" toast        |
| Expired TMA JWT             | `401`                                | existing interceptor re-login   |
| DB / Google error           | `500`                                | generic error toast             |

Logging: write handlers log a single `Info` lifecycle line on success (`tma_meeting_created`/`_updated`/`_cancelled` + `telegram_id`, `meeting_id`, `workspace_id` — no PII/email) and `Warn`/`Error` at the boundary on failure. No initData/JWT/email in fields.

## Testing

- **Backend unit (pure):** the create-request → `CreateMeetingInput` mapping (host fallback, participant emails, once-only guard) and the conflict-request time parse if extracted to a pure helper. `EnsureTMAOrganizer` is I/O (Store calls) → **build-verified** only (no DB harness, per convention); idempotency is guaranteed by `UpsertUserIdentity`'s existing semantics.
- **Backend build-verified:** the four handlers + DTO mapping + route wiring + `EnsureTMAOrganizer`.
- **Frontend:** `pnpm -C frontend typecheck` + `build`; mutations/hooks compile, wizard threads `editId`, review calls the real conflicts endpoint.
- Gate before merge: `make test && make lint && make build`.

## Out of scope (YAGNI / later)

- **Recurring create & whole-series edit/delete** — needs a wizard `recurrence_until` input + reconciling the `custom`/`recDays` UI with the backend's fixed recurrence model. (`CreateMeetingSeries`/`UpdateSeries`/`CancelSeries` already exist server-side; only the TMA surface is deferred.)
- **Reminder settings** (profile tab) + the `auto` scenarios tab — sub-project 4.
- Multi-workspace create targeting (first Google workspace is used).
- Participant add/remove on **edit** (`UpdateMeetingInput` does not carry participants — matches the bot's `/edit`).
- Optimistic UI for writes (invalidate-on-success is simple and correct at personal scale).
