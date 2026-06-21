# Load Test Harness

On-demand k6 load tests for the Lead Cat backend. **Not a CI gate** — run manually to capture baselines and regression-check after significant changes.

## Prerequisites

- Docker (tested with 29.3.1)
- No local k6 install needed — runs via `grafana/k6` Docker image

## Quick Start

```bash
bash load/run.sh [capacity|shedding|all]
```

| Argument   | What it does                                                                    |
|------------|---------------------------------------------------------------------------------|
| `capacity` | Throughput/latency baseline with rate limits **disabled** (RATE_LIMIT_DISABLED=true) |
| `shedding` | Confirms rate limiter sheds 429s at burst without 5xx                           |
| `all`      | Runs capacity then shedding (default)                                           |

The script:
1. Starts the compose stack (`deploy/docker-compose.e2e.yml`) under project `leadcat-load` (safe to run alongside the e2e stack — separate project name)
2. Seeds a deterministic booking fixture (`loadtest-intro` slug via `load/seed.sql`)
3. Runs k6 via Docker
4. Tears down on exit (trap)

### RATE_LIMIT_DISABLED note

`RATE_LIMIT_DISABLED=true` is a **non-production safety guard** — the config rejects it when `APP_ENV=production`. For the capacity run, `deploy/docker-compose.load.yml` injects it into the backend container. The shedding run uses `deploy/docker-compose.load-shedding.yml` (limits on) so 429 shedding can be confirmed.

### macOS vs Linux

On macOS (Docker Desktop), `--network host` is unavailable. The script auto-detects the OS and routes k6 traffic through `host.docker.internal:8081` — the backend port exposed directly (bypassing the admin nginx proxy on :8090, which exhausts ephemeral ports at high RPS on the bridge network). On Linux, `--network host` is used with `localhost:8081`.

### Makefile shortcut

```bash
make load    # equivalent to: bash load/run.sh all
```

## Scenarios

### `capacity.js` — throughput & latency (limits off)

- Two concurrent `ramping-vus` scenarios: `/api/health` and `/api/book/loadtest-intro`
- Ramp 0 → 20 VUs over 10s, hold 20s, ramp down 5s
- Soft thresholds: health p95 < 200ms, book p95 < 500ms, error rate < 1%
- k6 talks directly to the backend on `:8081` (bypasses nginx)

### `shedding.js` — rate-limiter validation (limits on)

- `constant-arrival-rate`: 30 req/s for 20s → 600 total requests
- Rate limiter allows 60/min per IP → expect ~540 429s
- Pass criteria (asserted by `run.sh`): 429s > 0, 5xx = 0, `/api/health` returns 200 after burst

## Baseline (2026-06-21, MacBook M-series, Docker Desktop 4.x)

Results captured running `bash load/run.sh all` against the local Docker Compose stack.
k6 talks directly to the backend via `host.docker.internal:8081` (not through nginx).

### Capacity (rate limits disabled)

| Metric | health | book |
|--------|--------|------|
| Total RPS (both scenarios) | ~10,114 req/s | (combined) |
| p50 | 2.31 ms | 3.67 ms |
| p90 | 3.88 ms | 5.97 ms |
| p95 | **4.54 ms** ✓ (<200ms) | **6.89 ms** ✓ (<500ms) |
| p99 | — | — |
| Error rate | 0.00% ✓ (<1%) | 0.00% ✓ (<1%) |
| Total requests | ~353,987 in 35s | |

All thresholds passed: health p95 < 200ms, book p95 < 500ms, error rate < 1%.

### Shedding (rate limits enabled, 30 req/s × 20s = 600 total)

| Metric | Value |
|--------|-------|
| Total requests | 600 |
| 429 responses | 540 (90%) |
| 200 responses | 60 (within 60/min limit) |
| 5xx responses | **0** |
| Health after burst | **200 OK** |
| Result | **PASS** |

The rate limiter correctly shed 540/600 requests as 429, served 0 5xx errors, and the backend remained healthy after the burst.

## Notes

- k6 bypasses nginx (port 8081 direct) to avoid macOS Docker Desktop bridge network ephemeral-port exhaustion at ~1700 RPS. This is infrastructure-specific; in production (Linux) or with a proper DNS resolver, nginx would work fine.
- The seed is deterministic (fixed UUIDs + `loadtest-intro` slug). Re-runs are idempotent due to `ON CONFLICT (id) DO NOTHING`.
- These numbers are **not production numbers** — Docker Desktop introduces extra latency vs a native Linux host.
