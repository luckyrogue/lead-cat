> **Superseded paths:** implemented under `frontend/src/features/*`, `shared/api`, `features/auth`. See `frontend/README.md`.

# TMA Write Paths Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the authed Telegram user create, edit, and delete their own meetings from the Mini App, with a real cross-participant conflict warning on create — replacing the optimistic-only React Query cache writes from sub-project 2.

**Architecture:** Four new `/api/tma/*` write endpoints (TMA-auth) reuse the existing `CreateMeeting`/`UpdateMeeting`/`CancelMeeting`/`MeetingConflicts` commands unchanged. A new application helper `EnsureTMAOrganizer` bridges identity: it find-or-creates the `platform_users` row backing the TMA `bot_user` (by email, via the `email:<email>` auth_sub convention) and links the telegram id, returning the UUID used as meeting organizer. Create targets the single Google-configured workspace; edit/delete resolve the workspace from the meeting (via `ListEditableMeetings`, which doubles as the ownership guard). Scope is non-recurring create + single-meeting edit/delete; recurring + whole-series + reminder settings are deferred. The frontend swaps optimistic cache surgery for real mutations + `invalidateQueries`, threads an `editId` through the create overlay so "edit" PATCHes instead of duplicating, and calls the real conflicts endpoint on the wizard's review step.

**Tech Stack:** Go, Fiber, pgx/Postgres, zap; React, axios, @tanstack/react-query, Vite/pnpm.

**Spec:** `docs/superpowers/specs/2026-06-05-tma-write-paths-design.md`

## Codebase facts (verified — rely on these)

- **Module path:** `github.com/luckyrogue/lead-cat`.
- **TMA group** (`backend/internal/delivery/http/app.go:149-154`): `tma := app.Group("/api/tma", tmaAuth.Middleware)` then `tma.Get("/me"…)`, `/meetings`, `/schedule`, `/employees`, `/free-slots`. New write routes append here. The middleware sets `c.Locals("bot_user").(postgres.BotUser)`.
- **`postgres.BotUser`** (`models.go:101`): `ID uuid.UUID, TelegramID int64, FullName, Email, Role, ReminderMinutes string`.
- **Read-handler conventions** (`handlers/tma_read.go`): receiver `*API`; `a.App` is `*application.Services`; `botUserEmail(c) (string, bool)` reads email from `c.Locals("bot_user")`; `almatyLoc()` helper (in `handlers/meeting_availability.go`) returns `*time.Location`; `a.toMeetingDTO(ctx, m postgres.Meeting) tmaMeetingDTO` already maps a meeting → the UI DTO. **Reuse all of these.**
- **REST handler conventions** (`handlers/meetings.go`): package already imports `errors`, `application`, `postgres`, `uuid`, `copy`. Create maps `application.ErrInvalidInput` **and** `application.ErrGoogleNotConfigured` → `400`, else `500`, returns `201 JSON(m)`. Delete maps `application.ErrForbidden` → `403` (`copy.APIError("forbidden")`), returns `204`. **There is no REST update handler** — `UpdateMeeting` is currently only called from the bot; the TMA PATCH is the first HTTP caller.
- **Write commands (reuse unchanged):**
  - `CreateMeeting(ctx, workspaceID, organizerID uuid.UUID, in application.CreateMeetingInput) (postgres.Meeting, error)`. `CreateMeetingInput{Dept, Type, Host, Date, Start, End, Recurrence, RecurrenceUntil, Description string; Participants []postgres.MeetingParticipant}` (`meeting_service.go:20`). Returns `ErrInvalidInput`/`ErrGoogleNotConfigured`.
  - `UpdateMeeting(ctx, workspaceID, userID, meetingID uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error)` (`meeting_service.go:241`). `UpdateMeetingInput{Dept,Type,Host,Date,Start,End,Recurrence,Description *string}` (all nil = unchanged; date/start/end are a unit). Authorizes via `ownerOrOrganizer` → `ErrForbidden`.
  - `CancelMeeting(ctx, workspaceID, userID, id uuid.UUID) error` (`meeting_service.go:297`). Authorizes via `ownerOrOrganizer`; idempotent; emits `meeting:cancelled` DM.
  - `ListEditableMeetings(ctx, telegramID int64) ([]postgres.MeetingWithTZ, error)` (`meeting_service.go:293`). `MeetingWithTZ` embeds `postgres.Meeting` (so `.ID`, `.WorkspaceID` are promoted) + `TZ string`. Filter is `telegram_id=$1 AND status='scheduled' AND starts_at>now()`.
  - `MeetingConflicts(ctx, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]application.Conflict, error)` (`conflict.go:46`). `Conflict{Email, PersonName, MeetingName string; Start, End time.Time /*UTC*/}`. Pass `uuid.Nil` to exclude nothing.
- **Identity bridge primitives:**
  - `a.App.Store.UpsertUserIdentity(ctx, authSub, email, phone string) (postgres.User, error)` (`auth_repo.go:19`) — `User.ID uuid.UUID`, `User.Email`.
  - `platformauth.SubEmail(email) string` → `"email:"+lower(trim(email))` (`platform/auth/otp.go:99`). **Import alias:** the package is imported elsewhere as `platformauth "…/internal/platform/auth"`.
  - `(*application.Services).LinkTelegram(ctx, userID uuid.UUID, telegramID int64, username string) error` (`services.go:33`).
  - `a.App.Store.ListWorkspacesWithGoogle(ctx) ([]uuid.UUID, error)` (`employee_repo.go:73`).
- **Frontend** (`frontend/src/`):
  - `shared/tma/api.ts` — axios `api`; private `toMeeting(d: MeetingDTO): Meeting`; `MeetingDTO` type; existing read fetchers. (Same file — new write fetchers reuse `toMeeting`.)
  - `shared/tma/queries.ts` — `useMyMeetings(scope)` key `["tma","meetings",scope]`; `useQuery`/`useMutation` from `@tanstack/react-query`; `useTmaApp()` for `lang`.
  - `shared/tma/auth-context.tsx` — `useTmaAuth(): {status, user, retry}`, `user: {telegramId, name, email, role}`.
  - `shared/tma/types.ts` — `MeetingDraft{dept,type,host,date,start,dur,rec,recDays,participants:Employee[],desc,end?}`; `OverlayState = {type:"create"; initial?:Partial<MeetingDraft>} | {type:"colleague"} | {type:"admin"} | null`.
  - `shared/tma/meeting-utils.ts` — `draftToMeeting(draft,id)` (hardcodes `ME.email`), `detailToDraft(m)`.
  - `shared/tma/i18n.ts` — `translate(lang, key)`; `useTmaApp().t(key)`; add new copy keys here following the existing pattern.
  - `features/tma/tma-app.tsx` — `TmaContent` (~line 142): `meetings = useMyMeetings("all")` + `useQueryClient()`; `openCreate(initial?)` (157); `completeCreate` (160, does `draftToMeeting` + `setQueryData` + `SuccessView`); `deleteMeeting(id)` (183, filters cache + toast); `MeetingDetail` rendered (286) with `onEdit={() => { setDetail(null); openCreate(detailToDraft(detail)) }}` and `onDelete={() => deleteMeeting(detail.id)}`.
  - `features/tma/screens/create-wizard.tsx` — `CreateWizard({initial, onComplete, meetings})`; `STEPS=[what,when,who,review]`; draft default `host: ME.name` (271); `onComplete({...draft, end: endTime})` on final step (304); `conflictPeople` client-side memo (310-328) rendered in the warning block (867); `finalMeeting.organizer = ME.email` (346); recurrence chips (520) incl. `custom`/`recDays`.

## Conventions

- Backend: build/test/lint from repo root `make test && make lint && make build`; Go as `env -u GOROOT go ...` from `backend/`; `make lint` includes gofmt. Pure logic unit-tested; handlers/wiring build-verified (no HTTP/DB harness).
- Frontend: `pnpm -C frontend typecheck` + `pnpm -C frontend format` per task; full `make build` at the end. No test runner.
- Logging: a single `Info` lifecycle line per successful write (`tma_meeting_created`/`_updated`/`_cancelled` + `zap.Int64("telegram_id",…)`, `zap.String("meeting_id",…)`, `zap.String("workspace_id",…)`); no email/initData/JWT/PII. `a.App.Log` is the `*zap.Logger`.
- Commit messages end with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Don't touch `frontend/vite.config.ts`.

## File structure (created/modified)

- Create `backend/internal/application/tma_organizer.go` (`EnsureTMAOrganizer`).
- Create `backend/internal/delivery/http/handlers/tma_write.go` (request DTOs, pure mappers, 4 handlers) + `tma_write_test.go` (pure-mapper tests).
- Modify `backend/internal/delivery/http/app.go` (register 4 routes).
- Modify `frontend/src/shared/tma/api.ts` (write fetchers + request types) + `queries.ts` (write hooks) + `i18n.ts` (copy keys) + `types.ts` (`OverlayState.editId`).
- Modify `frontend/src/features/tma/screens/create-wizard.tsx` + `features/tma/tma-app.tsx`.
- Modify `docs/MEETINGS.md`.

---

## Task 1: Backend — `EnsureTMAOrganizer` identity bridge

**Files:**

- Create: `backend/internal/application/tma_organizer.go`

Build-verified (pure I/O over the Store; idempotency comes from `UpsertUserIdentity`/`LinkTelegram` semantics — no DB harness, per convention).

- [ ] **Step 1: Create the helper**

Create `backend/internal/application/tma_organizer.go`:

```go
package application

import (
	"context"

	"github.com/google/uuid"

	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

// EnsureTMAOrganizer find-or-creates the platform_users row backing a TMA user
// (by email, via the email:<email> auth_sub convention) and links the telegram id,
// returning the platform_users UUID used as a meeting organizer. Idempotent; it
// unifies with native email-OTP login (same auth_sub), so a meeting organized on
// the web is editable from the Mini App and vice-versa.
func (s *Services) EnsureTMAOrganizer(ctx context.Context, email string, telegramID int64) (uuid.UUID, error) {
	u, err := s.Store.UpsertUserIdentity(ctx, platformauth.SubEmail(email), email, "")
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.LinkTelegram(ctx, u.ID, telegramID, ""); err != nil {
		return uuid.Nil, err
	}
	return u.ID, nil
}
```

- [ ] **Step 2: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/ && env -u GOROOT gofmt -l internal/application/tma_organizer.go`
Expected: all clean (gofmt prints nothing).

> If `platformauth` is the wrong alias, grep an existing importer: `grep -rn "platform/auth\"" backend/internal | head`. The verified alias is `platformauth`.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application/tma_organizer.go
git commit -m "feat(tma): EnsureTMAOrganizer — lazy platform_users link for TMA writes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Backend — create endpoint (`POST /api/tma/meetings`)

**Files:**

- Create: `backend/internal/delivery/http/handlers/tma_write.go`
- Create: `backend/internal/delivery/http/handlers/tma_write_test.go`
- Modify: `backend/internal/delivery/http/app.go`

TDD on the pure request→`CreateMeetingInput` mapper; the handler + route are build-verified.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/delivery/http/handlers/tma_write_test.go`:

```go
package handlers

import "testing"

func TestToCreateMeetingInput(t *testing.T) {
	// Host falls back to the bot user's full name when empty; blank participant
	// emails are dropped; recurrence/desc pass through.
	in := toCreateMeetingInput(tmaCreateRequest{
		Dept: "Eng", Type: "weekly", Host: "", Date: "2026-06-10",
		Start: "10:00", End: "10:30", Recurrence: "once", Desc: "sync",
		Participants: []string{"a@x.io", "", "  ", "b@x.io"},
	}, "Real Name")
	if in.Host != "Real Name" {
		t.Fatalf("host fallback: %q", in.Host)
	}
	if len(in.Participants) != 2 || in.Participants[0].Email != "a@x.io" || in.Participants[1].Email != "b@x.io" {
		t.Fatalf("participants: %+v", in.Participants)
	}
	if in.Date != "2026-06-10" || in.Start != "10:00" || in.End != "10:30" || in.Recurrence != "once" || in.Description != "sync" {
		t.Fatalf("passthrough: %+v", in)
	}
	// Non-empty host is kept.
	if got := toCreateMeetingInput(tmaCreateRequest{Host: "Custom"}, "Real").Host; got != "Custom" {
		t.Fatalf("host kept: %q", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToCreateMeetingInput' -v`
Expected: FAIL — `undefined: toCreateMeetingInput` / `tmaCreateRequest`.

- [ ] **Step 3: Implement the request DTO, mapper, and create handler**

Create `backend/internal/delivery/http/handlers/tma_write.go`:

```go
package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type tmaCreateRequest struct {
	Dept         string   `json:"dept"`
	Type         string   `json:"type"`
	Host         string   `json:"host"`
	Date         string   `json:"date"`  // YYYY-MM-DD
	Start        string   `json:"start"` // HH:MM
	End          string   `json:"end"`   // HH:MM
	Recurrence   string   `json:"recurrence"`
	Desc         string   `json:"desc"`
	Participants []string `json:"participants"` // emails
}

// toCreateMeetingInput maps the TMA request to the application input. Pure: host
// falls back to the bot user's name when blank; blank participant emails are dropped.
func toCreateMeetingInput(req tmaCreateRequest, hostFallback string) application.CreateMeetingInput {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = hostFallback
	}
	parts := make([]postgres.MeetingParticipant, 0, len(req.Participants))
	for _, e := range req.Participants {
		if e = strings.TrimSpace(e); e != "" {
			parts = append(parts, postgres.MeetingParticipant{Email: e})
		}
	}
	return application.CreateMeetingInput{
		Dept: req.Dept, Type: req.Type, Host: host,
		Date: req.Date, Start: req.Start, End: req.End,
		Recurrence: req.Recurrence, Description: req.Desc,
		Participants: parts,
	}
}

// botUser returns the authed TMA bot_user from locals.
func botUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// TMACreateMeeting creates a non-recurring meeting for the authed TMA user.
func (a *API) TMACreateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if rec := strings.TrimSpace(req.Recurrence); rec != "" && rec != "once" {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_recurring_unsupported")
	}
	wsIDs, err := a.App.Store.ListWorkspacesWithGoogle(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if len(wsIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_not_configured")
	}
	workspaceID := wsIDs[0]
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.CreateMeeting(c.Context(), workspaceID, organizerID, toCreateMeetingInput(req, bu.FullName))
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) || errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.App.Log.Info("tma_meeting_created",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", m.ID.String()),
		zap.String("workspace_id", workspaceID.String()))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}
```

- [ ] **Step 4: Register the route in `app.go`**

After `tma.Post("/free-slots", api.TMAFreeSlots)` (app.go:154), add:

```go
	tma.Post("/meetings", api.TMACreateMeeting)
```

- [ ] **Step 5: Run test + build + vet + gofmt**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToCreateMeetingInput' -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: PASS; build/vet clean; gofmt prints nothing.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/handlers/tma_write_test.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): POST /api/tma/meetings (create, once-only)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Backend — edit + delete endpoints (`PATCH`/`DELETE /api/tma/meetings/:id`)

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write.go`
- Modify: `backend/internal/delivery/http/app.go`

Build-verified. Both resolve the meeting's workspace from `ListEditableMeetings` — which already filters to the caller's own scheduled future meetings, so it doubles as the ownership/recency guard (a `:id` not in that set → `404`). `UpdateMeeting`/`CancelMeeting` still enforce `ownerOrOrganizer` server-side (`403`).

- [ ] **Step 1: Add a workspace-resolver + the two handlers**

Append to `tma_write.go` (add `"github.com/google/uuid"` to the import block):

```go
// editableWorkspace returns the workspace of a meeting the TMA user may edit, or
// false if the meeting is not in their editable set (not theirs / not scheduled / past).
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

// TMAUpdateMeeting edits a single meeting the authed TMA user organizes.
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

// TMADeleteMeeting cancels a single meeting the authed TMA user organizes.
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

- [ ] **Step 2: Register routes in `app.go`**

After the create route:

```go
	tma.Patch("/meetings/:id", api.TMAUpdateMeeting)
	tma.Delete("/meetings/:id", api.TMADeleteMeeting)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): PATCH + DELETE /api/tma/meetings/:id (edit/cancel single)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Backend — conflicts endpoint (`POST /api/tma/conflicts`)

**Files:**

- Modify: `backend/internal/delivery/http/handlers/tma_write.go`
- Modify: `backend/internal/delivery/http/handlers/tma_write_test.go`
- Modify: `backend/internal/delivery/http/app.go`

TDD on the pure `Conflict → tmaConflictDTO` mapper; handler/route build-verified.

- [ ] **Step 1: Add the failing test**

Append to `tma_write_test.go`:

```go
func TestToConflictDTO(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 09:00–10:00 UTC == 14:00–15:00 Almaty
	c := application.Conflict{
		Email: "a@x.io", PersonName: "Alice", MeetingName: "Weekly",
		Start: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	d := toConflictDTO(c, loc)
	if d.Email != "a@x.io" || d.Name != "Alice" || d.Title != "Weekly" || d.Start != "14:00" || d.End != "15:00" {
		t.Fatalf("got %+v", d)
	}
}
```

Add the imports this test needs to the test file (`"time"` and the `application` package).

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToConflictDTO' -v`
Expected: FAIL — `undefined: toConflictDTO` / `tmaConflictDTO`.

- [ ] **Step 3: Implement the DTO, mapper, and handler**

Append to `tma_write.go` (add `"time"` to the import block):

```go
type tmaConflictRequest struct {
	Participants []string `json:"participants"`
	Date         string   `json:"date"`  // YYYY-MM-DD
	Start        string   `json:"start"` // HH:MM
	End          string   `json:"end"`   // HH:MM
	ExcludeID    string   `json:"exclude_id"`
}

type tmaConflictDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Start string `json:"start"` // HH:MM Almaty
	End   string `json:"end"`
}

// toConflictDTO renders a conflict's UTC times into Almaty HH:MM. Pure.
func toConflictDTO(c application.Conflict, loc *time.Location) tmaConflictDTO {
	return tmaConflictDTO{
		Email: c.Email, Name: c.PersonName, Title: c.MeetingName,
		Start: c.Start.In(loc).Format("15:04"),
		End:   c.End.In(loc).Format("15:04"),
	}
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
		if id, err := uuid.Parse(s); err == nil {
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

- [ ] **Step 4: Register the route in `app.go`**

```go
	tma.Post("/conflicts", api.TMAConflicts)
```

- [ ] **Step 5: Run tests + build + vet + gofmt**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'ToCreateMeetingInput|ToConflictDTO' -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: PASS; build/vet clean; gofmt empty.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_write.go backend/internal/delivery/http/handlers/tma_write_test.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): POST /api/tma/conflicts (create-wizard warning)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Frontend — write fetchers

**Files:**

- Modify: `frontend/src/shared/tma/api.ts`

Typecheck-verified. Reuse the existing private `toMeeting` mapper (same file).

- [ ] **Step 1: Append the write fetchers + types to `api.ts`**

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

export async function createMeeting(input: MeetingInput): Promise<Meeting> {
  const res = await api.post<{ meeting: MeetingDTO }>("/tma/meetings", input)
  return toMeeting(res.data.meeting)
}

export async function updateMeeting(
  id: string,
  patch: MeetingPatch
): Promise<Meeting> {
  const res = await api.patch<{ meeting: MeetingDTO }>(
    `/tma/meetings/${id}`,
    patch
  )
  return toMeeting(res.data.meeting)
}

export async function deleteMeeting(id: string): Promise<void> {
  await api.delete(`/tma/meetings/${id}`)
}

export type Conflict = {
  email: string
  name: string
  title: string
  start: string
  end: string
}

export type ConflictsParams = {
  participants: string[]
  date: string // YYYY-MM-DD
  start: string // HH:MM
  end: string // HH:MM
  excludeId?: string
}

export async function fetchConflicts(
  params: ConflictsParams
): Promise<Conflict[]> {
  const res = await api.post<{ conflicts: Conflict[] }>("/tma/conflicts", {
    participants: params.participants,
    date: params.date,
    start: params.start,
    end: params.end,
    exclude_id: params.excludeId ?? "",
  })
  return res.data.conflicts
}
```

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: typecheck passes; prettier writes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/api.ts
git commit -m "feat(tma): write fetchers (create/update/delete meeting, conflicts)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Frontend — write hooks (mutations + conflicts)

**Files:**

- Modify: `frontend/src/shared/tma/queries.ts`

Typecheck-verified. On success, invalidate the meetings cache so every scope refetches (replaces the optimistic-only updates).

- [ ] **Step 1: Append the hooks to `queries.ts`**

Add `useQueryClient` to the `@tanstack/react-query` import, and the new fetchers/types to the `./api` import, then:

```ts
export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MeetingInput) => createMeeting(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tma", "meetings"] }),
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: MeetingPatch }) =>
      updateMeeting(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tma", "meetings"] }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMeeting(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tma", "meetings"] }),
  })
}

export function useConflicts() {
  return useMutation({
    mutationFn: (params: ConflictsParams) => fetchConflicts(params),
  })
}
```

Import additions from `./api`: `createMeeting`, `updateMeeting`, `deleteMeeting`, `fetchConflicts`, `type MeetingInput`, `type MeetingPatch`, `type ConflictsParams`.

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/queries.ts
git commit -m "feat(tma): write mutation hooks + useConflicts (invalidate on success)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Frontend — wizard uses real user, real conflicts, recurring guard

**Files:**

- Modify: `frontend/src/features/tma/screens/create-wizard.tsx`
- Modify: `frontend/src/shared/tma/i18n.ts`

Typecheck + build verified. The wizard does **not** learn about `editId` — `onComplete(draft)` is unchanged; `tma-app` decides create-vs-edit from the overlay state (Task 8).

- [ ] **Step 1: Add i18n copy keys**

Read `frontend/src/shared/tma/i18n.ts` first. Add keys for each language (ru/kk/en) following the existing structure:

- `recurringSoon` — e.g. ru "Повторяющиеся встречи скоро будут доступны", en "Recurring meetings coming soon", kk equivalent.
- `errNotConfigured` — "Создание встреч не настроено" / "Meeting creation isn't configured" / kk.
- `errNotYours` — "Это не ваша встреча" / "Not your meeting" / kk.
- `errGeneric` — "Что-то пошло не так" / "Something went wrong" / kk.

(These are consumed in Tasks 7–8.)

- [ ] **Step 2: Replace mock `ME` with the real authed user**

In `create-wizard.tsx`:

- Add `import { useTmaAuth } from "@/shared/tma/auth-context"`.
- Inside `CreateWizard`, `const { user } = useTmaAuth()`.
- Draft default `host: ME.name` → `host: initial?.host ?? user?.name ?? ""` (keep the `...initial` spread; ensure host still wins from `initial` when editing).
- `finalMeeting` (line 346) `organizer: ME.email` → `organizer: user?.email ?? ""`.
- Remove the now-unused `ME` import if nothing else in the file uses it (the "я" host badge stays — it's static text).

- [ ] **Step 3: Replace client-side conflict check with the real endpoint**

- Add `import { useConflicts } from "@/shared/tma/queries"`.
- Remove the `conflictPeople` `useMemo` (lines ~310-328).
- Add `const conflictsMut = useConflicts()`.
- When the wizard reaches the review step, fire the mutation once with the current draft. Use an effect keyed on the review step + relevant draft fields:
  ```tsx
  useEffect(() => {
    if (
      STEPS[step] !== "review" ||
      !draft.date ||
      draft.participants.length === 0
    )
      return
    conflictsMut.mutate({
      participants: draft.participants.map((x) => x.email),
      date: draft.date,
      start: draft.start,
      end: endTime,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, draft.date, draft.start, endTime, draft.participants])
  ```
- Derive the warning list from `conflictsMut.data ?? []`. Replace the `conflictPeople.length > 0` block (line ~867) to render names from the conflicts (e.g. unique `c.name`), and the confirm-button label logic (line ~932) to use `(conflictsMut.data?.length ?? 0) > 0` instead of `conflictPeople.length`. Keep the warning non-blocking (confirm still proceeds).

> The conflict warning stays informational. The wizard already had a "proceed anyway" affordance — preserve it.

- [ ] **Step 4: Recurring guard on the review step**

- Compute `const recurringBlocked = draft.rec !== "once"`.
- On the review step, when `recurringBlocked`, render a note using `t("recurringSoon")` (reuse the warning box styling) and set the primary confirm button `disabled={recurringBlocked}` (combine with the existing `!canNext`). Non-review steps are unaffected, so the user can still navigate and change recurrence back to "once".

- [ ] **Step 5: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: all pass.

> Troubleshooting: if `useEffect` isn't imported, add it to the `react` import. If removing `ME`/`conflictPeople` leaves an unused import/var, delete it. Keep `meetings` prop in the signature for now (Task 8 may drop it) — an unused prop is not a TS error with the current config, but if `noUnusedParameters`/lint flags it, prefix `_meetings` or remove it and its call-site together in Task 8.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/features/tma/screens/create-wizard.tsx frontend/src/shared/tma/i18n.ts
git commit -m "feat(tma): wizard uses real user + live conflicts, guards recurring create

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Frontend — wire create/edit/delete mutations in `tma-app`

**Files:**

- Modify: `frontend/src/shared/tma/types.ts`
- Modify: `frontend/src/features/tma/tma-app.tsx`

Typecheck + build verified. This replaces the optimistic cache surgery with real mutations and threads `editId` so "edit" PATCHes.

- [ ] **Step 1: Add `editId` to the create overlay state**

In `types.ts`, change the create variant:

```ts
export type OverlayState =
  | { type: "create"; initial?: Partial<MeetingDraft>; editId?: string }
  | { type: "colleague" }
  | { type: "admin" }
  | null
```

- [ ] **Step 2: Wire the mutations in `TmaContent`**

In `tma-app.tsx`:

- Add `import { useTmaAuth } from "@/shared/tma/auth-context"` and `const { user } = useTmaAuth()`.
- Add `import { useCreateMeeting, useUpdateMeeting, useDeleteMeeting, useMyMeetings } from "@/shared/tma/queries"` (extend the existing import) and instantiate: `const createMut = useCreateMeeting()`, `const updateMut = useUpdateMeeting()`, `const deleteMut = useDeleteMeeting()`.
- `openCreate` gains an optional `editId`:
  ```tsx
  const openCreate = (initial?: Partial<MeetingDraft>, editId?: string) =>
    setOverlay({ type: "create", initial, editId })
  ```
- Replace `completeCreate` body. Read the current `editId` from the overlay BEFORE closing it, then branch:
  ```tsx
  const completeCreate = async (m: MeetingDraft & { end: string }) => {
    const editId = overlay?.type === "create" ? overlay.editId : undefined
    const payload = {
      dept: m.dept, type: m.type, host: m.host, date: m.date,
      start: m.start, end: m.end, desc: m.desc,
    }
    try {
      if (editId) {
        await updateMut.mutateAsync({ id: editId, patch: payload })
        setOverlay(null)
        p.showToast(/* localized "Meeting updated" */, "✏️")
      } else {
        const created = await createMut.mutateAsync({
          ...payload,
          recurrence: m.rec,
          participants: m.participants.map((x) => x.email),
        })
        setOverlay(null)
        setTab("meetings")
        setBurst(true)
        setTimeout(() => setBurst(false), 1100)
        setSuccess(created)
      }
    } catch (err) {
      p.showToast(writeErrorMessage(err, p), "⚠️")
    }
  }
  ```
  Keep the `SuccessView`/paw-burst flow for create (now fed by the server-returned `created`). Drop the `draftToMeeting` import/usage here.
- Replace `deleteMeeting`:
  ```tsx
  const deleteMeeting = async (id: string) => {
    try {
      await deleteMut.mutateAsync(id)
      setDetail(null)
      p.showToast(/* existing localized "Meeting deleted" */, "🗑️")
    } catch (err) {
      p.showToast(writeErrorMessage(err, p), "⚠️")
    }
  }
  ```
- Add a small `writeErrorMessage(err, p)` helper (module-scope in this file) that maps the backend error code in the axios response to copy: `meetings_not_configured` → `t("errNotConfigured")`, `403`/`forbidden` → `t("errNotYours")`, else `t("errGeneric")`. Use `isAxiosError` from `axios`; read `err.response?.data?.message` or status. Keep it defensive (no PII logging).

- [ ] **Step 3: Thread `editId` from the detail sheet**

In the `MeetingDetail` render (line ~286):

- `onEdit`: `setDetail(null); openCreate(detailToDraft(detail), detail.id)`.
- Gate the edit/delete actions so only the organizer sees them: pass a prop (e.g. `canModify={detail.organizer === user?.email}`) to `MeetingDetail` and have it hide/disable the edit + delete buttons when false. Read `MeetingDetail`'s current props first; if it doesn't accept such a flag, add an optional `canModify?: boolean` prop (default true) and guard the action buttons. (The backend still returns `403`/`404`, so this is UX-only.)

- [ ] **Step 4: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: all pass.

> Troubleshooting: `mutateAsync` rejects on HTTP error — the try/catch handles it. If `draftToMeeting` becomes unused project-wide, leave it in `meeting-utils.ts` (other code/tests may import it) but remove its import from `tma-app.tsx`. If `meetings` is still passed to `CreateWizard` and the wizard no longer uses it, remove the prop from both the JSX and the wizard signature in this task. The `SuccessView` expects a `Meeting`; the server returns exactly that shape via `toMeeting`.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/shared/tma/types.ts frontend/src/features/tma/tma-app.tsx
git commit -m "feat(tma): real create/edit/delete mutations + edit threading + organizer-only actions

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Docs + final verification

**Files:**

- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

After the read-paths "done" line, add:

```markdown
> **Mini App write paths (frontend integration #3, done):** four TMA-auth write endpoints — `POST /api/tma/meetings` (create, non-recurring; `EnsureTMAOrganizer` lazily find-or-creates the `platform_users` row backing the bot_user by email + links telegram_id, then reuses `CreateMeeting` against the single Google-configured workspace via `ListWorkspacesWithGoogle`), `PATCH /api/tma/meetings/:id` and `DELETE /api/tma/meetings/:id` (single-meeting edit/cancel; workspace + ownership resolved from `ListEditableMeetings`, reusing `UpdateMeeting`/`CancelMeeting`), and `POST /api/tma/conflicts` (real cross-participant warning via `MeetingConflicts`). Frontend: `createMeeting`/`updateMeeting`/`deleteMeeting`/`fetchConflicts` fetchers + `useCreateMeeting`/`useUpdateMeeting`/`useDeleteMeeting`/`useConflicts` hooks (invalidate `["tma","meetings"]` on success, replacing sub-project 2's optimistic cache writes); the create wizard uses the real authed user, shows live conflicts on review, and threads an `editId` through the overlay so "edit" PATCHes instead of duplicating; edit/delete actions are organizer-only in the UI (backend enforces 403/404). Deferred: recurring create + whole-series edit/delete (the wizard collects no `recurrence_until`; backend `400 meetings_recurring_unsupported`) and reminder settings (→ sub-project 4).
```

- [ ] **Step 2: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt: `cd backend && env -u GOROOT gofmt -w ./internal/...` and re-run. If the frontend build flags prettier: `pnpm -C frontend format` and re-run.

- [ ] **Step 3: Commit**

```bash
git add docs/MEETINGS.md
git commit -m "docs(tma): document Mini App write paths (frontend integration #3)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** identity bridge (`EnsureTMAOrganizer`) → Task 1; `POST /meetings` create + once-only guard + Google-workspace resolution → Task 2; `PATCH`/`DELETE /meetings/:id` single edit/cancel + `ListEditableMeetings` ownership guard → Task 3; `POST /conflicts` → Task 4; write fetchers → Task 5; mutation + conflicts hooks (invalidate) → Task 6; wizard real-user + live conflicts + recurring guard → Task 7; mutation wiring + `editId` threading + organizer-only actions → Task 8; docs → Task 9.
- **Type consistency:** backend `tmaCreateRequest`/`tmaUpdateRequest`/`tmaConflictRequest`/`tmaConflictDTO` + `toCreateMeetingInput`/`toConflictDTO`/`editableWorkspace`/`botUser` (Tasks 2-4) reuse `tmaMeetingDTO` + `toMeetingDTO` + `almatyLoc` from `tma_read.go`. Frontend `MeetingInput`/`MeetingPatch`/`Conflict`/`ConflictsParams` (Task 5) consumed by hooks (Task 6); query key `["tma","meetings"]` (prefix) invalidates the `["tma","meetings","all"]` list. `OverlayState.editId` (Task 8) read in `completeCreate`.
- **Identity model:** every TMA write resolves the organizer via `EnsureTMAOrganizer` (idempotent by `auth_sub="email:<email>"`), so editing/deleting authorizes correctly against `ownerOrOrganizer` for meetings the user organized via TMA _or_ the web (same row). Participant-only meetings → `ownerOrOrganizer` false → `403` (and they aren't in `ListEditableMeetings`, so they `404` first).
- **Known deferrals (in scope as 400/UX guards, full feature later):** recurring create (`rec != once` → 400 + disabled confirm), whole-series edit/delete (single only), participant edit (UpdateMeetingInput carries none), multi-workspace targeting (first Google workspace), reminder settings (sub-project 4).
- **Known approximations:** create uses `wsIDs[0]` when multiple Google workspaces exist (documented); `editableWorkspace` does an N-row scan of the user's editable meetings per edit/delete (personal-scale, fine); conflicts are advisory (non-blocking), matching §4.7.3.

```

```
