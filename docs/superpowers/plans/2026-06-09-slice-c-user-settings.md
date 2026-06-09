# Slice C — User settings: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the existing per-user `bot_users.reminder_minutes` over `GET/PATCH /api/miniapp/settings` and rewrite the Profile page chips from local state to server-backed via TanStack Query with optimistic updates.

**Architecture:** Backend exists end-to-end already (column, store methods, parse helper, scheduler, bot `/settings`); this slice adds a thin application service with a whitelist guard, two HTTP handlers, OpenAPI entries, and a frontend `entities/user-settings` slice that the Profile page consumes.

**Tech Stack:** Go 1.22 (Fiber, pgx, zap), Postgres 15, React 18, TanStack Router/Query, Vitest, OpenAPI 3.1.

**Spec:** [`docs/superpowers/specs/2026-06-09-slice-c-user-settings-design.md`](../specs/2026-06-09-slice-c-user-settings-design.md).

**Branch:** `feat/meetings-settings-c` (created from `main` at `3ac3095`).

---

## File structure

```
backend/
├── internal/
│   ├── application/
│   │   ├── user_settings.go                                    [NEW]
│   │   └── user_settings_test.go                               [NEW]
│   ├── platform/botsettings/
│   │   └── settings.go                                         [MODIFY: export Format]
│   └── delivery/http/
│       ├── app.go                                              [MODIFY: register routes]
│       └── handlers/
│           └── miniapp_settings.go                             [NEW]
├── openapi/openapi.json                                        [MODIFY: 2 paths + 2 schemas]
└── docs/openapi.json                                           [MODIFY: byte-identical mirror]

frontend/
├── src/
│   ├── shared/api/generated/schema.ts                          [REGEN]
│   ├── shared/tma/i18n.ts                                      [MODIFY: ~8 keys × 3 langs]
│   ├── entities/user-settings/                                 [NEW slice]
│   │   ├── api.ts
│   │   ├── write-api.ts
│   │   ├── mutations.ts
│   │   ├── queries.ts
│   │   ├── types.ts
│   │   └── constants.ts
│   └── features/profile/pages/profile-page.tsx                 [MODIFY]

docs/
├── API.md                                                      [MODIFY]
└── MEETINGS.md                                                 [MODIFY]
```

---

### Task C-T0: Branch already created

- [x] Branch `feat/meetings-settings-c` created from `main` at `3ac3095`. Spec committed as `b4519db`. Nothing to do.

---

### Task C-T1: Export `botsettings.Format`

**Files:**
- Modify: `backend/internal/platform/botsettings/settings.go` (one-line addition next to existing `Parse` export)

- [ ] **Step 1: Add the export**

Open `backend/internal/platform/botsettings/settings.go`. Find the existing public `Parse` function (around line 30). Add this new function next to it (typically directly under `Parse`):

```go
// Format exposes the canonical CSV writer to other packages.
func Format(mins []int) string { return format(mins) }
```

- [ ] **Step 2: Build to verify**

Run: `cd backend && go build ./...`
Expected: clean.

- [ ] **Step 3: Run existing tests still pass**

Run: `cd backend && go test ./internal/platform/botsettings/ -v`
Expected: PASS (no new tests yet; existing must not break).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/platform/botsettings/settings.go
git commit -m "$(cat <<'EOF'
feat(botsettings): export Format for application-layer reuse

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T2: Application service `user_settings.go` (TDD)

**Files:**
- Create: `backend/internal/application/user_settings.go`
- Create: `backend/internal/application/user_settings_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/application/user_settings_test.go`:

```go
package application

import (
	"context"
	"errors"
	"testing"

	"github.com/Jaryq-Lab/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeUserSettingsStore struct {
	bu        postgres.BotUser
	getErr    error
	setCSV    string
	setErr    error
	setCalled bool
}

func (f *fakeUserSettingsStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	if f.getErr != nil {
		return postgres.BotUser{}, f.getErr
	}
	return f.bu, nil
}

func (f *fakeUserSettingsStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.setCSV = csv
	f.setCalled = true
	return nil
}

func TestSetUserReminderMinutes_WhitelistOK(t *testing.T) {
	f := &fakeUserSettingsStore{}
	err := setUserReminderMinutes(context.Background(), f, 42, []int{10, 30})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !f.setCalled || f.setCSV != "10,30" {
		t.Fatalf("expected CSV '10,30', got '%s' (called=%v)", f.setCSV, f.setCalled)
	}
}

func TestSetUserReminderMinutes_RejectsNonWhitelist(t *testing.T) {
	f := &fakeUserSettingsStore{}
	err := setUserReminderMinutes(context.Background(), f, 42, []int{7})
	if !errors.Is(err, ErrInvalidReminderMinute) {
		t.Fatalf("expected ErrInvalidReminderMinute, got %v", err)
	}
	if f.setCalled {
		t.Fatalf("store must not be written on validation failure")
	}
}

func TestSetUserReminderMinutes_DedupeSort(t *testing.T) {
	f := &fakeUserSettingsStore{}
	if err := setUserReminderMinutes(context.Background(), f, 42, []int{30, 10, 30}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if f.setCSV != "10,30" {
		t.Fatalf("expected '10,30', got '%s'", f.setCSV)
	}
}

func TestSetUserReminderMinutes_Empty(t *testing.T) {
	f := &fakeUserSettingsStore{}
	if err := setUserReminderMinutes(context.Background(), f, 42, []int{}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !f.setCalled || f.setCSV != "" {
		t.Fatalf("expected empty CSV, got '%s' (called=%v)", f.setCSV, f.setCalled)
	}
}

func TestGetUserSettings_ParsesCSV(t *testing.T) {
	f := &fakeUserSettingsStore{bu: postgres.BotUser{ReminderMinutes: "15,60"}}
	got, err := getUserSettings(context.Background(), f, 42)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.ReminderMinutes) != 2 || got.ReminderMinutes[0] != 15 || got.ReminderMinutes[1] != 60 {
		t.Fatalf("expected [15,60], got %v", got.ReminderMinutes)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/application/ -run 'TestSetUserReminderMinutes|TestGetUserSettings' -v`
Expected: FAIL — `setUserReminderMinutes`, `getUserSettings`, `ErrInvalidReminderMinute`, `UserSettings` undefined.

- [ ] **Step 3: Write implementation**

Create `backend/internal/application/user_settings.go`:

```go
package application

import (
	"context"
	"errors"
	"slices"

	"github.com/Jaryq-Lab/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/lead-cat/internal/platform/botsettings"
)

// ErrInvalidReminderMinute is returned when a minute value is not in the
// botsettings.Intervals whitelist.
var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

// UserSettings is the per-user settings projection exposed to the Mini App.
type UserSettings struct {
	ReminderMinutes []int `json:"reminder_minutes"`
}

// userSettingsStore is the narrow store interface used by GetUserSettings /
// SetUserReminderMinutes — defined here so unit tests can mock it.
type userSettingsStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
}

// GetUserSettings returns the authed bot user's settings.
func (s *Services) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	return getUserSettings(ctx, s.Store, telegramID)
}

// SetUserReminderMinutes validates input against the whitelist, dedupes/sorts,
// and writes canonical CSV.
func (s *Services) SetUserReminderMinutes(ctx context.Context, telegramID int64, minutes []int) error {
	return setUserReminderMinutes(ctx, s.Store, telegramID, minutes)
}

func getUserSettings(ctx context.Context, store userSettingsStore, telegramID int64) (UserSettings, error) {
	u, err := store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return UserSettings{}, err
	}
	return UserSettings{ReminderMinutes: botsettings.Parse(u.ReminderMinutes)}, nil
}

func setUserReminderMinutes(ctx context.Context, store userSettingsStore, telegramID int64, minutes []int) error {
	allowed := allowedReminderMinutes()
	for _, m := range minutes {
		if !slices.Contains(allowed, m) {
			return ErrInvalidReminderMinute
		}
	}
	cp := append([]int(nil), minutes...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	return store.SetReminderMinutes(ctx, telegramID, botsettings.Format(cp))
}

func allowedReminderMinutes() []int {
	out := make([]int, 0, len(botsettings.Intervals))
	for _, iv := range botsettings.Intervals {
		out = append(out, iv.Minutes)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/application/ -run 'TestSetUserReminderMinutes|TestGetUserSettings' -v`
Expected: PASS — 5 tests.

- [ ] **Step 5: Build full**

Run: `cd backend && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/application/user_settings.go backend/internal/application/user_settings_test.go
git commit -m "$(cat <<'EOF'
feat(application): UserSettings + whitelist validation

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T3: HTTP handlers + route registration

**Files:**
- Create: `backend/internal/delivery/http/handlers/miniapp_settings.go`
- Modify: `backend/internal/delivery/http/app.go` (append 2 lines after `miniapp.Get("/me", ...)`)

- [ ] **Step 1: Create the handler file**

Create `backend/internal/delivery/http/handlers/miniapp_settings.go`:

```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/lead-cat/internal/application"
	"github.com/Jaryq-Lab/lead-cat/internal/infrastructure/persistence/postgres"
)

// miniAppSettingsBotUser extracts the authed bot user identity for settings handlers.
func miniAppSettingsBotUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// MiniAppGetSettings — GET /api/miniapp/settings
func (a *API) MiniAppGetSettings(c *fiber.Ctx) error {
	bu, ok := miniAppSettingsBotUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	s, err := a.App.GetUserSettings(c.Context(), bu.TelegramID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(s)
}

// MiniAppPatchSettings — PATCH /api/miniapp/settings
func (a *API) MiniAppPatchSettings(c *fiber.Ctx) error {
	bu, ok := miniAppSettingsBotUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var body struct {
		ReminderMinutes *[]int `json:"reminder_minutes"`
	}
	if err := c.BodyParser(&body); err != nil || body.ReminderMinutes == nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if err := a.App.SetUserReminderMinutes(c.Context(), bu.TelegramID, *body.ReminderMinutes); err != nil {
		if errors.Is(err, application.ErrInvalidReminderMinute) {
			return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

**Note for implementer:** if `miniAppBotUser` already exists in another handler file (D-T8 introduced one in `tma_admin.go`), do NOT redeclare — rename this file's helper to `miniAppSettingsBotUser` as shown above OR drop the helper and inline the type assertion. Choose whichever the project uses elsewhere for non-admin Mini App handlers.

- [ ] **Step 2: Append route registrations to `app.go`**

Open `backend/internal/delivery/http/app.go`. Find the line `miniapp.Get("/me", api.MiniAppMe)`. Immediately after it, insert:

```go
miniapp.Get("/settings", api.MiniAppGetSettings)
miniapp.Patch("/settings", api.MiniAppPatchSettings)
```

- [ ] **Step 3: Build to verify**

Run: `cd backend && go build ./...`
Expected: clean.

- [ ] **Step 4: Run all tests**

Run: `cd backend && go test ./...`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/handlers/miniapp_settings.go backend/internal/delivery/http/app.go
git commit -m "$(cat <<'EOF'
feat(http): GET/PATCH /api/miniapp/settings

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T4: OpenAPI changes + frontend schema regen

**Files:**
- Modify: `backend/openapi/openapi.json` (add 2 paths + 2 schemas)
- Modify: `backend/docs/openapi.json` (byte-identical mirror via `cp`)
- Regen: `frontend/src/shared/api/generated/schema.ts`

- [ ] **Step 1: Add the 2 paths under `paths` in `backend/openapi/openapi.json`**

Insert this block as a new entry in the `paths` object (location does not matter — JSON objects are unordered):

```json
"/api/miniapp/settings": {
  "get": {
    "operationId": "miniapp_settings_get",
    "tags": ["miniapp"],
    "summary": "Get current user settings",
    "security": [{ "bearerAuth": [] }],
    "responses": {
      "200": {
        "description": "Current settings",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/MiniAppUserSettings" } } }
      },
      "401": {
        "description": "Unauthorized",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApiErrorResponse" } } }
      },
      "500": {
        "description": "Internal error",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApiErrorResponse" } } }
      }
    }
  },
  "patch": {
    "operationId": "miniapp_settings_patch",
    "tags": ["miniapp"],
    "summary": "Update reminder minutes",
    "security": [{ "bearerAuth": [] }],
    "requestBody": {
      "required": true,
      "content": { "application/json": { "schema": { "$ref": "#/components/schemas/MiniAppUserSettingsPatch" } } }
    },
    "responses": {
      "204": { "description": "Saved" },
      "400": {
        "description": "validation_failed",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApiErrorResponse" } } }
      },
      "401": {
        "description": "Unauthorized",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApiErrorResponse" } } }
      },
      "500": {
        "description": "Internal error",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApiErrorResponse" } } }
      }
    }
  }
}
```

- [ ] **Step 2: Add the 2 schemas under `components.schemas`**

```json
"MiniAppUserSettings": {
  "type": "object",
  "required": ["reminder_minutes"],
  "properties": {
    "reminder_minutes": {
      "type": "array",
      "items": { "type": "integer", "format": "int32" }
    }
  }
},
"MiniAppUserSettingsPatch": {
  "type": "object",
  "required": ["reminder_minutes"],
  "properties": {
    "reminder_minutes": {
      "type": "array",
      "items": {
        "type": "integer",
        "format": "int32",
        "enum": [10, 15, 30, 60, 120, 1440]
      }
    }
  }
}
```

- [ ] **Step 3: Validate JSON syntax**

Run: `jq . backend/openapi/openapi.json > /dev/null`
Expected: silent success.

- [ ] **Step 4: Mirror to docs/**

Run: `cp backend/openapi/openapi.json backend/docs/openapi.json`

- [ ] **Step 5: Confirm Go embed still works**

Run: `cd backend && go build ./...`
Expected: clean.

- [ ] **Step 6: Regen frontend schema**

Run: `cd frontend && npx openapi-typescript ../backend/openapi/openapi.json -o src/shared/api/generated/schema.ts`
Expected: file rewritten; new symbols `MiniAppUserSettings`, `MiniAppUserSettingsPatch` present. Verify with `grep -c MiniAppUserSettings frontend/src/shared/api/generated/schema.ts` — should be ≥ 2.

- [ ] **Step 7: Typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add backend/openapi/openapi.json backend/docs/openapi.json frontend/src/shared/api/generated/schema.ts
git commit -m "$(cat <<'EOF'
feat(api): OpenAPI for slice C — settings GET/PATCH + 2 schemas

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T5: Frontend `entities/user-settings` slice

**Files:**
- Create: `frontend/src/entities/user-settings/types.ts`
- Create: `frontend/src/entities/user-settings/constants.ts`
- Create: `frontend/src/entities/user-settings/api.ts`
- Create: `frontend/src/entities/user-settings/write-api.ts`
- Create: `frontend/src/entities/user-settings/queries.ts`
- Create: `frontend/src/entities/user-settings/mutations.ts`

- [ ] **Step 1: types.ts**

Create `frontend/src/entities/user-settings/types.ts`:

```ts
export type UserSettings = { reminderMinutes: number[] }
```

- [ ] **Step 2: constants.ts**

Create `frontend/src/entities/user-settings/constants.ts`:

```ts
// Must stay in sync with backend/internal/platform/botsettings/settings.go Intervals.
export const REMINDER_INTERVALS = [
  { value: 10, labelKey: "rem10m" as const },
  { value: 15, labelKey: "rem15m" as const },
  { value: 30, labelKey: "rem30m" as const },
  { value: 60, labelKey: "rem1h" as const },
  { value: 120, labelKey: "rem2h" as const },
  { value: 1440, labelKey: "rem1d" as const },
] as const

export type ReminderMinute = (typeof REMINDER_INTERVALS)[number]["value"]
```

- [ ] **Step 3: api.ts**

Create `frontend/src/entities/user-settings/api.ts`:

```ts
import { apiFetch } from "@/shared/api/client"
import type { UserSettings } from "./types"

type DTO = { reminder_minutes: number[] }

export async function getUserSettings(): Promise<UserSettings> {
  const d = await apiFetch<DTO>("/api/miniapp/settings")
  return { reminderMinutes: d.reminder_minutes ?? [] }
}
```

- [ ] **Step 4: write-api.ts**

Create `frontend/src/entities/user-settings/write-api.ts`:

```ts
import { apiFetch } from "@/shared/api/client"

export async function patchUserSettings(reminderMinutes: number[]): Promise<void> {
  await apiFetch("/api/miniapp/settings", {
    method: "PATCH",
    body: JSON.stringify({ reminder_minutes: reminderMinutes }),
  })
}
```

- [ ] **Step 5: queries.ts**

Create `frontend/src/entities/user-settings/queries.ts`:

```ts
import { useQuery } from "@tanstack/react-query"
import { getUserSettings } from "./api"

export const userSettingsKeys = {
  all: ["user-settings"] as const,
  current: () => ["user-settings", "current"] as const,
}

export function useUserSettings() {
  return useQuery({
    queryKey: userSettingsKeys.current(),
    queryFn: getUserSettings,
  })
}
```

- [ ] **Step 6: mutations.ts (with optimistic update)**

Create `frontend/src/entities/user-settings/mutations.ts`:

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { patchUserSettings } from "./write-api"
import { userSettingsKeys } from "./queries"
import type { UserSettings } from "./types"

export function useUpdateReminderMinutes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (reminderMinutes: number[]) => patchUserSettings(reminderMinutes),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: userSettingsKeys.current() })
      const prev = qc.getQueryData<UserSettings>(userSettingsKeys.current())
      qc.setQueryData<UserSettings>(userSettingsKeys.current(), {
        reminderMinutes: [...next].sort((a, b) => a - b),
      })
      return { prev }
    },
    onError: (_err, _next, ctx) => {
      if (ctx?.prev) qc.setQueryData(userSettingsKeys.current(), ctx.prev)
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: userSettingsKeys.current() })
    },
  })
}
```

- [ ] **Step 7: Typecheck + build**

Run: `cd frontend && pnpm typecheck && pnpm build`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/entities/user-settings/
git commit -m "$(cat <<'EOF'
feat(miniapp): entities/user-settings layer

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T6: i18n keys (ru/kk/en)

**Files:**
- Modify: `frontend/src/shared/tma/i18n.ts`

- [ ] **Step 1: Add 8 keys × 3 langs to existing language packs**

Open `frontend/src/shared/tma/i18n.ts`. Structure: `export const I18N = { ru: {...}, kk: {...}, en: {...} }`. APPEND the new keys at the end of each block (preserve existing keys; insert before the closing `}`).

RU block:
```ts
rem10m: "10 мин",
rem15m: "15 мин",
rem30m: "30 мин",
rem1h: "1 час",
rem2h: "2 часа",
rem1d: "1 день",
remindersOff: "Напоминания выключены — выбери интервалы",
settingsSaveFailed: "Не удалось сохранить",
```

KK block:
```ts
rem10m: "10 мин",
rem15m: "15 мин",
rem30m: "30 мин",
rem1h: "1 сағат",
rem2h: "2 сағат",
rem1d: "1 күн",
remindersOff: "Хабарландырулар сөнді — интервалды таңда",
settingsSaveFailed: "Сақтау сәтсіз",
```

EN block:
```ts
rem10m: "10 min",
rem15m: "15 min",
rem30m: "30 min",
rem1h: "1 hour",
rem2h: "2 hours",
rem1d: "1 day",
remindersOff: "Reminders are off — pick intervals",
settingsSaveFailed: "Failed to save",
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: clean. The `as const` keys from `REMINDER_INTERVALS` (T5) now resolve to real `I18nKey` union members.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/i18n.ts
git commit -m "$(cat <<'EOF'
feat(miniapp): i18n keys for slice C (ru/kk/en)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T7: Profile page rewrite

**Files:**
- Modify: `frontend/src/features/profile/pages/profile-page.tsx`

- [ ] **Step 1: Survey the existing block**

Read `frontend/src/features/profile/pages/profile-page.tsx`. Identify and plan to remove:
- `useState(["15m"])` for `reminders`
- `useState(true)` for `remOn`
- Local `intervals` array (it lists values like `"10m"`, `"15m"`, etc.)
- The `remOn && (...)` wrapper around the chip grid
- The `<CatToggle on={remOn} onChange={setRemOn} />` in the reminders `SettingsRow.right`

- [ ] **Step 2: Add imports + hooks**

Near the top of the file, add:

```ts
import { REMINDER_INTERVALS } from "@/entities/user-settings/constants"
import { useUserSettings } from "@/entities/user-settings/queries"
import { useUpdateReminderMinutes } from "@/entities/user-settings/mutations"
import { toastError } from "@/shared/lib/toast"
```

(If `toastError` is already imported from `@/shared/lib/toast`, skip.)

Inside the `ProfilePage` component body, near the other hooks:

```ts
const { data: settings } = useUserSettings()
const updateMut = useUpdateReminderMinutes()
const current = settings?.reminderMinutes ?? []

const toggleInterval = (minutes: number) => {
  const next = current.includes(minutes)
    ? current.filter((m) => m !== minutes)
    : [...current, minutes]
  updateMut.mutate(next, {
    onError: (err) => toastError(err, t("settingsSaveFailed")),
  })
}
```

- [ ] **Step 3: Replace the reminders JSX block**

Replace the existing reminders SettingsGroup + chip grid with:

```tsx
<SettingsGroup title={t("reminders")}>
  <SettingsRow
    icon="bell"
    hue={45}
    label={t("reminders")}
    right={
      current.length > 0 ? (
        <span className="text-miniapp-muted text-xs font-bold">{current.length}</span>
      ) : undefined
    }
    last
  />
</SettingsGroup>
<div className="mx-1 -mt-3 mb-5 flex flex-col gap-2">
  {current.length === 0 && (
    <p className="text-miniapp-muted px-1 text-xs">{t("remindersOff")}</p>
  )}
  <div className="flex flex-wrap gap-2">
    {REMINDER_INTERVALS.map((it) => {
      const on = current.includes(it.value)
      return (
        <button
          key={it.value}
          type="button"
          onClick={() => toggleInterval(it.value)}
          className={cn(
            "font-display cursor-pointer rounded-[11px] border-[1.5px] px-[13px] py-2 text-[13.5px] font-bold",
            on
              ? "border-miniapp-accent bg-miniapp-accent-soft text-miniapp-accent"
              : "border-miniapp-border bg-miniapp-card text-miniapp-muted"
          )}
        >
          {on ? "✓ " : ""}
          {t(it.labelKey)}
        </button>
      )
    })}
  </div>
</div>
```

- [ ] **Step 4: Remove the now-unused state**

Delete these lines from the component body:

```ts
const [reminders, setReminders] = useState(["15m"])
const [remOn, setRemOn] = useState(true)
const intervals = [
  { v: "10m", l: `10 ${t("min")}` },
  // ...rest of local intervals array
]
```

Also remove unused imports: if `CatToggle` is no longer used anywhere in the file, drop its import. If `useState` is no longer used, drop that too. **Verify** by running typecheck.

- [ ] **Step 5: Typecheck + build**

Run: `cd frontend && pnpm typecheck && pnpm build`
Expected: clean.

- [ ] **Step 6: Run frontend tests**

Run: `cd frontend && pnpm test`
Expected: green (sa-validate tests must still pass).

- [ ] **Step 7: Commit**

```bash
git add frontend/src/features/profile/pages/profile-page.tsx
git commit -m "$(cat <<'EOF'
feat(miniapp): server-backed reminder chips on profile page

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task C-T8: Docs refresh + final verification

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: API.md — add the two new Mini App rows**

Open `docs/API.md`. Find the "Mini App — present" table (post-rename). Add these two rows in the right table:

```markdown
| `GET`    | `/api/miniapp/settings`                       | Current user reminder settings              |
| `PATCH`  | `/api/miniapp/settings`                       | Update reminder minutes (whitelist enforced) |
```

If the file structure differs post-rename, add the rows in the closest equivalent section — find the existing `/api/miniapp/me` row and insert nearby.

- [ ] **Step 2: MEETINGS.md — flip §7 row**

Open `docs/MEETINGS.md`. Find any pending row about user settings / §7. Replace with:

```markdown
### User settings (done)

Reminder intervals are user-configurable in the Profile screen, persisted in `bot_users.reminder_minutes`. See [`docs/superpowers/specs/2026-06-09-slice-c-user-settings-design.md`](superpowers/specs/2026-06-09-slice-c-user-settings-design.md). Timezone + language remain Slice H scope.
```

If MEETINGS.md doesn't already mention §7 user settings, append the section near the other "done" rows.

- [ ] **Step 3: Final verification**

```sh
cd backend && go build ./... && go vet ./... && go test ./...
cd ../frontend && pnpm typecheck && pnpm build && pnpm test
```

Report each command's pass/fail. All must pass.

- [ ] **Step 4: Commit**

```bash
git add docs/API.md docs/MEETINGS.md
git commit -m "$(cat <<'EOF'
docs(slice-c): API/MEETINGS reflect user settings live

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Self-review

| Spec section | Plan task |
|--------------|-----------|
| §1 Surface (routes, error codes) | C-T3 (handlers + app.go) |
| §2 Backend: `botsettings.Format` export | C-T1 |
| §2 Backend: `user_settings.go` + TDD | C-T2 |
| §2 Backend: HTTP handlers | C-T3 |
| §3 Frontend: entities/user-settings | C-T5 |
| §3 Frontend: profile-page rewrite | C-T7 |
| §3 Frontend: i18n keys | C-T6 |
| §4 OpenAPI + frontend schema regen | C-T4 |
| §6 Final verification + docs | C-T8 |

**Placeholder scan:** none. Every step has concrete code or commands.

**Type consistency:**
- `UserSettings` defined in T2 (Go) and T5 (TS) — same field name (`ReminderMinutes` / `reminderMinutes`).
- `ErrInvalidReminderMinute` named the same in T2 and T3.
- `REMINDER_INTERVALS.value` (number) matches backend `botsettings.Intervals[].Minutes` (int).
- Route paths `/api/miniapp/settings` consistent across T3 (handlers), T4 (OpenAPI), T5 (api.ts), T8 (docs).

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-09-slice-c-user-settings.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, two-stage review per task
2. **Inline execution** — execute tasks in this session with checkpoints

Which approach?
