# Cat design system

## Tokens (`frontend/src/shared/theme/cat-tokens.css`)

| Token | Value |
|-------|-------|
| `--cat-bg` | `#FFF8F0` |
| `--cat-primary` | `#E87B35` |
| `--cat-secondary` | `#5B6B7A` |

## Components

- `CatShell` — header logo, paw nav, paw-pattern background
- shadcn overrides via CSS variables
- SVG assets in `shared/assets/cats/`

## Copy

- Save: «Мур, сохранить»
- Error: «Кот уронил сервер»
- Empty: «Пока тихо… заведи логово 🐾»

## Sign-off checklist

- [ ] All routes use CatShell
- [ ] Primary buttons use `--cat-primary`
- [ ] Scenario builder has cat node labels
- [ ] TMA header colors match palette
