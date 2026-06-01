# TMA Read Paths Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the Mini App's read-only screens (my meetings, employee search, colleague schedule, free-slots checker) to the backend through the sub-project-1 TMA auth, replacing mock data.

**Architecture:** Four new `/api/tma/*` read endpoints (TMA-auth, global-by-email) reuse existing application methods and return UI-shaped DTOs via a backend mapper. The frontend gets typed fetchers + React Query hooks; `tma-app` fetches the meetings list (`scope=all`) and passes it to the screens (which keep client-side upcoming/past/all filtering). A 401→re-login axios interceptor is installed. Writes stay client-side (optimistic React Query cache updates) until sub-project 3.

**Tech Stack:** Go, Fiber, pgx/Postgres, zap; React, axios, @tanstack/react-query, Vite/pnpm.

**Spec:** `docs/superpowers/specs/2026-06-01-tma-read-paths-design.md`

## Codebase facts (verified — rely on these)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`.
- **TMA group** (`backend/internal/delivery/http/app.go`): after sub-project 1 there is `tmaAuth := middleware.NewTMAAuth(tmaToken, store)` then `tma := app.Group("/api/tma", tmaAuth.Middleware)` then `tma.Get("/me", api.TMAMe)`. New read routes are added to this `tma` group.
- **TMA auth context:** the middleware sets `c.Locals("bot_user").(postgres.BotUser)` — `BotUser{TelegramID int64, FullName, Email, Role string, ...}`.
- **Application methods** (receiver `*application.Services`, field `a.App` on the handler `*API`):
  - `EmployeeSchedule(ctx, email string, from, to time.Time) ([]postgres.Meeting, error)` (participant OR organizer by email; `starts_at` in `[from,to)`; participants NOT hydrated).
  - `SearchEmployeesGlobal(ctx, query string) ([]postgres.Employee, error)` (cap 20).
  - `FreeSlots(ctx, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)`; `application.FreeSlot{Day time.Time /*start-of-day Almaty*/, Start, End time.Time /*UTC*/, Mins int}`.
  - `a.App.Store.ListParticipants(ctx, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)` — `MeetingParticipant{EmployeeID *uuid.UUID, Email string}`.
  - `a.App.Store.GetUserByID(ctx, id uuid.UUID) (postgres.User, error)` → `User.Email` (organizer resolution; `OrganizerUserID` is `*uuid.UUID`).
- **`postgres.Meeting`**: `ID uuid.UUID, OrganizerUserID *uuid.UUID, Dept, Type, Host string, StartsAt, EndsAt time.Time, Recurrence, Name, Description, MeetLink, Status string, Participants []MeetingParticipant`. **`postgres.Employee`**: `ID uuid.UUID, FullName, Email, Dept string, HasTelegram bool`.
- **Handlers**: receiver `*API`; `almatyLoc()` helper exists in `handlers/meeting_availability.go` (returns `*time.Location` for Asia/Almaty). Errors via `fiber.NewError`. Package imports `postgres`, `time`, `strings` already in sibling files.
- **Frontend**: `shared/api/client.ts` exports `api` (axios, baseURL `/api`) + `setAuthToken`. `shared/tma/auth.ts` exports `tmaLogin(initData)`, `getInitData()`. `shared/tma/auth-context.tsx` `TmaAuthProvider`. `shared/tma/context.tsx` `useTmaApp()` → `{lang, ...}`. `shared/tma/meeting-utils.ts` `fmtDate(iso: string, lang: Lang): string`. Types (`shared/tma/types.ts`): `Meeting{id,type,dept,host,date,start,end,rec,recDays?,organizer,participants:string[],desc?}`, `Employee{id,name,email,dept,tg,role?}`, `FreeSlot{day,iso,start,end,mins}`, `Lang`. `@tanstack/react-query` wired at `app/providers.tsx` (`QueryClientProvider`); `useQuery`/`useMutation`/`useQueryClient` available.
- **Screens** (prop-driven): `HomeScreen({meetings: Meeting[], onMeeting, ...})`, `MeetingsScreen({meetings: Meeting[], onMeeting, ...})` + `MeetingDetail`, `CheckerScreen({...})`, colleague-schedule inside `profile-screen.tsx`. `tma-app.tsx` (the `TmaContent` component, ~line 140) holds `const [meetings, setMeetings] = useState<Meeting[]>(INITIAL_MEETINGS)` and passes `meetings` down; create (`completeCreate`) and delete (`deleteMeeting`) call `setMeetings`.

## Conventions

- Backend: build/test/lint from repo root `make test && make lint && make build`; Go as `env -u GOROOT go ...` from `backend/`; `make lint` includes gofmt. Pure logic unit-tested; handlers/wiring build-verified.
- Frontend: `pnpm -C frontend typecheck` + `pnpm -C frontend format` per task; full `make build` at the end. No test runner.
- Commit messages end with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Don't touch `frontend/vite.config.ts`.

## File structure (created/modified)

- Create `backend/internal/delivery/http/handlers/tma_read.go` (DTOs, pure helpers, mapper, 4 handlers) + `tma_read_test.go` (pure-helper tests).
- Modify `backend/internal/delivery/http/app.go` (register 4 routes).
- Create `frontend/src/shared/tma/api.ts` (typed fetchers) + `frontend/src/shared/tma/queries.ts` (hooks).
- Modify `frontend/src/shared/tma/auth.ts` (401 interceptor) + `auth-context.tsx` (install it).
- Modify `frontend/src/features/tma/tma-app.tsx` + `screens/{home,meetings,checker,profile}-screen.tsx` (wire reads).
- Modify `docs/MEETINGS.md`.

---

## Task 1: Backend pure helpers + DTOs

**Files:**
- Create: `backend/internal/delivery/http/handlers/tma_read.go`
- Test: `backend/internal/delivery/http/handlers/tma_read_test.go`

TDD on the two pure helpers; the DTO structs ride along (used by later tasks).

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/delivery/http/handlers/tma_read_test.go`:

```go
package handlers

import (
	"testing"
	"time"
)

func TestSplitMeetingTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 2026-06-01 14:00–15:00 Almaty == 09:00–10:00 UTC
	s := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	e := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	date, start, end := splitMeetingTime(s, e, loc)
	if date != "2026-06-01" || start != "14:00" || end != "15:00" {
		t.Fatalf("got %q %q %q", date, start, end)
	}
}

func TestTmaScopeWindow(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	if f, to, ok := tmaScopeWindow("upcoming", now); !ok || !f.Equal(now) || to.Before(now.AddDate(0, 0, 364)) {
		t.Fatalf("upcoming: %v %v %v", f, to, ok)
	}
	if f, to, ok := tmaScopeWindow("past", now); !ok || !to.Equal(now) || f.After(now.AddDate(0, 0, -364)) {
		t.Fatalf("past: %v %v %v", f, to, ok)
	}
	if f, to, ok := tmaScopeWindow("all", now); !ok || !f.Before(now) || !to.After(now) {
		t.Fatalf("all: %v %v %v", f, to, ok)
	}
	if _, _, ok := tmaScopeWindow("bogus", now); ok {
		t.Fatal("bogus scope should not be ok")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'SplitMeetingTime|TmaScopeWindow' -v`
Expected: FAIL — `undefined: splitMeetingTime` / `tmaScopeWindow`.

- [ ] **Step 3: Implement helpers + DTOs**

Create `backend/internal/delivery/http/handlers/tma_read.go`:

```go
package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type tmaMeetingDTO struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Dept         string   `json:"dept"`
	Host         string   `json:"host"`
	Date         string   `json:"date"`  // YYYY-MM-DD, Almaty
	Start        string   `json:"start"` // HH:MM, Almaty
	End          string   `json:"end"`   // HH:MM, Almaty
	Rec          string   `json:"rec"`
	Organizer    string   `json:"organizer"`    // email
	Participants []string `json:"participants"` // emails
	Desc         string   `json:"desc"`
	MeetLink     string   `json:"meet_link"`
	Status       string   `json:"status"`
}

type tmaEmployeeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
	Tg    bool   `json:"tg"`
}

type tmaFreeSlotDTO struct {
	ISO   string `json:"iso"`   // YYYY-MM-DD, Almaty
	Start string `json:"start"` // HH:MM, Almaty
	End   string `json:"end"`   // HH:MM, Almaty
	Mins  int    `json:"mins"`
}

// splitMeetingTime renders a meeting's UTC start/end into Almaty-local date + times.
func splitMeetingTime(startsAt, endsAt time.Time, loc *time.Location) (date, start, end string) {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	return s.Format("2006-01-02"), s.Format("15:04"), e.Format("15:04")
}

// tmaScopeWindow maps a scope to a [from,to) window around now (ListScheduleForEmail
// filters starts_at in [from,to)). Unknown scope → ok=false.
func tmaScopeWindow(scope string, now time.Time) (from, to time.Time, ok bool) {
	const horizon = 365
	switch scope {
	case "upcoming":
		return now, now.AddDate(0, 0, horizon), true
	case "past":
		return now.AddDate(0, 0, -horizon), now, true
	case "all":
		return now.AddDate(0, 0, -horizon), now.AddDate(0, 0, horizon), true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// toMeetingDTO maps a meeting to the UI-shaped DTO, resolving organizer email and
// participant emails (N+1 per meeting; fine for personal-scale lists).
func (a *API) toMeetingDTO(ctx context.Context, m postgres.Meeting) tmaMeetingDTO {
	loc := almatyLoc()
	date, start, end := splitMeetingTime(m.StartsAt, m.EndsAt, loc)
	organizer := ""
	if m.OrganizerUserID != nil {
		if u, err := a.App.Store.GetUserByID(ctx, *m.OrganizerUserID); err == nil {
			organizer = u.Email
		}
	}
	emails := []string{}
	if parts, err := a.App.Store.ListParticipants(ctx, m.ID); err == nil {
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
	}
	return tmaMeetingDTO{
		ID: m.ID.String(), Type: m.Type, Dept: m.Dept, Host: m.Host,
		Date: date, Start: start, End: end, Rec: m.Recurrence,
		Organizer: organizer, Participants: emails, Desc: m.Description,
		MeetLink: m.MeetLink, Status: m.Status,
	}
}

func (a *API) toMeetingDTOs(ctx context.Context, ms []postgres.Meeting) []tmaMeetingDTO {
	out := make([]tmaMeetingDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, a.toMeetingDTO(ctx, m))
	}
	return out
}

// botUserEmail returns the authed TMA user's email, or "" if absent.
func botUserEmail(c *fiber.Ctx) (string, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok {
		return "", false
	}
	return bu.Email, true
}
```

> Task 1 does not import `strings` (nothing here uses it); Task 2 adds that import when `TMASchedule` uses `strings.TrimSpace`.

- [ ] **Step 4: Run tests + build**

Run: `cd backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run 'SplitMeetingTime|TmaScopeWindow' -v && env -u GOROOT go build ./... && env -u GOROOT gofmt -l internal/delivery/http/handlers/tma_read.go`
Expected: PASS; build clean; gofmt empty.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_read.go backend/internal/delivery/http/handlers/tma_read_test.go
git commit -m "feat(tma): read DTOs + pure mapper helpers (splitMeetingTime, tmaScopeWindow)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Meeting-list handlers (my meetings + colleague schedule) + routes

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_read.go`
- Modify: `backend/internal/delivery/http/app.go`

Build-verified. `TMASchedule` uses `strings.TrimSpace`, so **add `"strings"` to the `tma_read.go` import block** in this task.

- [ ] **Step 1: Add the two handlers**

Append the two handlers to `tma_read.go`:

```go
// TMAMyMeetings lists the authed user's meetings for a scope window.
func (a *API) TMAMyMeetings(c *fiber.Ctx) error {
	email, ok := botUserEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	from, to, ok := tmaScopeWindow(c.Query("scope"), time.Now())
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scope")
	}
	ms, err := a.App.EmployeeSchedule(c.Context(), email, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meetings": a.toMeetingDTOs(c.Context(), ms)})
}

// TMASchedule lists a colleague's meetings (read-only directory feature, §4.6).
func (a *API) TMASchedule(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email required")
	}
	from, to, ok := tmaScopeWindow(c.Query("scope"), time.Now())
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scope")
	}
	ms, err := a.App.EmployeeSchedule(c.Context(), email, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meetings": a.toMeetingDTOs(c.Context(), ms)})
}
```

Add `"strings"` to the import block (used by `TMASchedule`).

- [ ] **Step 2: Register routes in `app.go`**

In `app.go`, after `tma.Get("/me", api.TMAMe)`, add:

```go
	tma.Get("/meetings", api.TMAMyMeetings)
	tma.Get("/schedule", api.TMASchedule)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_read.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): GET /api/tma/meetings + /schedule (my + colleague)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Employees search + free-slots handlers + routes

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_read.go`
- Modify: `backend/internal/delivery/http/app.go`

Build-verified. (`strings` is already imported as of Task 2.)

- [ ] **Step 1: Add the two handlers**

Append to `tma_read.go`:

```go
// TMAEmployees searches the global directory (empty q → empty list).
func (a *API) TMAEmployees(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	q := strings.TrimSpace(c.Query("q"))
	out := []tmaEmployeeDTO{}
	if q != "" {
		emps, err := a.App.SearchEmployeesGlobal(c.Context(), q)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
		for _, e := range emps {
			out = append(out, tmaEmployeeDTO{ID: e.ID.String(), Name: e.FullName, Email: e.Email, Dept: e.Dept, Tg: e.HasTelegram})
		}
	}
	return c.JSON(fiber.Map{"employees": out})
}

type tmaFreeSlotsRequest struct {
	Participants []string `json:"participants"`
	From         string   `json:"from"` // YYYY-MM-DD (inclusive)
	To           string   `json:"to"`   // YYYY-MM-DD (inclusive)
	DurationMins int      `json:"duration_mins"`
}

// TMAFreeSlots finds common free time across participants (§4.8).
func (a *API) TMAFreeSlots(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaFreeSlotsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	from, err1 := time.ParseInLocation("2006-01-02", req.From, loc)
	toIncl, err2 := time.ParseInLocation("2006-01-02", req.To, loc)
	if err1 != nil || err2 != nil || toIncl.Before(from) || req.DurationMins <= 0 || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/participants/duration")
	}
	slots, err := a.App.FreeSlots(c.Context(), req.Participants, from, toIncl.AddDate(0, 0, 1), req.DurationMins)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	out := make([]tmaFreeSlotDTO, 0, len(slots))
	for _, sl := range slots {
		out = append(out, tmaFreeSlotDTO{
			ISO:   sl.Day.In(loc).Format("2006-01-02"),
			Start: sl.Start.In(loc).Format("15:04"),
			End:   sl.End.In(loc).Format("15:04"),
			Mins:  sl.Mins,
		})
	}
	return c.JSON(fiber.Map{"slots": out})
}
```

- [ ] **Step 2: Register routes in `app.go`**

After the routes added in Task 2:

```go
	tma.Get("/employees", api.TMAEmployees)
	tma.Post("/free-slots", api.TMAFreeSlots)
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_read.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): GET /api/tma/employees + POST /api/tma/free-slots

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Frontend typed fetchers

**Files:**
- Create: `frontend/src/shared/tma/api.ts`

Typecheck-verified.

- [ ] **Step 1: Create the fetchers**

Create `frontend/src/shared/tma/api.ts`:

```ts
import { api } from "@/shared/api/client"
import { fmtDate } from "./meeting-utils"
import type { Employee, FreeSlot, Lang, Meeting } from "./types"

export type Scope = "upcoming" | "past" | "all"

type MeetingDTO = {
  id: string
  type: string
  dept: string
  host: string
  date: string
  start: string
  end: string
  rec: string
  organizer: string
  participants: string[]
  desc: string
  meet_link: string
  status: string
}
type EmployeeDTO = { id: string; name: string; email: string; dept: string; tg: boolean }
type FreeSlotDTO = { iso: string; start: string; end: string; mins: number }

function toMeeting(d: MeetingDTO): Meeting {
  return {
    id: d.id,
    type: d.type,
    dept: d.dept,
    host: d.host,
    date: d.date,
    start: d.start,
    end: d.end,
    rec: d.rec,
    organizer: d.organizer,
    participants: d.participants,
    desc: d.desc,
  }
}

export async function fetchMyMeetings(scope: Scope): Promise<Meeting[]> {
  const res = await api.get<{ meetings: MeetingDTO[] }>("/tma/meetings", { params: { scope } })
  return res.data.meetings.map(toMeeting)
}

export async function fetchColleagueSchedule(email: string, scope: Scope): Promise<Meeting[]> {
  const res = await api.get<{ meetings: MeetingDTO[] }>("/tma/schedule", { params: { email, scope } })
  return res.data.meetings.map(toMeeting)
}

export async function searchEmployees(q: string): Promise<Employee[]> {
  const res = await api.get<{ employees: EmployeeDTO[] }>("/tma/employees", { params: { q } })
  return res.data.employees.map((e) => ({ id: e.id, name: e.name, email: e.email, dept: e.dept, tg: e.tg }))
}

export type FreeSlotsParams = {
  participants: string[]
  from: string // YYYY-MM-DD
  to: string // YYYY-MM-DD
  durationMins: number
}

export async function fetchFreeSlots(params: FreeSlotsParams, lang: Lang): Promise<FreeSlot[]> {
  const res = await api.post<{ slots: FreeSlotDTO[] }>("/tma/free-slots", {
    participants: params.participants,
    from: params.from,
    to: params.to,
    duration_mins: params.durationMins,
  })
  return res.data.slots.map((s) => ({
    day: fmtDate(s.iso, lang),
    iso: s.iso,
    start: s.start,
    end: s.end,
    mins: s.mins,
  }))
}
```

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: typecheck passes; prettier writes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/api.ts
git commit -m "feat(tma): typed read fetchers (meetings, employees, schedule, free-slots)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: React Query hooks

**Files:**
- Create: `frontend/src/shared/tma/queries.ts`

Typecheck-verified.

- [ ] **Step 1: Create the hooks**

Create `frontend/src/shared/tma/queries.ts`:

```ts
import { useMutation, useQuery } from "@tanstack/react-query"
import { useTmaApp } from "./context"
import {
  fetchColleagueSchedule,
  fetchFreeSlots,
  fetchMyMeetings,
  searchEmployees,
  type FreeSlotsParams,
  type Scope,
} from "./api"

export function useMyMeetings(scope: Scope) {
  return useQuery({ queryKey: ["tma", "meetings", scope], queryFn: () => fetchMyMeetings(scope) })
}

export function useEmployeeSearch(q: string) {
  return useQuery({
    queryKey: ["tma", "employees", q],
    queryFn: () => searchEmployees(q),
    enabled: q.trim().length > 0,
  })
}

export function useColleagueSchedule(email: string, scope: Scope) {
  return useQuery({
    queryKey: ["tma", "schedule", email, scope],
    queryFn: () => fetchColleagueSchedule(email, scope),
    enabled: email.trim().length > 0,
  })
}

export function useFreeSlots() {
  const { lang } = useTmaApp()
  return useMutation({ mutationFn: (params: FreeSlotsParams) => fetchFreeSlots(params, lang) })
}
```

- [ ] **Step 2: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: typecheck passes; prettier writes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/queries.ts
git commit -m "feat(tma): React Query hooks for read paths

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: 401 → re-login axios interceptor

**Files:**
- Modify: `frontend/src/shared/tma/auth.ts`
- Modify: `frontend/src/shared/tma/auth-context.tsx`

Typecheck-verified. Installed from the auth provider (keeps the `shared/tma → shared/api` dependency direction; avoids `shared/api` importing `shared/tma`).

- [ ] **Step 1: Add the interceptor installer to `auth.ts`**

Append to `frontend/src/shared/tma/auth.ts`:

```ts
import { isAxiosError } from "axios"

let interceptorInstalled = false

// installTmaAuthInterceptor wires a one-shot re-login on 401 from /api/tma/* calls:
// it re-exchanges initData for a fresh TMA JWT once and replays the request.
export function installTmaAuthInterceptor(): void {
  if (interceptorInstalled) return
  interceptorInstalled = true
  api.interceptors.response.use(undefined, async (error: unknown) => {
    if (!isAxiosError(error) || !error.config) return Promise.reject(error)
    const cfg = error.config as typeof error.config & { __tmaRetried?: boolean }
    const url = cfg.url ?? ""
    if (error.response?.status === 401 && url.startsWith("/tma/") && !cfg.__tmaRetried) {
      cfg.__tmaRetried = true
      try {
        await tmaLogin(getInitData())
        return api(cfg)
      } catch {
        // fall through to the original rejection
      }
    }
    return Promise.reject(error)
  })
}
```

> `tmaLogin`, `getInitData`, and `api` are already imported/defined in this file (from sub-project 1). Add only the `isAxiosError` import if not present.

- [ ] **Step 2: Install it from the provider**

In `frontend/src/shared/tma/auth-context.tsx`, import and call the installer once. Add to the import from `./auth`:

```tsx
import { getInitData, installTmaAuthInterceptor, tmaLogin, type TmaUser } from "./auth"
```

and call it at the very start of the `run()` function (before `getInitData()`), so it's installed before any `/tma/*` request:

```tsx
  function run() {
    installTmaAuthInterceptor()
    setStatus("loading")
    // ... existing body unchanged
```

- [ ] **Step 3: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: typecheck passes; prettier writes.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/shared/tma/auth.ts frontend/src/shared/tma/auth-context.tsx
git commit -m "feat(tma): 401 -> re-login axios interceptor for /api/tma/*

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Wire screens to live data

**Files:**
- Modify: `frontend/src/features/tma/tma-app.tsx`
- Modify: `frontend/src/features/tma/screens/checker-screen.tsx`
- Modify: `frontend/src/features/tma/screens/profile-screen.tsx`

Typecheck + build verified. Home and meetings screens stay prop-driven (no edits needed there) — `tma-app` feeds them live data. Checker and colleague-schedule self-fetch via hooks.

- [ ] **Step 1: `tma-app.tsx` — feed the meetings list from the backend**

In the `TmaContent` component (~line 140):
1. Add imports:
   ```tsx
   import { useQueryClient } from "@tanstack/react-query"
   import { useMyMeetings } from "@/shared/tma/queries"
   ```
2. Replace `const [meetings, setMeetings] = useState<Meeting[]>(INITIAL_MEETINGS)` with:
   ```tsx
   const { data: meetings = [] } = useMyMeetings("all")
   const queryClient = useQueryClient()
   ```
   (Remove the now-unused `INITIAL_MEETINGS` import if `INITIAL_SCENARIOS` is still imported from the same module, keep `INITIAL_SCENARIOS`.)
3. The create handler (`completeCreate`, which did `setMeetings((arr) => [...arr, nm])`) — replace the `setMeetings(...)` call with an optimistic cache prepend (interim; real POST is sub-project 3):
   ```tsx
   queryClient.setQueryData<Meeting[]>(["tma", "meetings", "all"], (old = []) => [nm, ...old])
   ```
4. The delete handler (`deleteMeeting`, which did `setMeetings((arr) => arr.filter(...))`) — replace the `setMeetings(...)` call with:
   ```tsx
   queryClient.setQueryData<Meeting[]>(["tma", "meetings", "all"], (old = []) => old.filter((m) => m.id !== id))
   ```
   Keep the surrounding toast/sheet-close logic unchanged.

> The displayed list is now server-backed; create/delete update the cache optimistically only (a refetch/reopen reverts them) until sub-project 3 adds real mutations. `useMyMeetings("all")` returns every meeting in the ±365-day window; `HomeScreen`/`MeetingsScreen` keep their existing client-side today/upcoming/past/all filtering unchanged.

- [ ] **Step 2: `checker-screen.tsx` — live employee search + free-slots**

Read the file first. Replace the mock-data usage:
- Remove imports of `EMPLOYEES` and `FREE_SLOTS` from `@/shared/tma/mock-data`.
- Add `import { useEmployeeSearch, useFreeSlots } from "@/shared/tma/queries"`.
- For the participant picker's search box: drive matches from `useEmployeeSearch(query).data ?? []` instead of filtering the `EMPLOYEES` mock (keep the existing selected-participants UI/state).
- For the "find slots" action: call the `useFreeSlots()` mutation with `{ participants: selectedEmails, from, to, durationMins }` (map the screen's existing date-range + duration inputs; if the screen currently uses a fixed range, pass the equivalent `from`/`to` it already computes). Render `mutation.data ?? []` (a `FreeSlot[]`) where `FREE_SLOTS` was rendered; show `mutation.isPending` as the loading state and `mutation.isError` as an error message.

> Preserve the screen's existing layout/flow and the `FreeSlot`/`Employee` shapes — only the data source changes. If the screen passed `EMPLOYEES`/`FREE_SLOTS` in via props from `tma-app`, switch those props to the hook results (and drop the now-dead props).

- [ ] **Step 3: `profile-screen.tsx` — live colleague schedule**

Read the colleague-schedule sub-view. Replace its mock data:
- Use `useEmployeeSearch(query)` for the colleague picker (instead of filtering the `EMPLOYEES` mock).
- Once a colleague is picked, use `useColleagueSchedule(picked.email, "all")` for their meetings (instead of filtering the mock `meetings` by email). Render its `data ?? []`; show loading/empty states.
- Add `import { useColleagueSchedule, useEmployeeSearch } from "@/shared/tma/queries"`; remove the now-unused `EMPLOYEES`/meetings-mock references in this sub-view (keep any still used by other parts of the profile screen).

- [ ] **Step 4: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: all pass.

> Troubleshooting: if removing a mock import leaves it referenced elsewhere in the same file (e.g. another sub-view still uses `EMPLOYEES`), keep the import and only swap the read sites listed above. If a screen received `meetings`/`employees`/`slots` as props that no longer have a source after `tma-app` changes, update the prop wiring so the component reads from its hook instead. Resolve any unused-variable TS error minimally.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/features/tma/tma-app.tsx frontend/src/features/tma/screens/checker-screen.tsx frontend/src/features/tma/screens/profile-screen.tsx
git commit -m "feat(tma): wire meetings list, checker, colleague schedule to backend

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Docs + final verification

**Files:**
- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

In the Backend (planned) block, after the last `> **...(done):**` line, add:

```markdown
> **Mini App read paths (frontend integration #2, done):** four TMA-auth read endpoints — `GET /api/tma/meetings?scope=upcoming|past|all` (the authed user's meetings, global-by-email via `EmployeeSchedule`), `GET /api/tma/schedule?email=&scope=` (a colleague's, §4.6), `GET /api/tma/employees?q=` (`SearchEmployeesGlobal`), `POST /api/tma/free-slots` (§4.8) — return UI-shaped DTOs (organizer/participants resolved to emails, times split into date/start/end in Almaty by the pure `splitMeetingTime`; scope→window by the pure `tmaScopeWindow`). Frontend: typed fetchers (`shared/tma/api.ts`) + React Query hooks (`shared/tma/queries.ts`) wire the meetings list (home/meetings tab), checker (employee search + free-slots), and the profile colleague-schedule; a 401→re-login interceptor refreshes the TMA JWT. Writes (create/edit/delete) remain client-side (optimistic cache) until sub-project 3.
```

- [ ] **Step 2: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/...` and re-run. If the frontend build flags prettier, run `pnpm -C frontend format` and re-run.

- [ ] **Step 3: Commit**

```bash
git add docs/MEETINGS.md
git commit -m "docs(tma): document Mini App read paths (frontend integration #2)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** 4 endpoints → Tasks 2,3; UI-shaped DTO + organizer/participant resolution + Almaty split → Task 1 (`toMeetingDTO`, `splitMeetingTime`); scope enum windows → Task 1 (`tmaScopeWindow`); fetchers/hooks → Tasks 4,5; 401 interceptor → Task 6; screen wiring (meetings list, checker, colleague schedule; home stays prop-fed) → Tasks 1-step1 fed `tma-app`, 7; writes-stay-client-side (optimistic cache) → Task 7; docs → Task 8. Detail endpoint intentionally omitted (detail renders from the list item, per spec Decision 3).
- **Type consistency:** backend `tmaMeetingDTO`/`tmaEmployeeDTO`/`tmaFreeSlotDTO` (Task 1) produced by handlers (Tasks 2,3); `toMeetingDTO`/`toMeetingDTOs`/`botUserEmail`/`tmaScopeWindow`/`splitMeetingTime` defined Task 1, used Tasks 2,3. Frontend `MeetingDTO`/`EmployeeDTO`/`FreeSlotDTO` (Task 4) map to `Meeting`/`Employee`/`FreeSlot`; `Scope`/`FreeSlotsParams` (Task 4) consumed by hooks (Task 5); query key `["tma","meetings","all"]` used in Task 5 hook and Task 7 optimistic updates (must match exactly).
- **Known approximations:** N+1 `GetUserByID`/`ListParticipants` per meeting in `toMeetingDTO` (fine at personal scale); the frontend fetches `scope=all` for the list and filters client-side (existing behavior); create/delete are optimistic-cache-only until sub-project 3; the free-slots handler duplicates the workspace-scoped `meeting_availability.go` logic under TMA auth (acceptable — different auth scope).
```
