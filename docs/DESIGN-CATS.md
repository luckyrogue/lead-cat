# Cat design system

## Tokens (`frontend/src/shared/theme/cat-tokens.css`)

| Token             | Value     |
| ----------------- | --------- |
| `--cat-bg`        | `#FFF8F0` |
| `--cat-primary`   | `#E87B35` |
| `--cat-secondary` | `#5B6B7A` |

Display font: **Baloo 2** (`--font-display`). Body: **Inter** (`--font-body`).

## Mini App shell

Main app at `/` uses the Telegram Mini App layout:

- `components/miniapp-shell/` — `TgBar`, `TabBar` + FAB, `Sheet`, `Overlay`, `miniapp-layout.tsx`
- `routes/_miniapp/*` — TanStack Router layout + tab/overlay routes with loaders
- `features/{home,meetings,meeting-create,checker,profile}/` — vertical slices
- `features/auth/` — Mini App login, session, refresh-session, auth-context
- `entities/{meeting,employee,admin}/` — domain types + API hooks
- `shared/api/` — client, types, query-client, query-keys, list-params, health, generated/
- `shared/auth/` — module-policies, route-access, permissions, require-permission
- `shared/lib/` — list-url-params, use-list-url-state, route-revalidation, toast
- `shared/miniapp/` — i18n, palette, context (runtime only)
- `components/miniapp-list-page-shell.tsx` — list page shell (sadu `admin-list-page-shell`)
- `components/maintenance-screen.tsx` — health gate fallback
- `app/app-content.tsx` — health + Toaster + router (sadu `root.tsx`)

The SPA is Mini App-only at `/`. Operator setup uses `/api/miniapp/admin/*` inside Telegram — see [API.md](API.md). Frontend structure: [frontend/README.md](../frontend/README.md).

### Sadu admin mapping

| Sadu                             | Lead Cat                          |
| -------------------------------- | --------------------------------- |
| `components/app-sidebar.tsx`     | `components/miniapp-shell/` TabBar |
| `getVisibleSidebarModules`       | `getVisibleTabBarModules`         |
| `shared/auth/module-policies.ts` | same path (Mini App roles)        |
| `features/auth/session.ts`       | `shared/auth/session.ts`          |
| `shared/api/generated/schema.ts` | same (from `/openapi.json`)       |

## Components

- shadcn overrides via CSS variables in `app/app.css`
- Cat primitives in `shared/ui/cat/` (`CatBtn`, `CatCard`, `CatIcon`, `Paw`, …)

## Copy

- Save: «Мур, сохранить»
- Error: «Кот уронил сервер»
- Empty: «Пока тихо… заведи логово 🐾»

## Sign-off checklist

- [x] Main route uses Mini App shell + cat tokens
- [x] Primary accent `#E87B35`, paw pattern background
- [x] Tab bar + create FAB
- [x] Frontend structure aligned with sadu/admin patterns
- [x] Meetings/checker wired to backend API
- [ ] Mini App header colors match Telegram theme params
