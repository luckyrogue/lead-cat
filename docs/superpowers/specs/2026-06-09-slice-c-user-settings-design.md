# Slice C — User settings: design

**Status:** approved (brainstorm), ready for writing-plans.
**Date:** 2026-06-09
**Goal:** Expose the existing per-user reminder preferences (`bot_users.reminder_minutes`) over the Mini App so the Profile screen becomes server-backed instead of mock-local.
**Topic:** ТЗ §7 user settings, scoped to reminders only. Roadmap reference: [`2026-06-06-roadmap-to-beta-design.md`](2026-06-06-roadmap-to-beta-design.md) → Slice C. This is the smallest slice — backend already implements reminders; we add HTTP exposure + frontend wire.

## Locked decisions

| # | Decision |
|---|----------|
| 1 | **Scope** — reminders only. Timezone (ТЗ §7 row 1) and language (ТЗ §7 row 3) deferred to Slice H. Reason: org is Almaty-only (single workspace TZ), multi-TZ value-per-user is near-zero today; language already works via TG WebApp context. |
| 2 | **On/off semantics** — empty `reminder_minutes[]` means off. No separate `enabled` boolean. Matches existing bot `/settings` command behavior; same backend column, single source of truth. |
| 3 | **HTTP shape** — `GET/PATCH /api/miniapp/settings`. Wire format `{"reminder_minutes": [10, 15, 30]}`. Resource is forward-compatible for future TZ/lang fields without a new route. |
| 4 | **Validation** — strict whitelist `{10, 15, 30, 60, 120, 1440}` from `botsettings.Intervals`. Any other minute value → `400 validation_failed`. Optimistic UI updates via TanStack Query. |

## Reality check vs current `main` (HEAD `3ac3095`)

| Surface | Status |
|---------|--------|
| `bot_users.reminder_minutes TEXT NOT NULL DEFAULT '15'` | Persisted, default = 15 min. Used by reminder_scheduler + bot `/settings`. |
| `botsettings.Intervals` | `{10, 15, 30, 60, 120, 1440}` — six fixed values. ТЗ §5.2-compliant. |
| `botsettings.Parse(csv) []int` | Public helper. Silently skips non-int values (legacy-safe). |
| `botsettings.format([]int) string` | Private. **Slice C exports it as `Format`** so application layer can write canonical CSV. |
| `Store.SetReminderMinutes(ctx, tgID, csv)` | Repo method exists. Last-write-wins on a single row. |
| `Store.GetBotUserByTelegramID(ctx, tgID)` | Returns `BotUser` with `ReminderMinutes` field. |
| `/api/miniapp/me` | Identity only (telegram_id, name, email, role). **No `/api/miniapp/settings`** today. |
| Profile page (`frontend/src/features/profile/pages/profile-page.tsx`) | Reminders chips + on/off toggle, **all local `useState`** — does not hit any API. |
| Reminder scheduler | Reads `bot_users.reminder_minutes` per tick. No cache. Changes take effect on the next scheduled tick (typically within a minute). |

## 1. Surface

### Routes

| Method | Path | Body | Response |
|--------|------|------|----------|
| `GET` | `/api/miniapp/settings` | — | `{"reminder_minutes": [10, 15]}` (sorted, may be empty) |
| `PATCH` | `/api/miniapp/settings` | `{"reminder_minutes": [10, 15, 30]}` | `204 No Content` |

Both routes require `Authorization: Bearer <miniapp_jwt>` (existing `miniappAuth.Middleware`). No admin guard — every authenticated bot user can read/write their own settings.

### Identification

User identity comes from `c.Locals("bot_user").(postgres.BotUser).TelegramID`, set by the JWT middleware. Settings are global per user (not workspace-scoped).

### Error codes

| Code | HTTP | When |
|------|------|------|
| `unauthorized` | 401 | Missing or invalid JWT |
| `validation_failed` | 400 | `reminder_minutes` is missing/null/not-an-array; element is not int; element is not in the whitelist |
| `internal_error` | 500 | DB write failed |

### Out of scope

- Timezone setting (ТЗ §7 row 1 — deferred to Slice H)
- Language setting (ТЗ §7 row 3 — deferred to Slice H)
- Per-workspace settings (single workspace model holds)
- Bot `/settings` command — unchanged, continues to work via `botsettings.Service.Toggle`
- Reminder-scheduler changes — already reads `bot_users.reminder_minutes`; no changes needed

## 2. Backend changes

### Reused (zero diff)

- `Store.GetBotUserByTelegramID(ctx, tgID)` — read
- `Store.SetReminderMinutes(ctx, tgID, csv string)` — write
- `botsettings.Parse(csv) []int` + `botsettings.Intervals` — parse + whitelist
- `botsettings.Service` (bot `/settings`) — unchanged, runs in parallel

### Modified

**`backend/internal/platform/botsettings/settings.go`** — export the private `format` helper so the application layer can produce canonical CSV without duplicating logic:

```go
// Format exposes the canonical CSV writer to other packages.
func Format(mins []int) string { return format(mins) }
```

One-line addition next to the existing `Parse` export.

### New files

**`backend/internal/application/user_settings.go`**:

```go
package application

import (
	"context"
	"errors"
	"slices"

	"github.com/Jaryq-Lab/lead-cat/internal/platform/botsettings"
)

// ErrInvalidReminderMinute is returned when a minute value is not in the
// botsettings.Intervals whitelist.
var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

// UserSettings is the per-user settings projection exposed to the Mini App.
type UserSettings struct {
	ReminderMinutes []int `json:"reminder_minutes"`
}

// GetUserSettings returns the authed bot user's settings.
func (s *Services) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	u, err := s.Store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return UserSettings{}, err
	}
	return UserSettings{ReminderMinutes: botsettings.Parse(u.ReminderMinutes)}, nil
}

// SetUserReminderMinutes validates input against the whitelist, dedupes/sorts,
// and writes canonical CSV. Empty slice → empty CSV → reminders disabled.
func (s *Services) SetUserReminderMinutes(ctx context.Context, telegramID int64, minutes []int) error {
	allowed := allowedReminderMinutes()
	for _, m := range minutes {
		if !slices.Contains(allowed, m) {
			return ErrInvalidReminderMinute
		}
	}
	cp := append([]int(nil), minutes...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	return s.Store.SetReminderMinutes(ctx, telegramID, botsettings.Format(cp))
}

func allowedReminderMinutes() []int {
	out := make([]int, 0, len(botsettings.Intervals))
	for _, iv := range botsettings.Intervals {
		out = append(out, iv.Minutes)
	}
	return out
}
```

**`backend/internal/application/user_settings_test.go`** — TDD covers:

| Test | Asserts |
|------|---------|
| `TestSetUserReminderMinutes_WhitelistOK` | `[10, 30]` → no error, store sees CSV `"10,30"` |
| `TestSetUserReminderMinutes_RejectsNonWhitelist` | `[7]` → `ErrInvalidReminderMinute`, no store write |
| `TestSetUserReminderMinutes_DedupeSort` | `[30, 10, 30]` → store sees `"10,30"` |
| `TestSetUserReminderMinutes_Empty` | `[]` → store sees `""` (off) |
| `TestGetUserSettings_ParsesCSV` | Store returns `"15,60"` → result `{ReminderMinutes: [15, 60]}` |

Uses a narrow store mock interface inline, same pattern as `admin_workspace_test.go`.

**`backend/internal/delivery/http/handlers/miniapp_settings.go`**:

```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/lead-cat/internal/application"
	"github.com/Jaryq-Lab/lead-cat/internal/infrastructure/persistence/postgres"
)

func miniAppBotUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// MiniAppGetSettings — GET /api/miniapp/settings
func (a *API) MiniAppGetSettings(c *fiber.Ctx) error {
	bu, ok := miniAppBotUser(c)
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
	bu, ok := miniAppBotUser(c)
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

`*[]int` distinguishes "not sent" from "sent empty array". A nil pointer → 400. An empty array → write empty CSV (reminders off).

### Routing

**`backend/internal/delivery/http/app.go`** — append after `miniapp.Get("/me", api.MiniAppMe)`:

```go
miniapp.Get("/settings", api.MiniAppGetSettings)
miniapp.Patch("/settings", api.MiniAppPatchSettings)
```

## 3. Frontend changes

### New FSD slice — `entities/user-settings/`

```
frontend/src/entities/user-settings/
├── types.ts          UserSettings type
├── constants.ts      REMINDER_INTERVALS (single source of truth)
├── api.ts            getUserSettings()
├── write-api.ts      patchUserSettings()
├── queries.ts        useUserSettings() + key factory
└── mutations.ts      useUpdateReminderMinutes() with optimistic update
```

### Types and constants

```ts
// types.ts
export type UserSettings = { reminderMinutes: number[] }

// constants.ts — must stay in sync with backend/internal/platform/botsettings/settings.go Intervals
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

### API

```ts
// api.ts
import { apiFetch } from "@/shared/api/client"
import type { UserSettings } from "./types"

type DTO = { reminder_minutes: number[] }

export async function getUserSettings(): Promise<UserSettings> {
  const d = await apiFetch<DTO>("/api/miniapp/settings")
  return { reminderMinutes: d.reminder_minutes ?? [] }
}
```

```ts
// write-api.ts
import { apiFetch } from "@/shared/api/client"

export async function patchUserSettings(reminderMinutes: number[]): Promise<void> {
  await apiFetch("/api/miniapp/settings", {
    method: "PATCH",
    body: JSON.stringify({ reminder_minutes: reminderMinutes }),
  })
}
```

### Queries + optimistic mutation

```ts
// queries.ts
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

```ts
// mutations.ts
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

### Profile page rewrite

`frontend/src/features/profile/pages/profile-page.tsx` — replace the local-state reminder block:

```tsx
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

return (
  <>
    <SettingsGroup title={t("reminders")}>
      <SettingsRow
        icon="bell"
        hue={45}
        label={t("reminders")}
        right={
          current.length > 0 ? (
            <span className="text-miniapp-muted text-xs font-bold">
              {current.length}
            </span>
          ) : undefined
        }
        last
      />
    </SettingsGroup>
    <div className="mx-1 -mt-3 mb-5 flex flex-col gap-2">
      {current.length === 0 && (
        <p className="text-miniapp-muted px-1 text-xs">
          {t("remindersOff")}
        </p>
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
    {/* timezone + language groups unchanged */}
  </>
)
```

Removed: `useState(["15m"])`, `useState(true)`, `remOn && (...)` wrapper, local `intervals` array.

### i18n keys

8 new keys × 3 langs = 24 entries:

| Key | RU | EN | KK |
|-----|----|----|----|
| `rem10m` | "10 мин" | "10 min" | "10 мин" |
| `rem15m` | "15 мин" | "15 min" | "15 мин" |
| `rem30m` | "30 мин" | "30 min" | "30 мин" |
| `rem1h` | "1 час" | "1 hour" | "1 сағат" |
| `rem2h` | "2 часа" | "2 hours" | "2 сағат" |
| `rem1d` | "1 день" | "1 day" | "1 күн" |
| `remindersOff` | "Напоминания выключены — выбери интервалы" | "Reminders are off — pick intervals" | "Хабарландырулар сөнді — интервалды таңда" |
| `settingsSaveFailed` | "Не удалось сохранить" | "Failed to save" | "Сақтау сәтсіз" |

Existing keys `"10m"`, `"15m"`, etc. are preserved — they may be referenced elsewhere (meeting detail, etc.).

### Tests

- entity layer typed contract — `pnpm typecheck` enforces
- mutations — no unit tests (project convention)
- profile-page — manual test via dev server

## 4. OpenAPI changes

Both `backend/openapi/openapi.json` and `backend/docs/openapi.json` (byte-identical mirror).

### New paths

```json
"/api/miniapp/settings": {
  "get": {
    "operationId": "miniapp_settings_get",
    "tags": ["miniapp"],
    "summary": "Get current user settings",
    "security": [{ "bearerAuth": [] }],
    "responses": {
      "200": { ... "$ref": "#/components/schemas/MiniAppUserSettings" ... },
      "401": { ... "$ref": "#/components/schemas/ApiErrorResponse" ... },
      "500": { ... }
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
      "400": { ... "validation_failed" ... },
      "401": { ... },
      "500": { ... }
    }
  }
}
```

### New schemas

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
      "items": { "type": "integer", "format": "int32", "enum": [10, 15, 30, 60, 120, 1440] }
    }
  }
}
```

The `enum` constraint in the patch schema mirrors the backend whitelist.

### Frontend schema regen

`npx openapi-typescript backend/openapi/openapi.json -o frontend/src/shared/api/generated/schema.ts` (run from `frontend/`).

## 5. Risks + open questions

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Frontend `REMINDER_INTERVALS` drifts from backend `botsettings.Intervals` | Med | Comment "must stay in sync" on both sides. Slice H may add codegen. |
| Reminder scheduler picks up changes lazily | Low | Scheduler reads `bot_users.reminder_minutes` per tick (no cache). Changes propagate within one tick. Document in UI hint if needed. |
| Concurrent edits via bot `/settings` and Mini App | Low | Single-row last-write-wins. Acceptable — both write the same column. |
| Empty PATCH body resets settings | Low | `*[]int` pointer check; nil → 400. Empty array → write empty CSV intentionally. |
| Legacy DB values outside whitelist (e.g. `45`) | Low | `Parse` returns them as is; UI does not render unknown chips, so they're invisible. Next user PATCH overwrites with whitelist-only set. KISS, no migration. |
| Optimistic update / server reject divergence | Low | TanStack `onError` rolls back; toast surfaces cause. Chips constrain input, so divergence only via curl. |

### Open questions (resolved)

1. **Should the API return updated settings in PATCH response?** → No, 204. Frontend already knows next (optimistic). Saves bytes.
2. **PATCH idempotency** → Yes — same body produces same CSV → same 204.
3. **What about `bot_users.reminder_minutes` values from old data?** → Read silently passes through `Parse`; non-whitelist values surface as ignored chips. Next write canonicalizes.

## 6. File structure (final)

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
│   └── features/profile/pages/profile-page.tsx                 [MODIFY: server-backed]

docs/
├── API.md                                                      [MODIFY: 2 new Mini App paths]
└── MEETINGS.md                                                 [MODIFY: §7 user settings Done]
```

## 7. Effort estimate

| Segment | New LoC | Time |
|---------|---------|------|
| Backend application + handler + TDD | ~250 | 0.5 day |
| OpenAPI + frontend schema regen | ~80 | 0.25 day |
| Frontend entities/user-settings | ~150 | 0.5 day |
| Profile-page rewrite + i18n | ~120 | 0.5 day |
| Docs + verification | ~80 | 0.25 day |
| **Total** | **~680** | **~2 days (≈0.5 week)** |

Smallest slice in the roadmap — backend bones already exist; this slice is HTTP exposure + frontend wire only.

## 8. Self-review

| Check | Status |
|-------|--------|
| Placeholder scan (TBD/TODO/vague) | None |
| Internal consistency (surface ↔ backend ↔ frontend ↔ openapi) | All four reference `reminder_minutes: int[]`, `{10,15,30,60,120,1440}`, `/api/miniapp/settings` |
| Scope check (single plan) | ~680 LoC, ~0.5 week — well under slice budget |
| Ambiguity check | All resolved inline (open questions §5) |

## 9. Next step

Hand off to `writing-plans` to produce the task-by-task implementation plan at `docs/superpowers/plans/2026-06-09-slice-c-user-settings.md`.
