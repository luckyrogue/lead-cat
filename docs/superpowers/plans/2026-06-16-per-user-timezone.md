# Per-User Timezone Implementation Plan (Phase 4a)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Each user picks their own timezone; meeting times render and are interpreted in that timezone instead of hardcoded `Asia/Almaty` (mini-app) / the browser zone (admin).

**Architecture:** Add `timezone` (+ `language`, reserved for Phase 4b) columns to both identity tables (`bot_users`, `platform_users`). Settings endpoints read/write them. The mini-app DTO is server-formatted, so the mini-app handlers resolve the caller's timezone (user → default) and format/parse in it. The admin receives raw RFC3339 timestamps, so the admin formats client-side in the user's chosen zone (surfaced via `/me`); web create/update parse the web user's zone (falling back to the org zone). A small `resolveLoc(tz, fallback)` helper centralizes timezone loading.

**Tech Stack:** Go (goose migrations, pgx, Fiber), OpenAPI → `@leadcat/api-client`, React Router admin + mini-app, TanStack Query.

**Scope:** Timezone only. The migration also adds a `language` column (so Phase 4b — i18n — needs no second migration), and the settings endpoints accept/return `language`, but nothing consumes `language` yet. No i18n in this phase.

**Prerequisite:** Branch `feat/mini-app-meeting-parity`, on top of Phase 3 (HEAD `483f4e4`). Run Go from `apps/backend` with `env -u GOROOT go ...`. A local Postgres is needed to run migrations (`make migrate`); if unavailable, the migration is still committed and verified by `go build` + SQL review. Stage explicit paths only; never `git add -A`; never `.gitignore`. Commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Frontend: no semicolons / double quotes / 2-space; format touched files with each app's `config/prettier.config.mjs` (never app-wide `pnpm format`, never config-less `npx prettier`). IGNORE IDE diagnostics (stale LSP) — trust `go`/`pnpm` output.

---

## File Structure

**Backend**
- Create migration `apps/backend/migrations/20260616120000_user_prefs.sql`.
- Modify `internal/application/model/model.go` — add `Timezone`/`Language` to `BotUser` + `PlatformUser`.
- Modify `internal/infrastructure/persistence/postgres/user_repo.go` (bot users) + the platform-user repo — SELECT the new columns; add setters.
- Modify `internal/application/repository.go` (+ stub) — setter signatures.
- Modify `internal/application/user_settings.go` — `UserSettings` + getters/setters for tz+language (mini-app + web).
- Create `internal/delivery/http/handlers/time_helpers.go` addition — `resolveLoc`.
- Modify `internal/delivery/http/handlers/miniapp_read.go`, `miniapp_write.go`, `miniapp_settings.go` — use the caller's timezone; settings accept tz+language.
- Modify `internal/delivery/http/handlers/web_auth.go` — `/me` exposes `timezone`/`language`; add `/me/settings` GET+PATCH.
- Modify `internal/application/command/meetings.go` — accept an optional input timezone (fallback org tz).
- Modify `internal/delivery/http/app.go` — `/me/settings` routes.
- Modify `apps/backend/openapi/openapi.json` + regen client.

**Frontend**
- Mini-app: `entities/settings/{api,queries}.ts`, `features/profile/components/*` (timezone picker; language picker stub).
- Admin: `shared/auth/{types,use-me}.ts`, `features/meetings/lib/format.ts` (tz param), a settings page/section + timezone picker, `entities`/api for web `/me/settings`.

---

## Task 1: Migration + model + repo columns

**Files:** new migration; `model/model.go`; `internal/infrastructure/persistence/postgres/user_repo.go` (and the platform-user repo file — find it); `internal/application/repository.go` (+ `repository_unimpl_test.go`).

- [ ] **Step 1: Create the migration** `apps/backend/migrations/20260616120000_user_prefs.sql` (match the goose `-- +goose Up/Down` format used by the latest migration — READ a recent one first, e.g. `20260610170000_audit_actor_web.sql`):
```sql
-- +goose Up
ALTER TABLE bot_users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE bot_users ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE platform_users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE platform_users ADD COLUMN language TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bot_users DROP COLUMN timezone;
ALTER TABLE bot_users DROP COLUMN language;
ALTER TABLE platform_users DROP COLUMN timezone;
ALTER TABLE platform_users DROP COLUMN language;
```

- [ ] **Step 2: Add model fields.** In `model/model.go`, add to `BotUser` and `PlatformUser`:
```go
	Timezone string
	Language string
```
(For `BotUser`, give them json tags consistent with the struct's existing tags: `json:"timezone"`, `json:"language"`. `PlatformUser` has no json tags — match that.)

- [ ] **Step 3: SELECT the new columns + add setters.** READ `internal/infrastructure/persistence/postgres/user_repo.go` (and whatever file holds platform-user reads — grep `GetPlatformUserByID`, `GetBotUserByTelegramID`). For each of `GetBotUserByTelegramID` and `GetPlatformUserByID`, add `timezone, language` to the SELECT column list and scan them into the new fields. Add setters:
```go
func (s *Store) SetBotUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bot_users SET timezone = $2, language = $3 WHERE telegram_id = $1`, telegramID, timezone, language)
	return err
}

func (s *Store) SetPlatformUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error {
	_, err := s.pool.Exec(ctx, `UPDATE platform_users SET timezone = $2, language = $3 WHERE id = $1`, userID, timezone, language)
	return err
}
```
(Confirm the pool field name `s.pool` and the existing scan style; match them. The bot-user and platform-user repos may be different files/structs — place each setter next to its sibling reads.)

- [ ] **Step 4: Add to the Repository port + stub.** In `internal/application/repository.go` add:
```go
	SetBotUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error
	SetPlatformUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error
```
In `internal/application/repository_unimpl_test.go` add matching stubs (match the file's stub style).

- [ ] **Step 5: Build + run the migration if a DB is available.**
`cd apps/backend && env -u GOROOT go build ./...` (must pass).
If Postgres is up: `make migrate` then `make migrate-status` (the new migration applied). If no DB, note it — the SQL is reviewed manually.

- [ ] **Step 6: Commit**
```bash
git add apps/backend/migrations/20260616120000_user_prefs.sql \
  apps/backend/internal/application/model/model.go \
  apps/backend/internal/infrastructure/persistence/postgres/ \
  apps/backend/internal/application/repository.go \
  apps/backend/internal/application/repository_unimpl_test.go
git commit -m "feat(users): timezone + language columns on bot_users + platform_users

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```
(Stage only the specific postgres files you edited, not the whole dir, if you prefer — list them explicitly.)

---

## Task 2: Settings application + endpoints (mini-app + web)

**Files:** `internal/application/user_settings.go`; `internal/delivery/http/handlers/miniapp_settings.go`; `internal/delivery/http/handlers/web_auth.go`; `internal/delivery/http/app.go`.

- [ ] **Step 1: Extend `UserSettings` + application methods.** READ `internal/application/user_settings.go`. Add `Timezone string` + `Language string` to the `UserSettings` struct; populate them in `GetUserSettings` from the bot user. Add:
```go
func (s *Services) SetUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return s.Store.SetBotUserPrefs(ctx, telegramID, timezone, language)
}

type WebUserSettings struct {
	Timezone string `json:"timezone"`
	Language string `json:"language"`
}

func (s *Services) GetWebUserSettings(ctx context.Context, userID uuid.UUID) (WebUserSettings, error) {
	u, err := s.Store.GetPlatformUserByID(ctx, userID)
	if err != nil {
		return WebUserSettings{}, err
	}
	return WebUserSettings{Timezone: u.Timezone, Language: u.Language}, nil
}

func (s *Services) SetWebUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return s.Store.SetPlatformUserPrefs(ctx, userID, timezone, language)
}
```
Add a pure validator (empty = "unset" is allowed for both):
```go
var supportedLanguages = map[string]bool{"": true, "ru": true, "en": true, "kk": true}

func validatePrefs(timezone, language string) error {
	if !supportedLanguages[language] {
		return fmt.Errorf("%w: language", ErrInvalidInput)
	}
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err != nil {
			return fmt.Errorf("%w: timezone", ErrInvalidInput)
		}
	}
	return nil
}
```
(Confirm `ErrInvalidInput`, `time`, `fmt`, `uuid` are imported/available; add if needed.)

- [ ] **Step 2: Mini-app settings handler.** In `miniapp_settings.go`, extend `MiniAppPatchSettings` so the body also accepts optional `timezone`/`language`, and apply them. Keep `reminder_minutes` working. Simplest: parse a body with all three optional pointers; if `reminder_minutes` present call the existing reminder setter; if `timezone` OR `language` present call `SetUserPrefs(telegramID, timezone, language)` (read current values for the one not supplied, or require both together — prefer: accept `timezone` and `language` together as a pair, falling back to current settings for a missing one). Concretely:
```go
	var body struct {
		ReminderMinutes *[]int  `json:"reminder_minutes"`
		Timezone        *string `json:"timezone"`
		Language        *string `json:"language"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if body.ReminderMinutes != nil {
		if err := a.App.SetUserReminderMinutes(c.Context(), bu.TelegramID, *body.ReminderMinutes); err != nil {
			...existing mapping...
		}
	}
	if body.Timezone != nil || body.Language != nil {
		cur, err := a.App.GetUserSettings(c.Context(), bu.TelegramID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		tz := cur.Timezone
		if body.Timezone != nil {
			tz = *body.Timezone
		}
		lang := cur.Language
		if body.Language != nil {
			lang = *body.Language
		}
		if err := a.App.SetUserPrefs(c.Context(), bu.TelegramID, tz, lang); err != nil {
			if errors.Is(err, application.ErrInvalidInput) {
				return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
```
The `GET /settings` now returns `{reminder_minutes, timezone, language}` automatically since `UserSettings` carries them.

- [ ] **Step 2b: Web `/me/settings`.** In `web_auth.go`, add a GET returning `{timezone, language}` for the web user, and a PATCH accepting `{timezone?, language?}`:
```go
func (a *API) WebGetMeSettings(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	s, err := a.App.GetWebUserSettings(c.UserContext(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(s)
}

func (a *API) WebPatchMeSettings(c *fiber.Ctx) error {
	user := c.Locals("web_user").(model.PlatformUser)
	cur, err := a.App.GetWebUserSettings(c.UserContext(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	var body struct {
		Timezone *string `json:"timezone"`
		Language *string `json:"language"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	tz, lang := cur.Timezone, cur.Language
	if body.Timezone != nil {
		tz = *body.Timezone
	}
	if body.Language != nil {
		lang = *body.Language
	}
	if err := a.App.SetWebUserPrefs(c.UserContext(), user.ID, tz, lang); err != nil {
		if errors.Is(err, application.ErrInvalidInput) {
			return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```
Also extend the `WebMe` response `user` map with `"timezone": user.Timezone, "language": user.Language` (so the admin can read it from `/me`).

- [ ] **Step 2c: Routes.** In `app.go`, under the `web := app.Group("/api/auth/web", ...)` group (same place `WebMe` is registered, behind web auth middleware), add:
```go
	web.Get("/me/settings", api.WebGetMeSettings)
	web.Patch("/me/settings", api.WebPatchMeSettings)
```
(Match the actual group var + middleware used for `WebMe`; READ app.go first. The PATCH must be behind the same CSRF-protected web-auth middleware as other web mutations.)

- [ ] **Step 3: Build + test:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
- [ ] **Step 4: Commit**
```bash
git add apps/backend/internal/application/user_settings.go \
  apps/backend/internal/delivery/http/handlers/miniapp_settings.go \
  apps/backend/internal/delivery/http/handlers/web_auth.go \
  apps/backend/internal/delivery/http/app.go
git commit -m "feat(users): timezone+language settings endpoints (mini-app + web /me/settings)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Backend — render mini-app times in the user's timezone

**Files:** `internal/delivery/http/handlers/time_helpers.go`; `miniapp_read.go`; `miniapp_write.go`.

- [ ] **Step 1: Add `resolveLoc`** to `time_helpers.go`:
```go
// resolveLoc loads tz, falling back to Asia/Almaty when tz is empty or invalid.
func resolveLoc(tz string) *time.Location {
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return almatyLoc()
}
```
- [ ] **Step 2: Use the caller's timezone in mini-app reads.** In `miniapp_read.go`, the meeting-list/DTO + free-slots handlers currently use `almatyLoc()`. The handlers have the `bu` (bot user) — READ each handler; replace `almatyLoc()` with `resolveLoc(bu.Timezone)` (fetch `bu` via the existing `botUser(c)` / settings helper if not already in scope). For `MiniAppMyMeetings`/`toMeetingDTO` path, thread the resolved loc so `MiniAppMeetingDTO(ctx, m, loc)` uses it. For `MiniAppFreeSlots`, use `resolveLoc(bu.Timezone)` instead of `almatyLoc()`.
- [ ] **Step 3: Use the caller's timezone in mini-app conflict checks.** In `miniapp_write.go` `MiniAppConflicts`, replace `almatyLoc()` with `resolveLoc(bu.Timezone)` for both parsing the requested window and formatting conflict DTOs.
- [ ] **Step 4: Build + test:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
- [ ] **Step 5: Commit**
```bash
git add apps/backend/internal/delivery/http/handlers/time_helpers.go \
  apps/backend/internal/delivery/http/handlers/miniapp_read.go \
  apps/backend/internal/delivery/http/handlers/miniapp_write.go
git commit -m "feat(meetings): render mini-app times in the user's timezone

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Backend — parse create/update input in the user's timezone

**Files:** `internal/application/command/meetings.go`; the create/update application paths (`series_edit.go`/`meeting_service.go` if they parse); the web + mini-app write handlers.

- [ ] **Step 1: Thread an input timezone into create/update.** READ `internal/application/command/meetings.go` (`CreateInput` + `CreateMeeting`, and the update path). Add an optional `Timezone string` to the transport input struct(s) (`CreateInput`, and the update input). In the parse code, change `loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))` to prefer the input timezone when set:
```go
	tz := w.TZ
	if in.Timezone != "" {
		tz = in.Timezone
	}
	loc, err := time.LoadLocation(orDefault(tz, "Asia/Almaty"))
```
Apply the same precedence anywhere create/update/series-end parse `date`+`time` or `until` (search for `ParseInLocation` in the create/update/series paths). For `ChangeSeriesEnd`, the anchor's own occurrences define the cadence; only the `until` date parse should respect the user's tz — thread a tz param through `ChangeSeriesEnd(..., tz string)` OR resolve from the org as today (KEEP series reshape using the existing org/loc resolution if threading is invasive — document the choice; the main win is create/update).
- [ ] **Step 2: Set the input timezone from the user in the handlers.** Mini-app create/update (`miniapp_write.go`): set `Timezone` from `resolveLoc`'s source — i.e. pass `bu.Timezone` into the input builder (`toCreateMeetingInput` / update input). Web create/update (`web_meetings.go`): set it from the web user's timezone — fetch via `a.App.GetWebUserSettings(ctx, user.ID)` (or read it off the `web_user` local which now carries `Timezone`) and pass into the input. When the user's tz is empty, the org tz is used (unchanged behavior).
- [ ] **Step 3: Build + test + lint:** `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./... && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/...`
Expected: clean; 0 issues on touched packages (pre-existing issues elsewhere not in scope).
- [ ] **Step 4: Commit**
```bash
git add apps/backend/internal/application/command/meetings.go \
  apps/backend/internal/delivery/http/handlers/miniapp_write.go \
  apps/backend/internal/delivery/http/handlers/web_meetings.go
git commit -m "feat(meetings): parse create/update times in the user's timezone

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```
(Add any application files you edited for the tz threading.)

---

## Task 5: OpenAPI + regenerate client

- [ ] **Step 1:** In `openapi.json`: add `timezone` + `language` (string) to the `MiniAppUserSettings` schema (properties; keep `reminder_minutes`); extend the mini-app `PATCH /api/miniapp/settings` request body to allow optional `timezone`/`language`; add the web `GET`/`PATCH /api/auth/web/me/settings` operations (`WebUserSettings` schema `{timezone, language}`); add `timezone`/`language` to the web `/me` user object schema. Keep compact style; do NOT prettier the file.
- [ ] **Step 2:** Validate + regen: `cd /Users/temirlan/Workspace/in-house/lead-cat && python3 -c "import json; json.load(open('apps/backend/openapi/openapi.json')); print('valid')" && pnpm openapi:generate`. Confirm additive openapi diff.
- [ ] **Step 3: Commit** `apps/backend/openapi/openapi.json` + `packages/api-client/src/generated/schema.ts`:
```bash
git commit -m "feat(users): openapi for user timezone/language settings; regen client

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Mini-app — timezone (+ language) picker in profile

**Files:** `apps/mini-app/app/entities/settings/{api,queries}.ts`; `apps/mini-app/app/features/profile/components/*` (a new `preferences-settings.tsx`); `profile-page.tsx`.

- [ ] **Step 1: Settings api/query.** Extend `entities/settings/api.ts`: `fetchSettings` now returns `{reminder_minutes, timezone, language}` (the type is the generated `MiniAppUserSettings`). Add `updatePrefs(timezone, language)` that PATCHes `/api/miniapp/settings` with `{timezone, language}`. Add a `useUpdatePrefs()` mutation in `queries.ts` invalidating settings.
- [ ] **Step 2: Picker UI.** Create `features/profile/components/preferences-settings.tsx`: a `Card` with a timezone `<select>` (a curated IANA list — provide ~12 common zones incl. `Asia/Almaty`, `Europe/Moscow`, `Asia/Almaty`, `Asia/Dubai`, `Europe/London`, `America/New_York`, `Asia/Tashkent`, `Asia/Bishkek`, `UTC`, etc.; value `""` = "Default (Almaty)") bound to the user's current `timezone`, plus a language `<select>` (`""`=Default, `ru`, `en`, `kk`) — language is stored now (consumed in Phase 4b). On change, call `useUpdatePrefs().mutate(...)` with optimistic local state + error toast (mirror `reminder-settings.tsx`'s pattern). Add `<PreferencesSettings />` to `profile-page.tsx` near `<ReminderSettings />`.
- [ ] **Step 3: Gate + format.** `cd apps/mini-app && pnpm typecheck && pnpm lint && pnpm build`; prettier-write the touched files with the mini-app config; re-typecheck. (Mini-app meeting times update automatically because the backend now formats them in the user's tz.)
- [ ] **Step 4: Commit** the touched mini-app files with:
```
feat(mini-app): timezone + language preference picker
```

---

## Task 7: Admin — render meeting times in the user's timezone + picker

**Files:** `apps/admin/app/shared/auth/{types,use-me}.ts`; `apps/admin/app/features/meetings/lib/format.ts`; meetings components passing the tz; a settings page/section + web `/me/settings` api/query.

- [ ] **Step 1: Surface the user's timezone.** Add `timezone?: string` + `language?: string` to the admin `WebUser` type (`shared/auth/types.ts`); it now comes from `/me`. Confirm `use-me.ts` passes the whole user object through (no change needed if it does).
- [ ] **Step 2: Format in the user's timezone.** Change `features/meetings/lib/format.ts` `formatDateTime`/`formatTimeRange` to accept an optional `timeZone?: string` and pass it to `toLocaleString`/`toLocaleTimeString` (`{ timeZone }` when provided; falls back to browser zone when undefined). Update the callers (`meetings-table.tsx`, the meetings page, any meeting card) to pass the current user's timezone (from `useMe()`), e.g. thread a `timeZone` prop into `MeetingsTable`/rows. Keep it simple: read `me.user.timezone` in the page and pass down.
- [ ] **Step 3: Settings UI + api.** Add a web `/me/settings` api (`entities/me/api.ts` or similar: `getMeSettings()`, `updateMeSettings({timezone, language})`) + query/mutation. Add a settings page or a section (e.g. a new route `/_app.settings` or a card on an existing profile/settings page — match the admin's existing page/route conventions; READ `app/routes.ts` + a sidebar to place it) with a timezone `<Select>` (curated IANA list, `""`=browser default) and a language `<Select>` (`""`/`ru`/`en`/`kk`, stored for 4b). On save, call the mutation; invalidate `/me` so the new timezone re-renders meeting times. Use `@leadcat/ui` Select.
- [ ] **Step 4: Gate + format.** `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`; prettier-write touched files with the admin config; re-typecheck.
- [ ] **Step 5: Commit** the touched admin files with:
```
feat(admin): render meeting times in the user's timezone + preference picker
```

---

## Task 8: Final verification

- [ ] **Step 1:** Backend `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`.
- [ ] **Step 2:** Admin `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`.
- [ ] **Step 3:** Mini-app `cd apps/mini-app && pnpm typecheck && pnpm lint && pnpm build`.
- [ ] **Step 4:** Coverage grep: `resolveLoc`, `SetBotUserPrefs`, `SetPlatformUserPrefs`, `WebGetMeSettings`, `GetWebUserSettings`, the migration file, and the two frontend pickers all exist.
- [ ] **Step 5:** `git status --short` clean.

---

## Notes & decisions

- **`language` is stored but unused this phase** — it's added to the schema/settings/pickers now so Phase 4b (i18n) needs no migration and the UI is ready; nothing reads it yet.
- **Mini-app display** is server-formatted, so per-user timezone is a backend change (resolve `bu.Timezone` → default). Mini-app needs no display-side change beyond the picker.
- **Admin display** is client-formatted from raw timestamps, so per-user timezone is a frontend change (format with `{ timeZone }` from `/me`).
- **Input precedence:** create/update parse in the user's timezone when set, else the org timezone, else `Asia/Almaty` (unchanged fallback). Empty timezone = today's behavior, so existing users are unaffected until they pick one.
- **Series reshape (`ChangeSeriesEnd`)** may keep org/loc resolution if threading the user tz is invasive — the cadence is anchor-derived; only the `until` parse is tz-sensitive and low-stakes. Document whichever you choose.
- **Timezone picker is a curated IANA shortlist**, not the full tz database, to keep the UI simple; "Default" (empty) preserves current behavior.
</content>
