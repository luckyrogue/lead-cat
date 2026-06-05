# Operations

## Logging

Structured JSON (production) or console (local) via **zap**. All output to stdout; Dokploy collects it.

### Configuration

| Env var      | Values                              | Default  |
| ------------ | ----------------------------------- | -------- |
| `LOG_LEVEL`  | `debug` / `info` / `warn` / `error` | `info`   |
| `LOG_FORMAT` | `json` (prod) / `console` (local)   | `json`   |

### Context propagation

Fields are attached to the context by middleware and passed through the call stack with `log.WithContext`. Standard fields:

- `request_id` — set by Fiber middleware on every HTTP request
- `workspace_id` — set whenever a workspace is resolved from the TMA session
- `meeting_id` — set in meeting-scoped handlers and workers
- `component` — `http`, `bot`, `asynq`, `scheduler`

### Log message conventions

Stable snake_case event names + structured fields; no interpolated sentences.

Examples:

```
tma_meeting_created   telegram_id=123  meeting_id=456  workspace_id=789
tma_auth_ok           telegram_id=123  workspace_id=789
reminder_enqueued     meeting_id=456   fire_at=2026-06-06T09:00:00Z
```

**Never log:** secrets, OTP codes, JWT tokens, Telegram `initData`, or any PII.

Errors are wrapped with `%w` up the stack and logged once at the boundary (HTTP handler or asynq worker) — not re-logged at every layer.

---

## Health & Metrics

### Health

```
GET /api/health
```

Returns a JSON object with component statuses (postgres, redis, bot connectivity, server version). Use this as the liveness/readiness probe in Dokploy.

### Metrics

```
GET /metrics
```

Prometheus exposition format (`prometheus/client_golang`). Protect with `METRICS_TOKEN`:

```
Authorization: Bearer <METRICS_TOKEN>
```

**Active counters:**

- `leadcat_http_requests_total{method,path,status}` — all HTTP requests; `status` bucket is `2xx`, `4xx`, `5xx`, etc.

**Removed:** `leadcat_*_runs_total` (the background job counter from the old engine is no longer present).

Example Prometheus scrape config (Dokploy internal network):

```yaml
scrape_configs:
  - job_name: lead-cat
    metrics_path: /metrics
    authorization:
      credentials: "<METRICS_TOKEN>"
    static_configs:
      - targets: ["lead-cat:8080"]
```

---

## Postgres

Managed via Dokploy's built-in Postgres service.

**Backup:** use Dokploy scheduled pg_dump or snapshot the volume before any destructive migration.

**Migrations:** forward-only (no down migrations). Run `make migrate` or the equivalent `atlas migrate apply`. If a migration fails mid-deploy, the previous image can be redeployed safely — the schema it expects is still valid.

**Rollback:** redeploy the previous GHCR image tag. Do not run `migrate down`; fix forward instead.

---

## Redis

asynq task queues + scheduler leader lock. See [REDIS.md](REDIS.md) for queue names, monitoring, and failure runbook.

Key operational note: `leadcat:scheduler:leader` (TTL 90 s) ensures only one replica fires reminder/notification jobs at a time. If reminders stop firing, check that exactly one replica holds the lock and that `REDIS_URL` is reachable.

---

## Meetings smoke checklist (E2E)

Run against a live environment (local or staging) with a valid Telegram test account. Steps marked **(planned)** require endpoints not yet implemented.

1. **(present)** `POST /api/auth/tma` with dev `initData` → receive TMA JWT. Verify `200` and a non-empty `token` field.
2. **(present)** `GET /api/tma/me` with `Authorization: Bearer <token>` → workspace profile returned.
3. **(present)** `GET /api/tma/meetings?scope=all` → `200` with an array (empty or non-empty depending on fixture state).
4. **(present)** `POST /api/tma/meetings` (body: title, start, end, participants) → `201` with meeting body including `id`.
5. **(present)** `POST /api/tma/free-slots` (body: participants + time window) → `200` with slot list.
6. **(planned)** `PATCH /api/tma/meetings/:id` → updated meeting returned.
7. **(planned)** `DELETE /api/tma/meetings/:id` → `204`.
8. **(planned)** `POST /api/tma/conflicts` → conflict report for a given time range.
