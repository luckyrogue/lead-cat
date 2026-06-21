# WS7b — Load / Performance (design)

## Context: WS7 hardening, sub-project b of 3

WS7 = accessibility (7a, done) + load/perf (7b, this) + pentest/DAST (7c, next). 7b gives
the team a repeatable load-testing harness and baseline numbers for the key public read
path — the "no load testing today" gap from the prod-readiness review. It rides the
existing docker-compose stack (`deploy/docker-compose.e2e.yml`, `CALENDAR_STUB=true` so
no external Google calls).

## Problem

There is no way to measure how the backend behaves under concurrent load, or to confirm
the rate limiter sheds excess traffic gracefully rather than collapsing. Two obstacles
shape the design:
- The rate limits are **per-IP** (`GET /api/book/:slug` = 60/min, etc.) and the limit
  values are **hardcoded constants** in `app.go`. From a single load-runner IP, a read
  endpoint hits `429` after 60 req/min — so true app throughput can't be measured without
  a way to disable the limiter.
- Load tests are latency-sensitive to the runner, so they make a poor blocking CI gate.

## Goal

An **on-demand** k6 harness (run via a script over the compose stack, non-blocking) that
measures (a) the backend's real throughput/latency on the key public read path with rate
limiting disabled, and (b) that the rate limiter correctly returns `429` under overload
with limits enabled — plus committed baseline numbers. Plus the minimal backend change
needed to disable rate limiting in a non-production load stack.

## Design

### 1. Backend — `RATE_LIMIT_DISABLED` (minimal, non-prod gated)

- Add `RateLimitDisabled bool` to `config.Config` from env `RATE_LIMIT_DISABLED`
  (`strings.EqualFold(..., "true")`, same style as `AuthDevMode`).
- In `config.go` validation, under `if cfg.IsProduction()`, error if
  `RateLimitDisabled` is set — identical guard to `AUTH_DEV_MODE` (it can never be
  enabled in production).
- In `app.go`, add a local wrapper instead of changing the `middleware.RateLimit`
  signature:
  ```go
  rateLimit := func(max int, window time.Duration, prefix string) fiber.Handler {
      if cfg.RateLimitDisabled {
          return func(c *fiber.Ctx) error { return c.Next() }
      }
      return middleware.RateLimit(rdb, log, max, window, prefix, cfg.TrustProxyHeaders, cfg.AuthDevMode)
  }
  ```
  Replace the 5 `middleware.RateLimit(...)` call sites with `rateLimit(max, window, prefix)`.
  The `middleware.RateLimit` function and `ratelimit_test.go` are unchanged.

### 2. Load harness `load/` (mirrors `e2e/`)

A self-contained directory: k6 scripts + `run.sh` + a SQL seed fixture + README.

- **`load/run.sh`**: brings up the compose stack (reusing `deploy/docker-compose.e2e.yml`,
  optionally with a `RATE_LIMIT_DISABLED=true` override for the capacity run), waits for
  `/api/health`, applies the SQL seed, runs the requested k6 script(s), prints the
  summary, tears the stack down on exit. Accepts which scenario(s) to run.
- **Seed (`load/seed.sql`)**: deterministically inserts one organization, one host
  `platform_user`, and one **active** `booking_event_type` with a FIXED slug
  (e.g. `loadtest-intro`), applied via `psql`/`docker compose exec` to the stack's
  Postgres before k6. No auth flow needed; with `CALENDAR_STUB` the booking page computes
  slots. (Match the real column set of those tables in the migration.)
- **`load/capacity.js`** (run against a stack with `RATE_LIMIT_DISABLED=true`): a ramping
  VU profile against `GET /api/health` (raw-server baseline) and
  `GET /api/book/loadtest-intro` (real read: DB + availability). k6 `thresholds`:
  `http_req_failed` rate `< 0.01`, `http_req_duration` p95 under a soft target (e.g.
  `p(95)<500` for book, tighter for health). Reports RPS + latency percentiles.
- **`load/shedding.js`** (run against a stack with limits ENABLED — default): a single VU
  bursts `GET /api/book/loadtest-intro` above 60/min and asserts (k6 `checks`) that a
  meaningful fraction return `429`, **zero** return 5xx, and `/api/health` still responds
  during/after the burst (the limiter sheds, the server survives).

### 3. On-demand, non-blocking; baseline captured

- Run via `bash load/run.sh [capacity|shedding|all]` (and a `make load` convenience
  target). Not wired into the PR CI gate (latency-flaky). Thresholds are soft signals.
- `load/README.md` documents how to run it and records the observed baseline numbers
  (RPS, p50/p95/p99 for health + book, the shedding result) so future runs have a
  reference.

### Out of scope (7b)

- The write/booking-submit path under load (`POST /api/book/:slug` 10/hour, write
  side-effects) and the magic-link/auth endpoints (very low limits) — read-path + shedding
  is the meaningful public-load signal.
- A blocking CI performance gate.
- Distributed/multi-IP load generation (single-runner is sufficient for the baseline; the
  capacity run disables limits to measure app throughput regardless of per-IP caps).
- Profiling/flame-graphs and DB tuning — 7b measures and baselines; remediation of any
  hotspot it surfaces is a follow-up.

## Error handling / fallbacks

- `run.sh` fails loudly if the stack doesn't become healthy or the seed fails (no false
  "passed" with an empty target).
- `RATE_LIMIT_DISABLED` is impossible in production (config validation error), so the
  bypass cannot weaken a real deployment.
- The capacity run uses `RATE_LIMIT_DISABLED=true`; the shedding run must NOT set it
  (limits on) — `run.sh` controls this per scenario.

## Testing / verification

- `go build ./... && go vet` + the existing `ratelimit_test.go` still pass (middleware
  unchanged); a small `config_test.go` case for the new `RATE_LIMIT_DISABLED` prod-guard.
- A manual run of `bash load/run.sh all` against the stack: capacity scenario completes
  within thresholds (or the baseline is recorded), and the shedding scenario shows `429`s
  with zero 5xx. The observed numbers are written into `load/README.md`.
- This harness is on-demand and Docker-dependent; it is not part of the blocking CI suite.
