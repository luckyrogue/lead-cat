# Slice A — TMA Writes (Non-Recurring) Finish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the remaining TMA non-recurring write surface — `PATCH /api/tma/meetings/:id`, `DELETE /api/tma/meetings/:id`, `POST /api/tma/conflicts` — and replace the frontend's optimistic-only React Query cache writes with real mutations + invalidation, threading `editId` from URL through the create wizard and surfacing real cross-participant conflicts on the review step.

**Architecture:** Three thin TMA handlers append to the existing `tma_write.go`; an `editableWorkspace` helper resolves the meeting's workspace via `ListEditableMeetings` (also doubles as the ownership/recency guard). The frontend gets typed fetchers + mutation hooks (invalidate `["tma"]` on success), the create wizard swaps its client-side conflict memo for a real `POST /api/tma/conflicts` call and gains a recurring guard, and `create-page.tsx` calls real mutations instead of `setQueryData`. OpenAPI gets the three new paths so the generated client picks them up.

**Tech Stack:** Go (Fiber, pgx/Postgres, zap), React + Vite + TanStack Router/Query, axios via `apiFetch`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-06-06-slice-a-tma-writes-design.md`.

---

## Grounded facts (verified at HEAD `94c0baa` — re-verify before asserting line numbers)

- **Module:** `github.com/Jaryq-Lab/notify-bot`. **Product:** Lead Cat.
- **Branch for this slice:** `feat/meetings-tma-write-paths-a` (new). The working tree has heavy in-progress code from the pivot — every task stages explicit paths only; never `git add -A`; never `frontend/vite.config.ts`.
- **`backend/internal/delivery/http/handlers/tma_write.go` (97 lines, shipped)** contains: `tmaCreateRequest`, `toCreateMeetingInput`, `botUser(c) (postgres.BotUser, bool)`, `TMACreateMeeting`. Imports already include `errors`, `strings`, `fiber/v2`, `zap`, `application`, `domain/meeting`, `infrastructure/persistence/postgres`. **Slice A appends to this file** — does NOT create a new file. New imports needed: `"time"`, `"github.com/google/uuid"`.
- **`backend/internal/delivery/http/handlers/tma_read.go`** has `(*API).toMeetingDTO(ctx context.Context, m postgres.Meeting) tmaMeetingDTO` (line 69) and `botUserEmail(c) (string, bool)` (line 103). Reuse `toMeetingDTO`; do NOT redefine.
- **`backend/internal/delivery/http/handlers/meeting_availability.go`** has `almatyLoc() *time.Location` (line 10). Reuse.
- **Application commands (signatures verified):**
  - `(*Services).UpdateMeeting(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (postgres.Meeting, error)` — `UpdateMeetingInput{Dept, Type, Host, Date, Start, End, Recurrence, Description *string}`. Authorises via `ownerOrOrganizer` → `application.ErrForbidden`. Date/Start/End must be supplied together to change the time.
  - `(*Services).CancelMeeting(ctx context.Context, workspaceID, userID, id uuid.UUID) error` — idempotent; emits `meeting:cancelled` DM; returns `application.ErrForbidden` when not owner/organizer.
  - `(*Services).ListEditableMeetings(ctx context.Context, telegramID int64) ([]postgres.MeetingWithTZ, error)` — `MeetingWithTZ` embeds `postgres.Meeting` (so `.ID`, `.WorkspaceID` are promoted) + `TZ string`. Filter is `telegram_id=$1 AND status='scheduled' AND starts_at>now()`.
  - `(*Services).MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)` — `Conflict{Email, PersonName, MeetingName string; Start, End time.Time /*UTC*/}`. Pass `uuid.Nil` to exclude nothing.
  - `(*Services).EnsureTMAOrganizer(ctx context.Context, email string, telegramID int64) (uuid.UUID, error)` — may return `application.ErrTelegramLinkedToOtherAccount` (→ `409`).
- **Routes registered today** (`backend/internal/delivery/http/app.go:151-156`):
  ```
  tma.Get("/me", api.TMAMe)
  tma.Get("/meetings", api.TMAMyMeetings)
  tma.Get("/schedule", api.TMASchedule)
  tma.Get("/employees", api.TMAEmployees)
  tma.Post("/free-slots", api.TMAFreeSlots)
  tma.Post("/meetings", api.TMACreateMeeting)
  ```
  Slice A appends three more (PATCH/DELETE meetings/:id, POST /conflicts).
- **OpenAPI:** `backend/openapi/openapi.json` is hand-maintained, embedded into the binary (`backend/internal/delivery/http/handlers/openapi.go`), and mirrored to `frontend/src/shared/api/generated/schema.ts`. Slice A edits the JSON, rebuilds, and regenerates the frontend schema.
- **Frontend:**
  - `frontend/src/features/meetings/api.ts` — `apiFetch` (not raw axios); private `MeetingDTO`; private `toMeeting(d): Meeting`; existing read fetchers `fetchMyMeetings`/`fetchColleagueSchedule`/`searchEmployees`/`fetchFreeSlots`. Slice A APPENDS write fetchers here.
  - `frontend/src/features/meetings/queries.ts` — uses `queryOptions` + `tmaKeys` from `@/shared/api/query-keys`. Slice A adds mutation hooks here.
  - `frontend/src/shared/api/query-keys.ts` — `tmaKeys.all = ["tma"]`; `tmaKeys.meetings(scope)` etc. Invalidate `tmaKeys.all` (broad) so every scope refetches.
  - `frontend/src/features/meeting-create/pages/create-page.tsx` — already reads `editId` from URL params and uses real `useTmaAuth().user.email` for organizer; currently does optimistic `queryClient.setQueryData(tmaKeys.meetings("all"), …)` for create AND edit (mock-only path). Slice A replaces those with real mutations.
  - `frontend/src/features/meeting-create/components/create-wizard.tsx` — split into substeps; delegates state to `useCreateWizard({initial, meetings, onComplete})` hook in `frontend/src/features/meeting-create/lib/use-create-wizard.ts`. The current client-side conflict memo lives in that hook (it computes `conflictPeople: string[]`). Slice A replaces that memo with a `useConflicts()` call.
  - `frontend/src/shared/tma/i18n.ts` — `translate(lang: Lang, key: I18nKey)` over a typed catalog with ru/kk/en. Slice A adds 6 keys.
  - `frontend/src/features/auth/auth-context.tsx` — `useTmaAuth(): {status, user, retry}`, `user: {telegramId, name, email, role}`. (Re-verify import path; some files may import from `@/shared/tma/auth-context`.)
- **REST error-mapping convention** (`backend/internal/delivery/http/handlers/meetings.go`): `application.ErrInvalidInput` / `ErrGoogleNotConfigured` → `400` with `err.Error()` as message; `application.ErrForbidden` → `403` (uses `copy.APIError("forbidden")` in REST; TMA handlers use plain `"forbidden"`); not-found → `404`; `DELETE` returns `204`.

## Conventions

- Backend: build/test/lint from repo root `make test && make lint && make build`; Go as `env -u GOROOT go ...` from `backend/`; `make lint` runs golangci-lint incl. gofmt. Pure logic unit-tested; handlers/wiring build-verified (no HTTP harness, by convention).
- Frontend: `pnpm -C frontend typecheck` + `pnpm -C frontend build`; `pnpm -C frontend format` per task. No test runner today; the OpenAPI regen step needs the frontend codegen command (see Task 5).
- Logging: one `Info` lifecycle line per successful write (`tma_meeting_updated`/`tma_meeting_cancelled` + `zap.Int64("telegram_id",…)`, `zap.String("meeting_id",…)`, `zap.String("workspace_id",…)`). No email/initData/JWT/PII.
- Commit messages end with `Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>`.
- Per-task git: explicit `git add <path1> <path2>`; never `-A`; never `frontend/vite.config.ts`.

## File structure (created/modified)

- **Modify:** `backend/internal/delivery/http/handlers/tma_write.go` (append helpers + 3 handlers + DTOs).
- **Create:** `backend/internal/delivery/http/handlers/tma_write_test.go` (or append if it already exists — there's a `TestToCreateMeetingInput` from the shipped create work; verify before writing).
- **Modify:** `backend/internal/delivery/http/app.go` (register 3 routes).
- **Modify:** `backend/openapi/openapi.json` (add 3 paths + DTO schemas).
- **Regen:** `frontend/src/shared/api/generated/schema.ts` (from updated OpenAPI).
- **Modify:** `frontend/src/features/meetings/api.ts` (append write fetchers + types).
- **Modify:** `frontend/src/features/meetings/queries.ts` (add mutation hooks).
- **Modify:** `frontend/src/features/meeting-create/lib/use-create-wizard.ts` (real conflicts + recurring guard).
- **Modify:** `frontend/src/features/meeting-create/pages/create-page.tsx` (real mutations + error mapper).
- **Modify:** the meeting-detail action site (gating + delete mutation). Path TBV by Task 11 (likely `frontend/src/features/meetings/components/...`).
- **Modify:** `frontend/src/shared/tma/i18n.ts` (6 new keys × 3 langs).
- **Modify:** `docs/MEETINGS.md`, `docs/API.md` (post-merge doc refresh).

---

## Task 0: Create the slice branch

**Files:** none (git only).

- [ ] **Step 1: Create and switch to the branch**

```bash
git checkout main
git checkout -b feat/meetings-tma-write-paths-a
```

> If `git checkout main` fails because of the dirty working tree, STOP and report — do NOT stash code WIP without confirmation. The user may want to commit their WIP first or use a fresh worktree.

---

## Task 1: Backend pure `toConflictDTO` mapper (TDD)

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write_test.go` (file already exists per the shipped create work; this task APPENDS a test).
- Modify: `backend/internal/delivery/http/handlers/tma_write.go` (append the DTO + mapper).

- [ ] **Step 1: Verify the existing test file shape**

Run: `cd backend && ls internal/delivery/http/handlers/tma_write_test.go && head -5 internal/delivery/http/handlers/tma_write_test.go`
Expected: file exists with `package handlers`. If it doesn't exist, create it with `package handlers` and an empty body before proceeding.

- [ ] **Step 2: Append the failing test**

Append to `backend/internal/delivery/http/handlers/tma_write_test.go`:

```go
func TestToConflictDTO(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 09:00–10:00 UTC == 14:00–15:00 Almaty
	c := application.Conflict{
		Email:       "a@x.io",
		PersonName:  "Alice",
		MeetingName: "Weekly",
		Start:       time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	d := toConflictDTO(c, loc)
	if d.Email != "a@x.io" || d.Name != "Alice" || d.Title != "Weekly" || d.Start != "14:00" || d.End != "15:00" {
		t.Fatalf("toConflictDTO got %+v", d)
	}
}
```

Ensure the test file imports `"time"` and `"github.com/Jaryq-Lab/notify-bot/internal/application"` (in the test file's import block).

- [ ] **Step 3: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToConflictDTO' -v`
Expected: FAIL — `undefined: toConflictDTO` and `undefined: tmaConflictDTO` (if no test setup).

- [ ] **Step 4: Implement the DTO + pure mapper**

Append to `backend/internal/delivery/http/handlers/tma_write.go`. Also extend the file's import block to include `"time"`:

```go
type tmaConflictDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`  // application.Conflict.PersonName
	Title string `json:"title"` // application.Conflict.MeetingName
	Start string `json:"start"` // HH:MM Almaty
	End   string `json:"end"`
}

// toConflictDTO renders a conflict's UTC times into Almaty HH:MM. Pure.
func toConflictDTO(c application.Conflict, loc *time.Location) tmaConflictDTO {
	return tmaConflictDTO{
		Email: c.Email,
		Name:  c.PersonName,
		Title: c.MeetingName,
		Start: c.Start.In(loc).Format("15:04"),
		End:   c.End.In(loc).Format("15:04"),
	}
}
```

- [ ] **Step 5: Run tests + build + gofmt**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToConflictDTO|ToCreateMeetingInput' -v && env -u GOROOT go build ./... && env -u GOROOT gofmt -l internal/delivery/http/handlers/`
Expected: PASS; build clean; gofmt prints nothing.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/handlers/tma_write_test.go
git commit -m "feat(tma): toConflictDTO pure mapper (Almaty HH:MM)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 2: `POST /api/tma/conflicts` handler + route

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write.go` (append request DTO + handler).
- Modify: `backend/internal/delivery/http/app.go` (register route).

Build-verified.

- [ ] **Step 1: Append the request DTO and handler to `tma_write.go`**

```go
type tmaConflictRequest struct {
	Participants []string `json:"participants"`
	Date         string   `json:"date"`  // YYYY-MM-DD
	Start        string   `json:"start"` // HH:MM
	End          string   `json:"end"`   // HH:MM
	ExcludeID    string   `json:"exclude_id"`
}

// TMAConflicts reports cross-participant conflicts for a pending meeting (§4.7).
func (a *API) TMAConflicts(c *fiber.Ctx) error {
	if _, ok := botUser(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaConflictRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/participants")
	}
	exclude := uuid.Nil
	if s := strings.TrimSpace(req.ExcludeID); s != "" {
		if id, perr := uuid.Parse(s); perr == nil {
			exclude = id
		}
	}
	conflicts, err := a.App.MeetingConflicts(c.Context(), req.Participants, start, end, exclude)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	out := make([]tmaConflictDTO, 0, len(conflicts))
	for _, cf := range conflicts {
		out = append(out, toConflictDTO(cf, loc))
	}
	return c.JSON(fiber.Map{"conflicts": out})
}
```

Add `"github.com/google/uuid"` to `tma_write.go`'s import block (it's not yet imported there).

- [ ] **Step 2: Register the route in `app.go`**

Insert immediately after `tma.Post("/meetings", api.TMACreateMeeting)` (line 156):

```go
	tma.Post("/conflicts", api.TMAConflicts)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): POST /api/tma/conflicts (create-wizard real warning)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 3: `editableWorkspace` helper + `PATCH /api/tma/meetings/:id` handler + route

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write.go`.
- Modify: `backend/internal/delivery/http/app.go`.

Build-verified. `ListEditableMeetings` already filters to the caller's scheduled future meetings; a meeting not in that set → `404`. `UpdateMeeting` still re-enforces `ownerOrOrganizer` server-side (`403`).

- [ ] **Step 1: Append the helper + request DTO + handler to `tma_write.go`**

```go
// editableWorkspace returns the workspace of a meeting the TMA user may edit,
// or false if the meeting is not in their editable set (not theirs / not
// scheduled / past). Used by edit + delete.
func (a *API) editableWorkspace(c *fiber.Ctx, telegramID int64, meetingID uuid.UUID) (uuid.UUID, bool, error) {
	ms, err := a.App.ListEditableMeetings(c.Context(), telegramID)
	if err != nil {
		return uuid.Nil, false, err
	}
	for _, m := range ms {
		if m.ID == meetingID {
			return m.WorkspaceID, true, nil
		}
	}
	return uuid.Nil, false, nil
}

type tmaUpdateRequest struct {
	Dept  *string `json:"dept"`
	Type  *string `json:"type"`
	Host  *string `json:"host"`
	Date  *string `json:"date"`
	Start *string `json:"start"`
	End   *string `json:"end"`
	Desc  *string `json:"desc"`
}

// TMAUpdateMeeting edits a single meeting the authed TMA user organizes (§4.4).
func (a *API) TMAUpdateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	var req tmaUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	workspaceID, found, err := a.editableWorkspace(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.UpdateMeeting(c.Context(), workspaceID, organizerID, meetingID, application.UpdateMeetingInput{
		Dept: req.Dept, Type: req.Type, Host: req.Host,
		Date: req.Date, Start: req.Start, End: req.End, Description: req.Desc,
	})
	if err != nil {
		switch {
		case errors.Is(err, application.ErrForbidden):
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		case errors.Is(err, application.ErrInvalidInput):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
	}
	a.App.Log.Info("tma_meeting_updated",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("workspace_id", workspaceID.String()))
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}
```

- [ ] **Step 2: Register the route in `app.go`**

Insert immediately after `tma.Post("/conflicts", api.TMAConflicts)` from Task 2:

```go
	tma.Patch("/meetings/:id", api.TMAUpdateMeeting)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): PATCH /api/tma/meetings/:id (single-meeting edit)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 4: `DELETE /api/tma/meetings/:id` handler + route

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write.go`.
- Modify: `backend/internal/delivery/http/app.go`.

Build-verified. Reuses the `editableWorkspace` helper from Task 3.

- [ ] **Step 1: Append the handler to `tma_write.go`**

```go
// TMADeleteMeeting cancels a single meeting the authed TMA user organizes (§4.5).
func (a *API) TMADeleteMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	workspaceID, found, err := a.editableWorkspace(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if err := a.App.CancelMeeting(c.Context(), workspaceID, organizerID, meetingID); err != nil {
		if errors.Is(err, application.ErrForbidden) {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.App.Log.Info("tma_meeting_cancelled",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("workspace_id", workspaceID.String()))
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 2: Register the route in `app.go`**

Insert immediately after `tma.Patch("/meetings/:id", api.TMAUpdateMeeting)` from Task 3:

```go
	tma.Delete("/meetings/:id", api.TMADeleteMeeting)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): DELETE /api/tma/meetings/:id (single-meeting cancel)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 5: OpenAPI — add 3 paths + regenerate frontend schema

**Files:**

- Modify: `backend/openapi/openapi.json` (add 3 paths + DTO schemas).
- Modify: `frontend/src/shared/api/generated/schema.ts` (regenerated; do not hand-edit).

- [ ] **Step 1: Verify the OpenAPI workflow**

Run: `cd frontend && grep -E "openapi|codegen|generate" package.json | head -10`
This reveals the regen command (e.g. `pnpm run codegen` or `pnpm run openapi:generate`). If no script exists, inspect `package.json`'s `scripts` block fully and `grep -rn "openapi-typescript" frontend/` to find how the schema is generated. Once found, note the exact command for Step 4.

> Troubleshooting: if there is no scripted regen, the typical command is `pnpm dlx openapi-typescript backend/openapi/openapi.json -o frontend/src/shared/api/generated/schema.ts`. Use that if no script is wired.

- [ ] **Step 2: Add the three paths to `backend/openapi/openapi.json`**

Append new entries under `paths`. For each, follow the shape of the existing `POST /api/tma/meetings` entry (request body schema, `200`/`201`/`204` responses, `401` reference, security `bearerAuth`). The three additions:

- `PATCH /api/tma/meetings/{id}` — path param `id` (UUID string); requestBody `application/json` with schema `TmaMeetingUpdateRequest`; `200` returns `{ meeting: TmaMeeting }`; `400` `meetings_recurring_unsupported|invalid body|invalid time` (component `Error`); `403` `forbidden`; `404` `not_found`; `409` `telegram_linked_to_other_account`.
- `DELETE /api/tma/meetings/{id}` — path param `id`; no body; `204` no content; `403` / `404` / `409` as above.
- `POST /api/tma/conflicts` — requestBody `TmaConflictsRequest`; `200` returns `{ conflicts: TmaConflict[] }`; `400` `invalid range/participants`.

Add these component schemas (under `components.schemas`):

```json
"TmaMeetingUpdateRequest": {
  "type": "object",
  "properties": {
    "dept":  { "type": "string", "nullable": true },
    "type":  { "type": "string", "nullable": true },
    "host":  { "type": "string", "nullable": true },
    "date":  { "type": "string", "nullable": true, "description": "YYYY-MM-DD" },
    "start": { "type": "string", "nullable": true, "description": "HH:MM" },
    "end":   { "type": "string", "nullable": true, "description": "HH:MM" },
    "desc":  { "type": "string", "nullable": true }
  }
},
"TmaConflictsRequest": {
  "type": "object",
  "required": ["participants", "date", "start", "end"],
  "properties": {
    "participants": { "type": "array", "items": { "type": "string", "format": "email" } },
    "date":  { "type": "string", "description": "YYYY-MM-DD" },
    "start": { "type": "string", "description": "HH:MM Almaty" },
    "end":   { "type": "string", "description": "HH:MM Almaty" },
    "exclude_id": { "type": "string", "format": "uuid" }
  }
},
"TmaConflict": {
  "type": "object",
  "required": ["email", "name", "title", "start", "end"],
  "properties": {
    "email": { "type": "string", "format": "email" },
    "name":  { "type": "string" },
    "title": { "type": "string" },
    "start": { "type": "string", "description": "HH:MM Almaty" },
    "end":   { "type": "string", "description": "HH:MM Almaty" }
  }
}
```

Reuse the existing `TmaMeeting` schema (added by the create endpoint) for the PATCH response.

- [ ] **Step 3: Rebuild the backend so the binary embed picks up the new JSON**

Run: `cd backend && env -u GOROOT go build ./...`
Expected: clean build.

- [ ] **Step 4: Regenerate the frontend client schema**

Run the codegen command discovered in Step 1 (or the dlx fallback). For example:

```bash
pnpm -C frontend run codegen   # or: pnpm dlx openapi-typescript backend/openapi/openapi.json -o frontend/src/shared/api/generated/schema.ts
```

- [ ] **Step 5: Verify the frontend still typechecks**

Run: `pnpm -C frontend typecheck`
Expected: PASS. If it fails, the regen probably renamed a path/component — adjust the generator config or the openapi.json to keep names stable.

- [ ] **Step 6: Commit**

```bash
git add backend/openapi/openapi.json frontend/src/shared/api/generated/schema.ts
git commit -m "feat(api): document TMA PATCH/DELETE/conflicts in OpenAPI

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 6: Frontend i18n keys (ru/kk/en)

**Files:**

- Modify: `frontend/src/shared/tma/i18n.ts`.

- [ ] **Step 1: Read the file**

Run: `head -40 frontend/src/shared/tma/i18n.ts`
Confirm the catalog shape: a typed key union (e.g. `I18nKey`) and per-language record(s).

- [ ] **Step 2: Add the 6 keys to all three packs**

Add each key to the `I18nKey` union (if explicit) and to all three language records (ru, kk, en). Exact copy:

| Key                | ru                                           | kk                                                   | en                                  |
| ------------------ | -------------------------------------------- | ---------------------------------------------------- | ----------------------------------- |
| `errNotConfigured` | "Создание встреч не настроено"               | "Кездесулерді жоспарлау бапталмаған"                 | "Meeting creation isn't configured" |
| `errNotYours`      | "Это не ваша встреча"                        | "Бұл сіздің кездесуіңіз емес"                        | "Not your meeting"                  |
| `recurringSoon`    | "Повторяющиеся встречи скоро будут доступны" | "Қайталанатын кездесулер жақында қол жетімді болады" | "Recurring meetings coming soon"    |
| `errGeneric`       | "Что-то пошло не так"                        | "Бір нәрсе дұрыс болмады"                            | "Something went wrong"              |
| `updated`          | "Встреча обновлена"                          | "Кездесу жаңартылды"                                 | "Meeting updated"                   |
| `deleted`          | "Встреча удалена"                            | "Кездесу жойылды"                                    | "Meeting deleted"                   |

> If any of these keys already exist (e.g. `deleted` may exist from earlier delete-toast wiring), do NOT duplicate; leave the existing copy and continue.

- [ ] **Step 3: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/shared/tma/i18n.ts
git commit -m "feat(tma): i18n keys for write-path toasts (ru/kk/en)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 7: Frontend write fetchers

**Files:**

- Modify: `frontend/src/features/meetings/api.ts` (append write fetchers + types).

Typecheck-verified.

- [ ] **Step 1: Append the write surface**

Append to `frontend/src/features/meetings/api.ts` (reuse the existing private `MeetingDTO`/`toMeeting` defined earlier in the file):

```ts
export type MeetingInput = {
  dept: string
  type: string
  host: string
  date: string // YYYY-MM-DD
  start: string // HH:MM
  end: string // HH:MM
  recurrence: string
  desc: string
  participants: string[] // emails
}

export type MeetingPatch = Partial<{
  dept: string
  type: string
  host: string
  date: string
  start: string
  end: string
  desc: string
}>

export type Conflict = {
  email: string
  name: string
  title: string
  start: string
  end: string
}

export type ConflictsParams = {
  participants: string[]
  date: string
  start: string
  end: string
  excludeId?: string
}

export async function createMeeting(input: MeetingInput): Promise<Meeting> {
  const data = await apiFetch<{ meeting: MeetingDTO }>("/tma/meetings", {
    method: "POST",
    body: input,
  })
  return toMeeting(data.meeting)
}

export async function updateMeeting(
  id: string,
  patch: MeetingPatch
): Promise<Meeting> {
  const data = await apiFetch<{ meeting: MeetingDTO }>(`/tma/meetings/${id}`, {
    method: "PATCH",
    body: patch,
  })
  return toMeeting(data.meeting)
}

export async function deleteMeeting(id: string): Promise<void> {
  await apiFetch<void>(`/tma/meetings/${id}`, { method: "DELETE" })
}

export async function fetchConflicts(
  params: ConflictsParams
): Promise<Conflict[]> {
  const data = await apiFetch<{ conflicts: Conflict[] }>("/tma/conflicts", {
    method: "POST",
    body: {
      participants: params.participants,
      date: params.date,
      start: params.start,
      end: params.end,
      exclude_id: params.excludeId ?? "",
    },
  })
  return data.conflicts
}
```

> If `apiFetch` does not support `method: "DELETE"` returning `void`, inspect its overloads and adapt the call (e.g. `apiFetch<null>(...)` or a small `void` cast). Do not introduce a new HTTP client.

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/meetings/api.ts
git commit -m "feat(tma): write fetchers (create/update/delete/conflicts)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 8: Frontend write hooks

**Files:**

- Modify: `frontend/src/features/meetings/queries.ts` (add mutation hooks).

Typecheck-verified. On success, invalidate `tmaKeys.all` so every meetings-scoped query refetches.

- [ ] **Step 1: Append the hooks**

At the top, extend the existing `@tanstack/react-query` import to include `useMutation` and `useQueryClient`. Extend the existing `@/features/meetings/api` import to include the write fetchers + types: `createMeeting`, `updateMeeting`, `deleteMeeting`, `fetchConflicts`, `type MeetingInput`, `type MeetingPatch`, `type ConflictsParams`. Then append:

```ts
export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MeetingInput) => createMeeting(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: MeetingPatch }) =>
      updateMeeting(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMeeting(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useConflicts() {
  return useMutation({
    mutationFn: (params: ConflictsParams) => fetchConflicts(params),
  })
}
```

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/meetings/queries.ts
git commit -m "feat(tma): write mutation hooks + useConflicts (invalidate on success)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 9: Create-wizard hook — real conflicts + recurring guard

**Files:**

- Modify: `frontend/src/features/meeting-create/lib/use-create-wizard.ts` (replace client-side `conflictPeople` memo with `useConflicts()` mutation; add `recurringBlocked`).
- Modify: `frontend/src/features/meeting-create/components/wizard-step-review.tsx` (render real conflict names + recurring note; disable confirm when blocked).
- Modify: `frontend/src/features/meeting-create/components/create-wizard.tsx` (pass through `recurringBlocked` to the confirm-button gating; consumers may need a small prop addition).

Typecheck + build verified. The wizard is split — most changes land in the `use-create-wizard` hook (data) and the review-step component (render).

- [ ] **Step 1: Read the current hook**

Run: `cat frontend/src/features/meeting-create/lib/use-create-wizard.ts`
Identify (a) the `conflictPeople` `useMemo` (returns `string[]` of unique short names), (b) the place the hook reads `draft.rec`, (c) how the hook is consumed by `create-wizard.tsx` and `wizard-step-review.tsx` (which destructures `conflictPeople`).

- [ ] **Step 2: Replace the client-side conflict memo with `useConflicts`**

In `use-create-wizard.ts`:

1. Add imports:
   ```ts
   import { useEffect } from "react"
   import { useConflicts } from "@/features/meetings/queries"
   ```
2. Remove the `conflictPeople` `useMemo` and its `overlaps` helper.
3. Add a mutation + effect:
   ```ts
   const conflictsMut = useConflicts()
   useEffect(() => {
     if (
       WIZARD_STEPS[step] !== "review" ||
       !draft.date ||
       draft.participants.length === 0
     ) {
       return
     }
     conflictsMut.mutate({
       participants: draft.participants.map((p) => p.email),
       date: draft.date,
       start: draft.start,
       end: endTime,
     })
     // eslint-disable-next-line react-hooks/exhaustive-deps
   }, [step, draft.date, draft.start, endTime, draft.participants])
   ```
4. Derive the wizard's previously-`string[]` shape from `conflictsMut.data`:
   ```ts
   const conflictPeople = Array.from(
     new Set((conflictsMut.data ?? []).map((c) => c.name))
   )
   ```
   This preserves the existing render shape (`string[]` of unique names).
5. Add the recurring-guard:
   ```ts
   const recurringBlocked = draft.rec !== "once"
   ```
6. Return `recurringBlocked` from the hook in addition to existing fields.

- [ ] **Step 3: Thread `recurringBlocked` into the confirm gating**

In `create-wizard.tsx`:

1. Destructure `recurringBlocked` from `wizard`.
2. The bottom "next/confirm" button currently uses `disabled={!canNext}`. On the review step, combine: `disabled={!canNext || recurringBlocked}`. (Other steps stay unaffected — `recurringBlocked` only blocks the final confirm.)

- [ ] **Step 4: Render the `recurringSoon` note on the review step**

In `wizard-step-review.tsx`: when `recurringBlocked`, render a warning box above (or below) the existing conflict warning using `t("recurringSoon")`. Reuse the existing warning-box styling. Accept `recurringBlocked` as a new prop and pass it from the parent.

- [ ] **Step 5: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: PASS.

> Troubleshooting: if the conflict-fetch mutation re-runs too frequently (each `useEffect` tick on `draft.participants` ref change), wrap the effect's gate with a deep-equality memo OR a stable participant-emails key (`participants.map(p=>p.email).join(",")`).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/features/meeting-create/lib/use-create-wizard.ts \
        frontend/src/features/meeting-create/components/create-wizard.tsx \
        frontend/src/features/meeting-create/components/wizard-step-review.tsx
git commit -m "feat(tma): wizard uses real conflicts endpoint + recurring guard

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 10: `create-page.tsx` — real mutations + error mapper

**Files:**

- Modify: `frontend/src/features/meeting-create/pages/create-page.tsx`.
- Create: `frontend/src/features/meetings/lib/write-error.ts` (new helper file).

Typecheck-verified. This replaces the optimistic `queryClient.setQueryData` calls with real `useCreateMeeting`/`useUpdateMeeting` mutations and adds a localized error-mapper.

- [ ] **Step 1: Create the error-mapper helper**

Create `frontend/src/features/meetings/lib/write-error.ts`:

```ts
import { isAxiosError } from "axios"
import type { I18nKey } from "@/shared/tma/i18n"

// writeErrorKey maps a write-path error to a localized I18nKey. Never log the
// error object (it may carry sensitive request body fields).
export function writeErrorKey(err: unknown): I18nKey {
  if (!isAxiosError(err)) return "errGeneric"
  const status = err.response?.status ?? 0
  const data = err.response?.data as
    | { code?: string; message?: string }
    | undefined
  const code = data?.code ?? data?.message ?? ""
  if (code === "meetings_not_configured") return "errNotConfigured"
  if (code === "meetings_recurring_unsupported") return "recurringSoon"
  if (status === 403 || code === "forbidden") return "errNotYours"
  if (status === 404 || code === "not_found") return "errNotYours"
  return "errGeneric"
}
```

> Confirm `I18nKey` is exported from `frontend/src/shared/tma/i18n.ts`. If it isn't (it's an internal union), export it: add `export` to the type. That single export is fine — it's already the catalog's de-facto contract.

- [ ] **Step 2: Rewrite `create-page.tsx`'s submit branch**

Replace the entire body of `CreateMeetingPage` after the `useMyMeetings`/`editing` setup with the version below (preserves the existing overlay/navigation; drops the optimistic cache surgery):

```tsx
const createMut = useCreateMeeting()
const updateMut = useUpdateMeeting()

const completeCreate = async (m: MeetingDraft & { end: string }) => {
  const payload = {
    dept: m.dept,
    type: m.type,
    host: m.host,
    date: m.date,
    start: m.start,
    end: m.end,
    desc: m.desc,
  }
  try {
    if (editId) {
      await updateMut.mutateAsync({ id: editId, patch: payload })
      p.showToast(translate(p.lang, "updated"), "✏️")
      void navigate({ to: "/meetings", search: { scope: "upcoming" } })
    } else {
      const created = await createMut.mutateAsync({
        ...payload,
        recurrence: m.rec,
        participants: m.participants.map((x) => x.email),
      })
      void navigate({
        to: "/meetings",
        search: { scope: "upcoming", success: created.id },
      })
    }
  } catch (err) {
    p.showToast(translate(p.lang, writeErrorKey(err)), "⚠️")
  }
}
```

Add imports:

```ts
import { useCreateMeeting, useUpdateMeeting } from "@/features/meetings/queries"
import { writeErrorKey } from "@/features/meetings/lib/write-error"
```

Remove these now-unused imports:

- `useQueryClient`
- `draftToMeeting` (still used by `slotInitial`? if not, remove)
- `tmaKeys`
- `Meeting` (if unused)

Verify before removing — keep any that other code in the file still references.

- [ ] **Step 3: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: PASS.

> Troubleshooting: `mutateAsync` rejects on HTTP error — the try/catch handles it. If the navigate-on-success leaves stale state, lean on the query invalidation that the mutation hooks already perform on `tmaKeys.all`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/features/meeting-create/pages/create-page.tsx \
        frontend/src/features/meetings/lib/write-error.ts
git commit -m "feat(tma): real create/update mutations + localized error mapper

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 11: Meeting-detail delete + organizer-only action gating

**Files:**

- Modify: the meeting-detail component(s). Path TBV (read `frontend/src/features/meetings/components/` and `pages/` to find the detail card with Edit/Delete actions).

Typecheck-verified.

- [ ] **Step 1: Locate the detail surface**

Run: `grep -rn "openCreate\|deleteMeeting\|onEdit\|onDelete\|MeetingDetail" frontend/src/features/meetings/ frontend/src/features/meeting-create/ | head -30`
Expected: at least one file rendering Edit + Delete actions for a meeting card or sheet. (Earlier code used `MeetingDetail` in the tma-app shell; under the FSD restructure it likely lives in `features/meetings/components/` or `features/meetings/pages/`.)

- [ ] **Step 2: Wire the delete mutation + organizer gating**

In whichever component renders the Edit/Delete actions:

1. Add imports:
   ```ts
   import { useDeleteMeeting } from "@/features/meetings/queries"
   import { writeErrorKey } from "@/features/meetings/lib/write-error"
   import { useTmaAuth } from "@/features/auth/auth-context"
   import { useTmaApp } from "@/shared/tma/context"
   import { translate } from "@/shared/tma/i18n"
   ```
   (Adjust the auth import path if `useTmaAuth` is re-exported from `@/shared/tma/auth-context`.)
2. Add state + handler:

   ```ts
   const { user } = useTmaAuth()
   const p = useTmaApp()
   const deleteMut = useDeleteMeeting()
   const canModify = !!user && detail.organizer === user.email

   const onDelete = async () => {
     try {
       await deleteMut.mutateAsync(detail.id)
       p.showToast(translate(p.lang, "deleted"), "🗑️")
       // close the detail sheet / navigate as the existing flow does
     } catch (err) {
       p.showToast(translate(p.lang, writeErrorKey(err)), "⚠️")
     }
   }
   ```

3. Gate the Edit and Delete buttons in render: render them only when `canModify` is true.

- [ ] **Step 3: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/features/meetings/   # plus any other modified paths from this task
git commit -m "feat(tma): meeting detail — real delete + organizer-only actions

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 12: Docs refresh + final verification

**Files:**

- Modify: `docs/MEETINGS.md` (move write paths from Planned → Done).
- Modify: `docs/API.md` (move PATCH/DELETE/conflicts from Planned → Present table).

- [ ] **Step 1: `docs/MEETINGS.md` — update the engineering status**

Find the row(s) noting `PATCH /api/tma/meetings/:id`, `DELETE /api/tma/meetings/:id`, `POST /api/tma/conflicts` as Planned and mark them Done. Add a one-line note that the create wizard now surfaces real conflicts and gates recurring writes.

- [ ] **Step 2: `docs/API.md` — move planned → present**

Move the three routes out of the "TMA — planned" section into the "TMA — present" table with one-line purposes:

- `PATCH /api/tma/meetings/:id` — edit a single meeting (organizer/owner only).
- `DELETE /api/tma/meetings/:id` — cancel a single meeting (organizer/owner only).
- `POST /api/tma/conflicts` — cross-participant conflict warning for the create wizard's review step.

- [ ] **Step 3: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt: `cd backend && env -u GOROOT gofmt -w ./internal/...`. If the frontend build flags prettier: `pnpm -C frontend format`.

- [ ] **Step 4: Commit**

```bash
git add docs/MEETINGS.md docs/API.md
git commit -m "docs(tma): write paths done — refresh MEETINGS + API

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** `toConflictDTO` → T1; `POST /api/tma/conflicts` → T2; `editableWorkspace` + `PATCH /meetings/:id` → T3; `DELETE /meetings/:id` → T4; OpenAPI + frontend schema regen → T5; i18n → T6; write fetchers → T7; mutation hooks → T8; real conflicts in wizard + recurring guard → T9; real create/update mutations + error mapper → T10; delete mutation + organizer-only gating → T11; docs refresh → T12. All spec decisions land.
- **Type consistency:** backend `tmaConflictDTO`/`tmaConflictRequest`/`tmaUpdateRequest` + `toConflictDTO`/`editableWorkspace` defined T1-T3, reused T2-T4. Frontend `MeetingInput`/`MeetingPatch`/`Conflict`/`ConflictsParams` defined T7, consumed T8 hooks, threaded T9-T11. Query key `tmaKeys.all` invalidates the `["tma", "meetings", scope]` and `["tma", "schedule", …]` subtrees consistently.
- **Identity model:** every write resolves the organizer via `EnsureTMAOrganizer` (idempotent by `auth_sub="email:<email>"`), so editing/deleting authorizes correctly against `ownerOrOrganizer` whether the meeting was created via the bot, the web (legacy), or the Mini App.
- **Known approximations:** create still uses `wsIDs[0]` when multiple Google workspaces exist (documented in the create handler comment); `editableWorkspace` does an N-row scan per edit/delete (personal-scale, fine); conflicts are advisory (non-blocking).
- **Known unknown:** Task 11 needs to locate the meeting-detail component because the FSD pivot moved files; the task includes a grep step to find it before editing.
