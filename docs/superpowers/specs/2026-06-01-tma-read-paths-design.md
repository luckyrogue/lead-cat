# TMA Read Paths — Design (frontend integration, sub-project 2)

**Status:** approved (brainstorm), ready for implementation plan.
**Part of:** "wire the meetings Mini App to the backend" — (1) TMA auth & identity [done] → **(2) API client + read paths [this spec]** → (3) write paths → (4) auto/profile tabs.
**Spec source (ТЗ):** `docs/NEW-FEATURES.md` §4.6 (schedule view), §4.8 (free-time checker). Feature status: `docs/MEETINGS.md`. Prior slice: `docs/superpowers/specs/2026-05-31-tma-auth-identity-design.md`.

## Goal

Wire the Mini App's read-only screens to the backend through the TMA auth from sub-project 1: the authenticated Telegram user sees **their own real meetings** (home + meetings tab), can **search the employee directory** and **find common free time** (checker), and can **view a colleague's schedule** (profile). All data is global-by-email, reusing the same application methods the bot's `/schedule` & `/checker` already use. **No writes** (create/edit/delete + conflict warnings stay sub-project 3).

## Decisions (locked during brainstorming)

1. **Backend TMA DTO (UI-shaped).** New `/api/tma/*` endpoints return JSON already in the frontend's shape (date/start/end strings, organizer **email**, participant **emails**). Identity resolution (`organizer_user_id` UUID → `platform_users.email`) and Almaty time-splitting must happen server-side anyway, so one mapping site lives on the backend; the frontend maps the DTO to its existing `Meeting`/`Employee`/`FreeSlot` types with near-zero work.
2. **Windowing = a `scope` enum** (`upcoming` / `past` / `all`), computed server-side in Asia/Almaty (reusing the bot's window approach). The frontend never computes "now"/timezone boundaries. Home derives "today" by filtering the `upcoming` list client-side.
3. **Detail renders from the list item (YAGNI refinement — see below).** The list DTO is complete (includes participants + organizer), and the Mini App only opens a detail sheet from a list tap, so a per-id detail endpoint and its membership-authorization are **not built** in this slice. *(This trims the `GET /api/tma/meetings/:id` + `GetMeetingByID` + membership check from the brainstormed design — flagged for review. It can be added later for deep-linking.)*
4. **Add the deferred 401 → re-login interceptor** now that authed read-calls exist (carried over from sub-project 1).
5. **Writes stay client-side.** Create/delete buttons keep their existing local-state handlers; a created meeting won't survive a React Query refetch until sub-project 3. Acceptable for a read-only slice.

## Codebase facts (verified)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`.
- **TMA auth (sub-project 1):** `middleware.TMAAuth.Middleware` guards a `tma := app.Group("/api/tma", tmaAuth.Middleware)` group; it sets `c.Locals("bot_user").(postgres.BotUser)` (fields `TelegramID, FullName, Email, Role`). New read routes are added to this existing group.
- **Application read methods (global-by-email):**
  - `(*Services).EmployeeSchedule(ctx, email string, from, to time.Time) ([]postgres.Meeting, error)` (`application/participants.go:63`) → `Store.ListScheduleForEmail` (`meeting_repo.go:241`: participant OR organizer by email, `status='scheduled'`, `starts_at` in `[from,to)`, ordered). **Participants are NOT hydrated** by this query.
  - `(*Services).SearchEmployeesGlobal(ctx, query string) ([]postgres.Employee, error)` (ILIKE substring on name/email, cap 20).
  - `(*Services).FreeSlots(ctx, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)` — Mon–Fri 09:00–18:00 Almaty, gaps ≥ durMins. `FreeSlot{Day time.Time (start-of-day Almaty), Start, End time.Time (UTC), Mins int}`.
  - `(*Services).ListParticipants(ctx, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)` — `MeetingParticipant{EmployeeID *uuid.UUID, Email string}`.
  - Organizer identity: `Store.GetUserByID(ctx, *m.OrganizerUserID) (User, error)` → `User.Email` (the pattern `application/conflict.go` already uses).
- **`postgres.Meeting`** (`models.go`): `ID, WorkspaceID uuid.UUID, OrganizerUserID *uuid.UUID, Dept, Type, Host string, StartsAt, EndsAt time.Time, Recurrence, Name, Description, GoogleEventID, MeetLink, Status string, SeriesID *uuid.UUID, RecurrenceUntil *time.Time, Participants []MeetingParticipant`.
- **Handlers** (`delivery/http/handlers`): receiver `*API`; `a.App` is `*application.Services`; errors via `fiber.NewError`; the package already imports `platformauth`, `telegram`, `postgres`, `zap`. The existing `meeting_availability.go` is the closest sibling (a read-ish handler with an `almatyLoc()` helper — reuse/match it). The `tmaUser` DTO + `tma_auth.go`/`tma_me.go` are the sub-project-1 siblings.
- **Frontend:** `@tanstack/react-query` is wired (`QueryClientProvider` at `frontend/src/app/providers.tsx`). Axios `api` (baseURL `/api`) + `setAuthToken` in `shared/api/client.ts`; the TMA bearer is set by `shared/tma/auth-context.tsx` (`tmaLogin`). Types in `shared/tma/types.ts`: `Meeting{id,type,dept,host,date,start,end,rec,recDays?,organizer,participants:string[],desc?}`, `Employee{id,name,email,dept,tg,role?}`, `FreeSlot{day,iso,start,end,mins}`. `meeting-utils.ts` has `fmtDate(iso,lang)` and `buildTitle`. Read screens: `screens/meetings-screen.tsx` (list + `MeetingDetail`), `home-screen.tsx`, `checker-screen.tsx`, colleague-schedule part of `profile-screen.tsx`. `tma-app.tsx` currently seeds `meetings` from `INITIAL_MEETINGS` via `useState` and threads it + create/delete handlers down.
- Conventions: backend pure logic unit-tested, handlers/wiring build-verified; Go run as `env -u GOROOT go ...` from `backend/`; `make lint` (gofmt). Frontend: no test runner — `pnpm -C frontend typecheck` + `pnpm -C frontend build`, `pnpm -C frontend format`. No secrets/initData/JWT logged. Don't touch `frontend/vite.config.ts`.

## Architecture

Thin TMA read handlers over the existing global-by-email application methods; a backend mapper produces UI-shaped DTOs; the frontend consumes via React Query hooks.

### Backend — new endpoints (all under the `/api/tma` group, TMA-auth)

| Method & path | Reuses | Returns |
| --- | --- | --- |
| `GET /api/tma/meetings?scope=upcoming\|past\|all` | `EmployeeSchedule(botUser.Email, from, to)` | `{meetings: tmaMeetingDTO[]}` |
| `GET /api/tma/employees?q=<query>` | `SearchEmployeesGlobal(q)` | `{employees: tmaEmployeeDTO[]}` |
| `GET /api/tma/schedule?email=<email>&scope=…` | `EmployeeSchedule(email, from, to)` | `{meetings: tmaMeetingDTO[]}` |
| `POST /api/tma/free-slots` `{participants:[email…], from, to, duration_mins}` | `FreeSlots(...)` | `{slots: tmaFreeSlotDTO[]}` |

All read the authed `bot_user` from `c.Locals("bot_user")`. The first endpoint scopes to `botUser.Email`; `/schedule` takes an explicit `email` (the §4.6 directory feature lets any registered user view any colleague's schedule, read-only — no per-meeting auth needed). `q` empty → empty list. Invalid `scope` → 400.

**DTOs** (new, in a `handlers/tma_read.go`):
```go
type tmaMeetingDTO struct {
    ID           string   `json:"id"`
    Type         string   `json:"type"`
    Dept         string   `json:"dept"`
    Host         string   `json:"host"`
    Date         string   `json:"date"`   // YYYY-MM-DD (Almaty)
    Start        string   `json:"start"`  // HH:MM (Almaty)
    End          string   `json:"end"`    // HH:MM (Almaty)
    Rec          string   `json:"rec"`    // recurrence
    Organizer    string   `json:"organizer"`    // email
    Participants []string `json:"participants"` // emails
    Desc         string   `json:"desc"`
    MeetLink     string   `json:"meet_link"`
    Status       string   `json:"status"`
}
type tmaEmployeeDTO struct { ID, Name, Email, Dept string; Tg bool }  // json: id,name,email,dept,tg
type tmaFreeSlotDTO struct { ISO, Start, End string; Mins int }       // json: iso,start,end,mins (Almaty)
```

**Mapper + helpers:**
- `splitMeetingTime(startsAt, endsAt time.Time, loc *time.Location) (date, start, end string)` — **pure**, unit-tested. `date=2006-01-02`, `start/end=15:04`, all in `loc`.
- `tmaScopeWindow(scope string, now time.Time) (from, to time.Time, ok bool)` — **pure**, unit-tested. `upcoming → [now, now+365d]`; `past → [now-365d, now]`; `all → [now-365d, now+365d]`; unknown → `ok=false`. (`ListScheduleForEmail` filters `starts_at` in `[from,to)`.)
- `toMeetingDTO(ctx, a *API, m postgres.Meeting) tmaMeetingDTO` — resolves organizer email (`GetUserByID` when `OrganizerUserID != nil`, best-effort: empty on error), `ListParticipants` → emails, `splitMeetingTime`. N+1 `GetUserByID`/`ListParticipants` per meeting — acceptable for personal-scale lists (tens of rows); a note documents it. (`recDays` is omitted — not modeled server-side; it's a create-only concern.)
- Free-slot mapping: `iso = sl.Day` formatted `2006-01-02` (Almaty), `start/end = sl.Start/sl.End.In(Almaty)` `15:04`, `mins = sl.Mins`. The human day **label** (e.g. "Пн, 01.06") is computed on the frontend (i18n) via `fmtDate`.

Almaty location: reuse the `almatyLoc()` helper pattern already in `handlers/meeting_availability.go`.

### Frontend

- **`shared/tma/api.ts`** (new) — typed fetchers over the axios `api` (bearer already set): `fetchMyMeetings(scope)`, `searchEmployees(q)`, `fetchColleagueSchedule(email, scope)`, `fetchFreeSlots({participants,from,to,durationMins})`. Each maps the DTO → the existing `Meeting`/`Employee`/`FreeSlot` types (the only real transform is `FreeSlot.day = fmtDate(iso)`; meetings map 1:1, dropping `meet_link`/`status` unless a screen needs them — add `meetLink?`/`status?` to the `Meeting` type only if the detail sheet uses them).
- **`shared/tma/queries.ts`** (new) — React Query hooks: `useMyMeetings(scope)`, `useEmployeeSearch(q)` (enabled when `q` non-empty, debounced by the screen), `useColleagueSchedule(email, scope)`, `useFreeSlots(params)` (a `useMutation`/lazily-enabled query triggered by the checker's "find" action). Stable query keys (`["tma","meetings",scope]`, etc.).
- **401 interceptor** — install once (in `shared/api/client.ts` or the auth provider): an axios response interceptor that, on a `401` from an `/api/tma/*` call (not `/api/auth/tma`), runs `tmaLogin(getInitData())` once (guarded by a retry flag to avoid loops) and replays the original request; on repeated failure, surfaces the error.
- **Screen wiring** (replace mock reads):
  - `meetings-screen.tsx`: list from `useMyMeetings(filter→scope)`; the upcoming/past/all toggle maps to the scope; `MeetingDetail` renders from the tapped list item (no extra fetch).
  - `home-screen.tsx`: `useMyMeetings("upcoming")`; "today" filtered client-side.
  - `checker-screen.tsx`: `useEmployeeSearch` for the picker; `useFreeSlots` for results.
  - `profile-screen.tsx` colleague schedule: `useEmployeeSearch` + `useColleagueSchedule`.
  - `tma-app.tsx`: drop the `INITIAL_MEETINGS` `useState` seeding; screens self-fetch. The create/delete handlers stay (local-only, sub-project 3); the detail sheet still opens from a tapped item.

## Data flow & error handling

```
screen → useMyMeetings(scope) → GET /api/tma/meetings?scope=
   → TMA mw resolves bot_user → EmployeeSchedule(email, scopeWindow) → []Meeting
   → per meeting: toMeetingDTO (organizer email, participants, Almaty split) → {meetings:[…]}
React Query: isLoading → skeleton; isError → error+retry; data → render
401 on any /api/tma/* → interceptor re-login once → replay (else error state)
```

| Case | Backend | Frontend |
| --- | --- | --- |
| OK | 200 `{meetings\|employees\|slots: […]}` | render |
| Invalid `scope` / bad body | 400 | error state |
| Expired TMA JWT | 401 | interceptor re-login once → replay |
| DB error | 500 | error + retry |
| Empty result | 200 `[]` | empty-state copy |

No PII beyond the response payloads; read handlers log nothing (or `Debug` only).

## Testing

- **Backend unit (pure):** `splitMeetingTime` (Almaty split incl. a meeting crossing local midnight); `tmaScopeWindow` (each scope's bounds + unknown→`ok=false`).
- **Backend build-verified:** the four handlers + DTO mapping + route wiring (no HTTP harness, per convention).
- **Frontend:** `pnpm -C frontend typecheck` + `pnpm -C frontend build`. The fetcher DTO→type mapping and hooks compile; screens render the new states.
- Gate before merge: `make test && make lint && make build`.

## Out of scope (YAGNI / later)

- All writes: create/edit/delete, conflict warnings, reminder settings (sub-project 3); the `auto` scenarios tab + profile settings writes (sub-project 4).
- `GET /api/tma/meetings/:id` detail endpoint + `GetMeetingByID` + membership auth (detail renders from the list item; add when deep-linking is needed).
- Pagination / infinite scroll (personal schedules are small; the 365-day window is enough).
- Caching/offline beyond React Query defaults; optimistic updates (no writes yet).
- `recDays` reconstruction from a series (create-only concern).
