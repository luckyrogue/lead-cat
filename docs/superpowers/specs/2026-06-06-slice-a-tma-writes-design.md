# Slice A — TMA Writes (Non-Recurring) Finish — Design

**Status:** approved (brainstorm), ready for implementation plan.
**Part of:** the meetings beta roadmap (`docs/superpowers/specs/2026-06-06-roadmap-to-beta-design.md`), slice A of 8.
**Ancestor spec:** `docs/superpowers/specs/2026-06-05-tma-write-paths-design.md` (earlier "sub-project 3" brainstorm) — supersedes its open parts; locks-in the parts that already shipped.
**ТЗ:** `docs/NEW-FEATURES.md` §4.4 (edit), §4.5 (delete), §4.7 (conflicts on create).

## Goal

Finish the TMA non-recurring write surface so an authed Telegram user can **edit** and **delete** their own meetings inside the Mini App, with a **real cross-participant conflict warning** on the create wizard's review step — replacing the optimistic-only React Query cache writes left in the current code. Reuses existing application commands (`UpdateMeeting` / `CancelMeeting` / `MeetingConflicts`) unchanged; bridges TMA `bot_users` → meeting-organizing `platform_users` via the already-shipped `EnsureTMAOrganizer`. Scope is **non-recurring single-meeting** writes; recurring create and whole-series edit/delete are slice B.

## Decisions (locked during brainstorming)

1. **Don't redo what shipped.** The previous sub-project-3 brainstorm proposed Tasks 1–9; Tasks 1 (`EnsureTMAOrganizer`) and 2 (`POST /api/tma/meetings`) already merged. Slice A re-uses them as-is and ships only the remaining three endpoints + the frontend write wiring.
2. **Edit path is URL-based, not overlay-state.** The route `meetings.create.$editId.tsx` already exists, and `create-page.tsx` reads `editId` from URL params. CreateWizard accepts `editId` as a prop, and `onComplete` branches PATCH vs POST. No `OverlayState.editId` plumbing.
3. **Frontend layout = FSD slice, not `shared/tma/`.** Recent pivot moved write code under `frontend/src/features/meetings/{api.ts,queries.ts}` and `frontend/src/features/meeting-create/`. New fetchers/hooks land there; `shared/tma/i18n.ts` stays the i18n catalog.
4. **OpenAPI is part of slice A.** `backend/openapi/openapi.json` is the hand-maintained source of truth; the file is embedded into the binary and the frontend generated client (`frontend/src/shared/api/generated/schema.ts`) is derived from it. Slice A adds the 3 new paths + schemas in the same slice so the generated client picks them up — no orphan deferral.
5. **Editable-set IS the ownership guard.** A new helper `editableWorkspace(c, telegramID, meetingID)` resolves the meeting's `WorkspaceID` by scanning `ListEditableMeetings(telegramID)` (filters to `telegram_id=$1 AND status='scheduled' AND starts_at>now()`). If `:id` isn't in that set → `404 not_found`. `UpdateMeeting`/`CancelMeeting` still re-enforce `ownerOrOrganizer` server-side and return `ErrForbidden` → `403`.
6. **Conflict warning is non-blocking** (matches §4.7.3 + bot `/edit` behavior). The wizard's review step calls `POST /api/tma/conflicts` once on entry; conflicts render in the existing warning box but the user may still confirm. The currently client-side `conflictPeople` memo is removed.
7. **Recurring guard, not recurring support.** Backend returns `400 meetings_recurring_unsupported` (in the existing create handler) for `rec != "once"`. Slice A adds a friendly UI guard on the review step: when `rec !== "once"` show a localized `recurringSoon` note and disable confirm. Slice B handles real recurring create.
8. **Writes invalidate; never optimistic cache surgery.** The current `queryClient.setQueryData` create/delete shortcuts are removed. Each mutation hook's `onSuccess` invalidates `["tma","meetings"]` so every scope refetches. The create success animation (`SuccessView`, paw burst, toast) is preserved and fed by the server-returned meeting.

## Codebase facts (verified at HEAD `94c0baa`)

- **Module path:** `github.com/luckyrogue/lead-cat`.
- **TMA group** (`backend/internal/delivery/http/app.go`, lines 151–156): `GET /me`, `GET /meetings`, `GET /schedule`, `GET /employees`, `POST /free-slots`, `POST /meetings` are wired. PATCH/DELETE/conflicts are NOT wired yet.
- **Existing handler file** (`backend/internal/delivery/http/handlers/tma_write.go`, 97 lines): contains `tmaCreateRequest`, `toCreateMeetingInput`, `botUser(c) (postgres.BotUser, bool)`, `TMACreateMeeting`. **Slice A appends here**, not in a new file.
- **Reusable application commands** (signatures verified):
  - `(*Services).CreateMeeting(ctx, workspaceID, organizerID uuid.UUID, in CreateMeetingInput) (postgres.Meeting, error)` — already wired by `TMACreateMeeting`.
  - `(*Services).UpdateMeeting(ctx, workspaceID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (postgres.Meeting, error)`. `UpdateMeetingInput{Dept,Type,Host,Date,Start,End,Recurrence,Description *string}` — all-pointer overrides; date/start/end are a unit. Returns `ErrForbidden` from `ownerOrOrganizer`.
  - `(*Services).CancelMeeting(ctx, workspaceID, userID, id uuid.UUID) error` — idempotent; emits `meeting:cancelled` DM; returns `ErrForbidden` when not owner/organizer.
  - `(*Services).ListEditableMeetings(ctx, telegramID int64) ([]postgres.MeetingWithTZ, error)` — `MeetingWithTZ` embeds `postgres.Meeting` (so `.ID`, `.WorkspaceID` are promoted) + `TZ string`.
  - `(*Services).MeetingConflicts(ctx, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)`. `Conflict{Email, PersonName, MeetingName string; Start, End time.Time /*UTC*/}`. Pass `uuid.Nil` to exclude nothing.
  - `(*Services).EnsureTMAOrganizer(ctx, email string, telegramID int64) (uuid.UUID, error)` — may return `ErrTelegramLinkedToOtherAccount` (→ `409`).
- **Read-handler conventions** (`backend/internal/delivery/http/handlers/tma_read.go`): receiver `*API`; `a.App` is `*application.Services`; `botUserEmail(c)` (in `tma_read.go`) and `botUser(c)` (in `tma_write.go`) share the `c.Locals("bot_user")` cast; `almatyLoc()` helper in `meeting_availability.go`; `a.toMeetingDTO(ctx, m)` already maps `postgres.Meeting` → `tmaMeetingDTO`. Reuse all.
- **REST handler conventions** (`handlers/meetings.go`): error mapping pattern — `application.ErrInvalidInput`/`ErrGoogleNotConfigured` → `400`, `ErrForbidden` → `403` (with `copy.APIError("forbidden")`), not-found → `404`. `DELETE` returns `204`.
- **OpenAPI** (`backend/openapi/openapi.json`): authoritative, hand-maintained, embedded into the binary; the generated client (`frontend/src/shared/api/generated/schema.ts`) mirrors it. Currently lists the 6 TMA paths above; PATCH/DELETE/conflicts are absent.
- **Frontend code surface** (`frontend/src/`):
  - `features/meetings/` — read fetchers + hooks live here (FSD pivot from earlier `shared/tma/`).
  - `features/meeting-create/components/create-wizard.tsx` — `STEPS=[what,when,who,review]`; draft default `host: ME.name` (line ~271); `onComplete({...draft, end: endTime})` on the final step; client-side `conflictPeople` memo (~lines 31, 97–99, 867); `finalMeeting.organizer = ME.email` (~line 346); recurrence chips incl. `custom`/`recDays`. **No `editId` prop yet**; no `useConflicts` call; no recurring guard.
  - `features/meeting-create/pages/create-page.tsx` (~79 lines) — reads `editId` from URL params (line ~20); currently does optimistic `queryClient.setQueryData` instead of real mutations (~lines 48–62); no `useCreateMeeting`/`useUpdateMeeting` calls.
  - `shared/tma/auth-context.tsx` — `useTmaAuth(): {status, user, retry}`, `user: {telegramId, name, email, role}`.
  - `shared/tma/i18n.ts` — `translate(lang, key)`; missing keys `errNotConfigured`, `errNotYours`, `recurringSoon`, `errGeneric`, `updated`, `deleted` in all three language packs.

## Architecture

Three thin TMA handlers append to `tma_write.go`, each (a) reads the `bot_user` from locals, (b) for edit/delete: resolves the meeting's workspace via `editableWorkspace` (which doubles as the ownership/recency guard), (c) bridges identity via `EnsureTMAOrganizer`, (d) delegates to the existing application command, (e) returns the same `tmaMeetingDTO` the read path uses. The frontend swaps optimistic cache surgery for real mutations + invalidation, threads `editId` from URL through CreateWizard, calls the real conflicts endpoint on the wizard's review step, and surfaces localized errors.

### Backend — endpoints to add (under the `/api/tma` group, TMA-auth)

| Method & path                  | Reuses                                                       | Returns                             |
| ------------------------------ | ------------------------------------------------------------ | ----------------------------------- |
| `PATCH /api/tma/meetings/:id`  | `editableWorkspace` + `EnsureTMAOrganizer` + `UpdateMeeting` | `200 {meeting: tmaMeetingDTO}`      |
| `DELETE /api/tma/meetings/:id` | `editableWorkspace` + `EnsureTMAOrganizer` + `CancelMeeting` | `204`                               |
| `POST /api/tma/conflicts`      | `MeetingConflicts`                                           | `200 {conflicts: tmaConflictDTO[]}` |

**New helper (`tma_write.go`):**

```go
// editableWorkspace returns the workspace of a meeting the TMA user may edit,
// or false if the meeting is not in their editable set (not theirs / not scheduled / past).
func (a *API) editableWorkspace(c *fiber.Ctx, telegramID int64, meetingID uuid.UUID) (uuid.UUID, bool, error)
```

**Request/response DTOs (new, in `tma_write.go`; reuse `tmaMeetingDTO` from `tma_read.go`):**

```go
type tmaUpdateRequest struct {
    Dept  *string `json:"dept"`
    Type  *string `json:"type"`
    Host  *string `json:"host"`
    Date  *string `json:"date"`
    Start *string `json:"start"`
    End   *string `json:"end"`
    Desc  *string `json:"desc"`
}
type tmaConflictRequest struct {
    Participants []string `json:"participants"`
    Date, Start, End string
    ExcludeID string `json:"exclude_id"`
}
type tmaConflictDTO struct {
    Email string `json:"email"`
    Name  string `json:"name"`  // Conflict.PersonName
    Title string `json:"title"` // Conflict.MeetingName
    Start string `json:"start"` // HH:MM Almaty
    End   string `json:"end"`
}
```

**OpenAPI:** add the three paths + schemas (`tmaUpdateRequest`, `tmaConflictRequest`, `tmaConflictDTO`, reuse the existing meeting/conflict schemas where present) to `backend/openapi/openapi.json` and rebuild so the binary embed + frontend `schema.ts` regenerate.

### Frontend — what changes

- **`features/meetings/api.ts`** — append `createMeeting(input)`, `updateMeeting(id, patch)`, `deleteMeeting(id)`, `fetchConflicts(params)` typed fetchers. Reuse the existing private `toMeeting(d: MeetingDTO): Meeting` mapper. Types: `MeetingInput`, `MeetingPatch`, `Conflict`, `ConflictsParams`.
- **`features/meetings/queries.ts`** — `useCreateMeeting`, `useUpdateMeeting`, `useDeleteMeeting` (each `useMutation` with `onSuccess: invalidateQueries({queryKey: ["tma","meetings"]})`); `useConflicts` (`useMutation`).
- **`features/meeting-create/components/create-wizard.tsx`** —
  - Drop `import { ME }`; use `useTmaAuth().user`. Draft default `host: initial?.host ?? user?.name ?? ""`. `finalMeeting.organizer = user?.email ?? ""`.
  - Add `editId?: string` prop; pass through `onComplete`.
  - Remove the `conflictPeople` `useMemo`; add `const conflictsMut = useConflicts()`; fire it from a `useEffect` keyed on the review step + relevant draft fields (`step`, `draft.date`, `draft.start`, `endTime`, `draft.participants`). Render the warning from `conflictsMut.data ?? []` (unique by `c.name`).
  - Add `const recurringBlocked = draft.rec !== "once"`. On the review step, render the `recurringSoon` note (warning-box styling) when `recurringBlocked`; combine `disabled={!canNext || recurringBlocked}` on the confirm button.
- **`features/meeting-create/pages/create-page.tsx`** —
  - Read `editId` from URL params (already done).
  - Replace `queryClient.setQueryData` with real mutations: `const createMut = useCreateMeeting(); const updateMut = useUpdateMeeting();`.
  - `onComplete(draft)` branches: `editId ? updateMut.mutateAsync({id: editId, patch: …}) : createMut.mutateAsync({…})`. On create-success keep `SuccessView` + paw burst, navigate to `/meetings`; on edit-success toast `updated` and navigate back.
  - Wrap in `try/catch`; map errors via `writeErrorMessage(err)` helper (module-scope or in `features/meetings/`) — `meetings_not_configured` → `errNotConfigured`, `meetings_recurring_unsupported` → `recurringSoon`, 403 / `"forbidden"` → `errNotYours`, else `errGeneric`. Use `isAxiosError` from `axios`; never log the error object (it may carry `init_data`).
- **Meeting detail** (`features/meetings/`) — pass `canModify={detail.organizer === user?.email}` to the detail component; hide/disable Edit + Delete actions when false (backend still 403/404s). Delete uses `useDeleteMeeting`.
- **i18n** (`shared/tma/i18n.ts`) — add keys to all three language packs (ru/kk/en):
  - `errNotConfigured` — "Создание встреч не настроено" / "Meeting creation isn't configured" / kk
  - `errNotYours` — "Это не ваша встреча" / "Not your meeting" / kk
  - `recurringSoon` — "Повторяющиеся встречи скоро будут доступны" / "Recurring meetings coming soon" / kk
  - `errGeneric` — "Что-то пошло не так" / "Something went wrong" / kk
  - `updated` — "Встреча обновлена" / "Meeting updated" / kk
  - `deleted` — already exists? if not, add — "Встреча удалена" / "Meeting deleted" / kk

## Data flow & error handling

```
wizard review → useConflicts(participants,date,start,end) → POST /api/tma/conflicts
   → MeetingConflicts(emails, start, end, Nil) → render names in warning (non-blocking)
confirm (create) → useCreateMeeting → POST /api/tma/meetings
   → EnsureTMAOrganizer → resolve Google workspace → CreateMeeting → toMeetingDTO
   → 201 → invalidate ["tma","meetings"] → SuccessView
confirm (edit) → useUpdateMeeting({id,patch}) → PATCH /api/tma/meetings/:id
   → editableWorkspace → EnsureTMAOrganizer → UpdateMeeting → 200 → invalidate → toast "updated"
delete → useDeleteMeeting(id) → DELETE /api/tma/meetings/:id
   → editableWorkspace → EnsureTMAOrganizer → CancelMeeting → 204 → invalidate → toast "deleted"
```

| Case                        | Backend                                  | Frontend                                       |
| --------------------------- | ---------------------------------------- | ---------------------------------------------- |
| OK create/edit              | `201`/`200 {meeting}`                    | invalidate + success UI                        |
| OK delete                   | `204`                                    | invalidate + toast                             |
| No Google workspace         | `400 meetings_not_configured`            | `errNotConfigured` toast                       |
| Recurring create attempted  | `400 meetings_recurring_unsupported`     | `recurringSoon` toast (UI also pre-blocks)     |
| Bad input                   | `400` (`ErrInvalidInput`, e.g. bad time) | `errGeneric` toast                             |
| Edit/delete others' meeting | `403` (`ErrForbidden`) / `404` not_found | `errNotYours` toast (UI also hides the action) |
| Expired TMA JWT             | `401`                                    | existing interceptor re-login                  |
| DB / Google error           | `500`                                    | `errGeneric` toast                             |

Logging: one `Info` per successful write — `tma_meeting_updated` / `tma_meeting_cancelled` + `zap.Int64("telegram_id",…)`, `zap.String("meeting_id",…)`, `zap.String("workspace_id",…)`. No email/initData/JWT/PII fields.

## Testing

- **Backend unit (pure):** TDD on `toConflictDTO(c application.Conflict, loc *time.Location) tmaConflictDTO` (Almaty `HH:MM` rendering, including a meeting that straddles local midnight). The `toCreateMeetingInput` test already exists from the shipped create work.
- **Backend build-verified:** the three new handlers + `editableWorkspace` + DTO mapping + route wiring + OpenAPI embed.
- **Frontend:** `pnpm -C frontend typecheck` + `pnpm -C frontend build`. The generated `schema.ts` must compile after the OpenAPI update; the fetchers/hooks/wizard wiring must typecheck.
- **Gate before merge:** `make test && make lint && make build` from repo root.

## Out of scope (slice B or later)

- **Recurring create** (the wizard's `rec != once` + `recDays`) — slice B; the `until` input + `recDays` reconciliation are dedicated to that slice.
- **Whole-series edit/delete** with `scope=this|whole` — slice B; existing `UpdateSeries`/`CancelSeries` will be exposed then.
- **Participant edits on update** — `UpdateMeetingInput` does not carry participants (matches bot `/edit`); separate increment if/when needed.
- **Multi-workspace TMA targeting** — TMA create still uses the first Google-configured workspace; surface in slice D when admin gains multi-workspace controls.
- **Optimistic UI for writes** — invalidate-on-success is the chosen model.
- **TMA admin** integrations / panel — slices D and E.

## Process

- Branch `feat/meetings-tma-write-paths-a`, full superpowers chain (writing-plans → subagent-driven-development → finishing-a-development-branch).
- Per the concurrent-git-on-shared-branch memory: every commit stages explicit paths only; never `git add -A`; never `frontend/vite.config.ts`. The working tree's in-progress code stays untouched.
- After merging, refresh `docs/MEETINGS.md` (write paths → Done) and `docs/API.md` (move the three routes from "Planned" to "Present").
