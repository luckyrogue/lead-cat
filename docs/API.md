# REST API

Base: `/api` — requires `Authorization: Bearer <JWT>` unless noted.

## Public

- `GET /api/health` — Postgres, Redis, bot_ok, version
- `GET /metrics` — Prometheus text (optional `METRICS_TOKEN`)

## Me

- `GET /api/me`
- `POST /api/me/link-telegram` — header `X-Telegram-Init-Data`

## Workspaces

- `GET /api/workspaces`
- `POST /api/workspaces` — `{ "name", "slug" }`
- `GET /api/workspaces/:id` — requires workspace access
- `POST /api/workspaces/:id/chat/link`
- `GET /api/workspaces/:id/chat/status`

## Members

- `GET /api/workspaces/:id/members` — includes `github_login`, `gitlab_login`
- `POST /api/workspaces/:id/members`
- `DELETE /api/workspaces/:id/members/:memberId`
- `POST /api/workspaces/:id/members/sync-chat`
- `PATCH /api/workspaces/:id/members/:username/vcs`

## Integrations

- `GET /api/workspaces/:id/integrations`
- `PATCH /api/workspaces/:id/integrations`
- `POST /api/workspaces/:id/integrations/verify`

## Scenarios

- `GET /api/workspaces/:id/scenarios`
- `POST /api/workspaces/:id/scenarios`
- `GET /api/workspaces/:id/scenarios/:sid`
- `PATCH /api/workspaces/:id/scenarios/:sid`
- `DELETE /api/workspaces/:id/scenarios/:sid`
- `POST /api/workspaces/:id/scenarios/:sid/run`
- `GET /api/workspaces/:id/scenarios/:sid/runs`
- `GET /api/workspaces/:id/employees`
- `GET /api/workspaces/:id/meetings`
- `POST /api/workspaces/:id/meetings`
- `GET /api/workspaces/:id/meetings/:mid`
- `DELETE /api/workspaces/:id/meetings/:mid`

Errors: `{ "error": "code", "message": "кошачий текст" }`
