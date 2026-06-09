# Lead Cat frontend (TMA)

Telegram Mini App — thin client to [backend API](../backend). Architecture mirrors [sadu/admin](https://github.com/jaryq-lab/sadu) patterns; UI stays cat-design + TabBar.

**Stack:** Vite, React 19, TypeScript, TanStack Router, TanStack Query, axios, Sonner.

## Quick start

```bash
pnpm install
cp .env.example .env   # if present
pnpm dev
# → http://localhost:3000
```

Backend (separate terminal):

```bash
cd ..
make dev
# → API http://localhost:8080
```

## Structure (`frontend/src/`)

```
src/
├── app/                    # providers, app-content (health gate), router
├── routes/_tma/*           # thin routes → feature pages
├── components/
│   ├── tma-shell/          # tg-bar, tab-bar, sheet, overlay, index barrel
│   ├── meetings/           # detail-row, meeting-title-preview, meeting-ui/*
│   ├── employee-picker.tsx
│   └── tma-list-page-shell, maintenance-screen, auth/
├── features/               # vertical slices
│   ├── auth/               # require-auth (route guard)
│   ├── home|meetings|meeting-create|checker|auto|profile/
│   │   meeting-create/     # wizard steps, mini-calendar, use-create-wizard
│   │   meetings/           # meeting-detail*, pages/, search-schema, list-url
│   │   checker/            # checker-* sections
│   │   profile/            # settings-group, profile-header
├── entities/
│   ├── employee/           # types, fixtures, api, queries (search)
│   ├── meeting/            # api, queries, write-api, mutations, scheduling-api, scheduling-queries, lib/
│   └── scenario/           # types
└── shared/
    ├── api/
    ├── auth/               # types, session, tma-api, auth-context, refresh-session, …
    ├── lib/                # cn, toast, use-list-url-state, …
    ├── tma/                # i18n, stored-lang, palette, context, surface-vars
    └── ui/cat/             # ChipGrid, DurationPicker, Segmented, … + primitives.tsx
```

Imports: `@/features/...`, `@/shared/...`, `@/entities/...`.

### Rules

1. `features/X` must not import `features/Y` — use `shared`, `entities`, `components`.
2. Routes are thin: `loader` + `queryOptions` → page UI.
3. Navigation and guards from `shared/auth/module-policies.ts` (TabBar ← `getVisibleTabBarModules`).
4. Keep source files **≤300 lines**; split large pages into feature `components/` and shared `components/`.

## Data scopes

- **Home** — `scope=upcoming` (not `all`); today/upcoming derived via `entities/meeting/lib/home-meetings.ts`
- **Meetings list** — `scope` from URL filter (`upcoming` / `past` / `all`); `scope=all` when detail or success sheet is open
- **Detail route** loader prefetches `all` for meeting lookup in cache

## Dev env

Optional ngrok / tunnel host for Vite (repo root or `frontend/.env`):

```bash
VITE_DEV_ALLOWED_HOSTS=your-subdomain.ngrok-free.app
```

## HTTP client (axios)

`shared/api/client.ts` — axios `api` + `apiFetch`. Bearer from `shared/auth/session.ts` (`lc.tma.auth` in sessionStorage). 401 on `/tma/*` → single-flight `refreshTmaSessionIfNeeded`.

## Tooling

```bash
pnpm lint          # ESLint + FSD import boundaries
pnpm test          # vitest (unit + component tests)
pnpm typecheck
pnpm build
```

## OpenAPI / codegen

```bash
# backend must serve GET /openapi.json
pnpm openapi:generate
# → src/shared/api/generated/schema.ts
```

Spec source: `backend/openapi/openapi.json`.

## URL list state

`useListUrlState` syncs debounced `q`, filters, `page` to query string (`replace: true`). Loader routes use `shouldReload: shouldReloadExceptSearch` so search-only nav does not refetch.

## Auth (TMA)

| Action | Endpoint |
|--------|----------|
| Login | `POST /api/auth/tma` `{ init_data }` |
| Session | `shared/auth/session.ts` |
| Refresh | `shared/auth/refresh-session.ts` (re-login via initData) |
| Me | `GET /api/tma/me` |

## Sadu mapping

| Sadu admin | Lead Cat TMA |
|------------|--------------|
| `components/app-sidebar.tsx` + `getVisibleSidebarModules` | `components/tma-shell/` + `getVisibleTabBarModules` |
| `admin-list-page-shell` | `tma-list-page-shell` |
| `root.tsx` health + Toaster | `app/app-content.tsx` |
| `features/auth/session.ts` | `shared/auth/session.ts` |
| `shared/auth/route-access.ts` | `shared/auth/route-access.ts` (roles) |
