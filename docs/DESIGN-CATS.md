# Cat design system

## Tokens (`frontend/src/shared/theme/cat-tokens.css`)

| Token             | Value     |
| ----------------- | --------- |
| `--cat-bg`        | `#FFF8F0` |
| `--cat-primary`   | `#E87B35` |
| `--cat-secondary` | `#5B6B7A` |

Display font: **Baloo 2** (`--font-display`). Body: **Inter** (`--font-body`).

## TMA shell (redesign)

Main app at `/` uses the Telegram Mini App layout:

- `widgets/tma-shell/` — `TgBar`, `TabBar` + FAB, `Sheet`, `Overlay`, toast, language dropdown
- `features/tma/` — Home, Meetings, Checker, Auto, Profile, create wizard (mock data until API)
- `shared/tma/` — palette, i18n (ru/kk/en), mock meetings/scenarios

Legacy admin routes (`/workspaces`, `/scenarios`, `/team`, …) keep `CatShell`.

## Components

- shadcn overrides via CSS variables in `app/app.css`
- Cat primitives in `shared/ui/cat/` (`CatBtn`, `CatCard`, `CatIcon`, `Paw`, …)

## Copy

- Save: «Мур, сохранить»
- Error: «Кот уронил сервер»
- Empty: «Пока тихо… заведи логово 🐾»

## Sign-off checklist

- [x] Main route uses TMA shell + cat tokens
- [x] Primary accent `#E87B35`, paw pattern background
- [x] Tab bar + create FAB
- [ ] Wire meetings/checker to backend API
- [ ] TMA header colors match Telegram theme params
