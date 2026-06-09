# Operations

## Logging

Structured JSON (production) or console (local) via **zap**. All output to stdout; Dokploy collects it.

### Configuration

| Env var      | Values                              | Default |
| ------------ | ----------------------------------- | ------- |
| `LOG_LEVEL`  | `debug` / `info` / `warn` / `error` | `info`  |
| `LOG_FORMAT` | `json` (prod) / `console` (local)   | `json`  |

### Context propagation

Fields are attached to the context by middleware and passed through the call stack with `log.WithContext`. Standard fields:

- `request_id` — set by Fiber middleware on every HTTP request
- `workspace_id` — set whenever a workspace is resolved from the Mini App session
- `meeting_id` — set in meeting-scoped handlers and workers
- `component` — `http`, `bot`, `asynq`, `scheduler`

### Log message conventions

Stable snake_case event names + structured fields; no interpolated sentences.

Examples:

```
miniapp_meeting_created   telegram_id=123  meeting_id=456  workspace_id=789
miniapp_auth_ok           telegram_id=123  workspace_id=789
reminder_enqueued         meeting_id=456   fire_at=2026-06-06T09:00:00Z
```

**Never log:** secrets, JWT tokens, Telegram `initData`, or any PII.

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

Run against a live environment (local or staging) with a valid Telegram test account.

1. `POST /api/auth/miniapp` with dev `initData` → receive Mini App JWT. Verify `200` and a non-empty `token` field.
2. `GET /api/miniapp/me` with `Authorization: Bearer <token>` → user identity returned.
3. `GET /api/miniapp/meetings?scope=all` → `200` with an array.
4. `POST /api/miniapp/meetings` (body: title, start, end, participants) → `201` with meeting body including `id`.
5. `POST /api/miniapp/free-slots` (body: participants + time window) → `200` with slot list.
6. `PATCH /api/miniapp/meetings/:id` → updated meeting returned.
7. `DELETE /api/miniapp/meetings/:id` → `204`.
8. `POST /api/miniapp/conflicts` → conflict report for a given time range.

---

## Master encryption key rotation

Lead Cat encrypts service-account JSON at rest with `MASTER_ENCRYPTION_KEY`. Rotating the key today requires brief downtime:

1. Generate a new 32-byte key: `openssl rand -base64 32`
2. In Google Cloud Console, revoke the existing SA private key (audit only — the SA itself remains valid)
3. Stop the service: `dokploy stop lead-cat`
4. Set the new `MASTER_ENCRYPTION_KEY` env var in Dokploy
5. Start the service
6. Open Mini App → Profile → Admin → Setup → Integrations → paste a freshly-issued SA JSON
7. Click "Verify" — confirm calendar metadata returns

The DB column `workspaces.google_sa_json_enc` will be unrecoverable until step 6. Loss of `MASTER_ENCRYPTION_KEY` follows the same recipe: the SA in Google remains valid, you just re-upload through the admin UI.

Zero-downtime rotation (two-key handoff with bulk re-encrypt) is post-beta backlog.
