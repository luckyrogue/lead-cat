# Operations

## Logs (Dokploy)

JSON zap to stdout. Correlate by `request_id` (Fiber middleware).

Suggested fields in app logs:

- `request_id`
- `workspace_id`
- `scenario_id`
- `run_id`
- `component` — `http`, `bot`, `asynq`, `scheduler`

## Health

`GET /api/health` → `{ "postgres", "redis", "bot_ok", "version" }`

## Metrics

`GET /metrics` — Prometheus exposition format (`prometheus/client_golang`).

Set `METRICS_TOKEN` and send `Authorization: Bearer <token>` if the endpoint is exposed publicly.

Counters:

- `leadcat_http_requests_total{method,path,status}` — `status` is `2xx`, `4xx`, `5xx`, etc.
- `leadcat_scenario_runs_total{status}` — `success` or `failed` when a run finishes

Example scrape (Dokploy / internal network):

```yaml
scrape_configs:
  - job_name: lead-cat
    metrics_path: /metrics
    authorization:
      credentials: "<METRICS_TOKEN>"
    static_configs:
      - targets: ["lead-cat:8080"]
```

## Redis

- Scheduler lock: `leadcat:scheduler:leader` TTL 90s
- See [REDIS.md](REDIS.md)

## Postgres backup

Standard Dokploy Postgres backup / pg_dump.

## Rollback

Redeploy previous GHCR image tag; migrations are forward-only.
