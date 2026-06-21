# WS7b — Load / Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a non-prod `RATE_LIMIT_DISABLED` toggle and an on-demand k6 load harness that measures backend capacity on the key public read path and confirms the rate limiter sheds excess traffic gracefully — with committed baseline numbers.

**Architecture:** A small backend change adds `RATE_LIMIT_DISABLED` (config, non-prod gated) wired via a local `rateLimit` wrapper in `app.go`. A new `load/` directory holds k6 scripts (`capacity.js`, `shedding.js`), a deterministic `seed.sql`, and `run.sh` that brings up the existing compose stack (`CALENDAR_STUB=true`), seeds a fixed booking event type, runs k6, prints the summary, and tears down. On-demand, not a blocking CI gate.

**Tech Stack:** Go (config + Fiber middleware wiring), k6, docker-compose (`deploy/docker-compose.e2e.yml`), Postgres (seed via psql).

## Global Constraints

- **`RATE_LIMIT_DISABLED` is non-prod only:** config validation must error if it is set when `APP_ENV=production` (identical guard to `AUTH_DEV_MODE`). It can never weaken a real deployment.
- **Do not change the `middleware.RateLimit` signature** or `ratelimit_test.go` — wrap it in `app.go`.
- **On-demand, non-blocking:** the harness runs via `bash load/run.sh`; it is NOT added to the PR CI gate. Thresholds are soft signals; the script reports and records baselines.
- **Docker prerequisite** for Task 2: the harness needs the compose stack. Run docker/compose/k6 commands with the sandbox disabled.
- **Seed self-verifies:** `run.sh` asserts `GET /api/book/<slug>` returns 200 after seeding; a bad/incomplete seed fails loudly (no load run against an empty target).
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Backend `RATE_LIMIT_DISABLED` (config + wrapper)

No Docker needed; verifiable by `go test`/`go build`.

**Files:**
- Modify: `apps/backend/internal/platform/config/config.go` (field + parse + prod guard)
- Modify: `apps/backend/internal/delivery/http/app.go` (local `rateLimit` wrapper + 5 call sites)
- Test: `apps/backend/internal/platform/config/config_test.go`

**Interfaces:**
- Produces: `config.Config.RateLimitDisabled bool`; an internal `rateLimit(max, window, prefix) fiber.Handler` wrapper in `app.go`.

- [ ] **Step 1: Add the failing config test**

In `config_test.go`, add a case asserting the prod guard (mirror the existing `AUTH_DEV_MODE` prod-guard test if present):

```go
func TestRateLimitDisabled_BlockedInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("RATE_LIMIT_DISABLED", "true")
	// plus the other env required for Load() to reach the prod validation block
	// (DATABASE_URL, REDIS_URL, MASTER_ENCRYPTION_KEY, JWT_SECRET, METRICS_TOKEN) —
	// copy the setup from the existing AUTH_DEV_MODE prod-guard test in this file.
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_DISABLED") {
		t.Fatalf("expected RATE_LIMIT_DISABLED prod guard error, got %v", err)
	}
}
```

> Read the existing `config_test.go` to copy the exact env-setup helper the other `Load()` tests use; match it so the test reaches the `IsProduction()` validation block.

Run: `cd apps/backend && go test ./internal/platform/config/ -run TestRateLimitDisabled 2>&1 | head` — Expected: FAIL (field + guard don't exist).

- [ ] **Step 2: Add the field + parse + guard in `config.go`**

Add to the `Config` struct near `AuthDevMode`:
```go
	RateLimitDisabled bool
```
Parse near the `AuthDevMode` parse line (~line 106):
```go
	cfg.RateLimitDisabled = strings.EqualFold(os.Getenv("RATE_LIMIT_DISABLED"), "true")
```
In the `if cfg.IsProduction() {` validation block, alongside the `AuthDevMode` guard:
```go
		if cfg.RateLimitDisabled {
			return cfg, fmt.Errorf("RATE_LIMIT_DISABLED must not be set when APP_ENV=production")
		}
```

- [ ] **Step 3: Add the `rateLimit` wrapper + swap call sites in `app.go`**

Near the top of the function that registers routes (where `rdb`, `log`, `cfg` are in scope), add:
```go
	rateLimit := func(max int, window time.Duration, prefix string) fiber.Handler {
		if cfg.RateLimitDisabled {
			return func(c *fiber.Ctx) error { return c.Next() }
		}
		return middleware.RateLimit(rdb, log, max, window, prefix, cfg.TrustProxyHeaders, cfg.AuthDevMode)
	}
```
Replace the 5 `middleware.RateLimit(rdb, log, MAX, WINDOW, "PREFIX", cfg.TrustProxyHeaders, cfg.AuthDevMode)` call sites with `rateLimit(MAX, WINDOW, "PREFIX")`:
- `/api/auth/miniapp` → `rateLimit(10, time.Minute, "miniapp_auth")`
- `/magic/request` → `rateLimit(5, 15*time.Minute, "magic")`
- `/magic/verify` → `rateLimit(10, time.Minute, "magic_verify")`
- `GET /api/book/:slug` → `rateLimit(60, time.Minute, "book_get")`
- `POST /api/book/:slug` → `rateLimit(10, time.Hour, "book_post")`

- [ ] **Step 4: Run tests + build + vet**

Run: `cd apps/backend && go test ./internal/platform/config/ && go build ./... && go vet ./internal/platform/config/ ./internal/delivery/http/`
Expected: config test PASS (incl. new guard), build + vet clean, `ratelimit_test.go` still passes (untouched).

- [ ] **Step 5: Commit**

```bash
git status --porcelain   # stage exactly the touched paths
git add apps/backend/internal/platform/config/config.go apps/backend/internal/platform/config/config_test.go apps/backend/internal/delivery/http/app.go
git commit -m "$(cat <<'EOF'
feat(load): RATE_LIMIT_DISABLED toggle (non-prod) for load testing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: k6 load harness (`load/`) + baseline

Docker-dependent. Builds the harness and runs it to capture baseline numbers.

**Files:**
- Create: `load/seed.sql`, `load/capacity.js`, `load/shedding.js`, `load/run.sh`, `load/README.md`
- Modify: `Makefile` (add a `load` target — optional convenience)

**Interfaces:**
- Consumes (Task 1): `RATE_LIMIT_DISABLED` env on the backend.

- [ ] **Step 1: `load/seed.sql` — deterministic booking fixture**

Insert one org + host platform_user + active event type with a FIXED slug. Use the real column sets from the migrations (platform_users: `id, auth_sub, email`; booking_event_types: `host_user_id, organization_id, slug, title, duration_mins` + defaulted availability cols). CONFIRM the `organizations` table's required columns from `apps/backend/migrations/` (it's the SaaS-era org table, not legacy `workspaces`) and include them:

```sql
-- Fixed UUIDs so the slug + host are deterministic across runs.
INSERT INTO organizations (id, /* required cols: name/slug/tz per migration */ ...)
VALUES ('11111111-1111-1111-1111-111111111111', /* ... */)
ON CONFLICT (id) DO NOTHING;

INSERT INTO platform_users (id, auth_sub, email)
VALUES ('22222222-2222-2222-2222-222222222222', 'loadtest-host', 'loadhost@e2e.test')
ON CONFLICT (id) DO NOTHING;

INSERT INTO booking_event_types
  (id, host_user_id, organization_id, slug, title, description, duration_mins,
   active, timezone, avail_weekdays, avail_start_minute, avail_end_minute)
VALUES
  ('33333333-3333-3333-3333-333333333333',
   '22222222-2222-2222-2222-222222222222',
   '11111111-1111-1111-1111-111111111111',
   'loadtest-intro', 'Loadtest Intro', '', 30,
   true, 'Asia/Almaty', '{1,2,3,4,5}', 540, 1020)
ON CONFLICT (id) DO NOTHING;
```

> Replace the `organizations` column list with the actual required columns from the migration (and any membership row if `GET /api/book/:slug` needs the host to be an org member — the Step-4 200-check confirms whether the seed is sufficient).

- [ ] **Step 2: `load/capacity.js` — throughput/latency baseline (limits disabled)**

```js
import http from "k6/http"
import { check } from "k6"

const BASE = __ENV.LOAD_BASE_URL || "http://localhost:8090"
const SLUG = __ENV.LOAD_SLUG || "loadtest-intro"

export const options = {
  scenarios: {
    health: { executor: "ramping-vus", exec: "health", startVUs: 0,
      stages: [{ duration: "10s", target: 20 }, { duration: "20s", target: 20 }, { duration: "5s", target: 0 }] },
    book: { executor: "ramping-vus", exec: "book", startVUs: 0,
      stages: [{ duration: "10s", target: 20 }, { duration: "20s", target: 20 }, { duration: "5s", target: 0 }] },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    "http_req_duration{scenario:health}": ["p(95)<200"],
    "http_req_duration{scenario:book}": ["p(95)<500"],
  },
}

export function health() {
  const r = http.get(`${BASE}/api/health`)
  check(r, { "health 200": (res) => res.status === 200 })
}

export function book() {
  const r = http.get(`${BASE}/api/book/${SLUG}`)
  check(r, { "book 200": (res) => res.status === 200 })
}
```

- [ ] **Step 3: `load/shedding.js` — rate-limiter sheds under overload (limits enabled)**

```js
import http from "k6/http"
import { check } from "k6"

const BASE = __ENV.LOAD_BASE_URL || "http://localhost:8090"
const SLUG = __ENV.LOAD_SLUG || "loadtest-intro"

// One VU bursting well past the 60/min book_get limit — expect 429s, never 5xx.
export const options = {
  scenarios: {
    burst: { executor: "constant-arrival-rate", rate: 30, timeUnit: "1s",
      duration: "20s", preAllocatedVUs: 20, maxVUs: 50 },
  },
}

let total = 0
let got429 = 0
let got5xx = 0

export default function () {
  const r = http.get(`${BASE}/api/book/${SLUG}`)
  total++
  if (r.status === 429) got429++
  if (r.status >= 500) got5xx++
  check(r, { "no 5xx": (res) => res.status < 500 })
}

export function handleSummary() {
  const ok429 = got429 > 0
  const noServerErr = got5xx === 0
  // Confirm the limiter shed (some 429s) and the server never 5xx'd.
  const health = http.get(`${BASE}/api/health`)
  const alive = health.status === 200
  const pass = ok429 && noServerErr && alive
  return {
    stdout:
      `\nshedding: total=${total} 429=${got429} 5xx=${got5xx} health_after=${health.status} => ${pass ? "PASS" : "FAIL"}\n`,
  }
}
```

> If `handleSummary` can't issue an http request in this k6 version, move the post-burst `/api/health` check into `run.sh` (curl) instead and have the script decide PASS/FAIL from the 429/5xx counts k6 prints. Keep the rule: 429s present, zero 5xx, health 200 after.

- [ ] **Step 4: `load/run.sh` — orchestrate stack + seed + k6**

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-load"
scenario="${1:-all}"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

bring_up() { # $1 = extra env (e.g. RATE_LIMIT_DISABLED=true)
  env $1 $compose up -d --build
  for i in $(seq 1 60); do
    curl -fsS http://localhost:8090/api/health >/dev/null 2>&1 && return 0
    sleep 3
  done
  echo "stack not healthy"; $compose logs --tail=50; exit 1
}
seed() {
  $compose exec -T postgres psql -U notify -d notify -v ON_ERROR_STOP=1 < load/seed.sql
  curl -fsS http://localhost:8090/api/book/loadtest-intro >/dev/null \
    || { echo "seed verify failed: /api/book/loadtest-intro not 200"; exit 1; }
}

run_capacity() {
  cleanup
  RATE_LIMIT_DISABLED=true bring_up "RATE_LIMIT_DISABLED=true"
  seed
  docker run --rm -i --network host -e LOAD_BASE_URL=http://localhost:8090 \
    -v "$PWD/load:/load" grafana/k6 run /load/capacity.js
}
run_shedding() {
  cleanup
  bring_up ""   # limits ON
  seed
  docker run --rm -i --network host -e LOAD_BASE_URL=http://localhost:8090 \
    -v "$PWD/load:/load" grafana/k6 run /load/shedding.js
}

case "$scenario" in
  capacity) run_capacity ;;
  shedding) run_shedding ;;
  all) run_capacity; run_shedding ;;
  *) echo "usage: run.sh [capacity|shedding|all]"; exit 2 ;;
esac
```

> Notes for the implementer: (a) confirm the compose backend reads `RATE_LIMIT_DISABLED` from the environment passed at `up` time (the e2e compose backend `environment:` block lists explicit vars — you may need to add `RATE_LIMIT_DISABLED: ${RATE_LIMIT_DISABLED:-}` to the backend service in a load override, OR use a `docker-compose.load.yml` override file rather than inline `env`); (b) the psql user/db (`notify`/`notify`) match `deploy/docker-compose.e2e.yml`; (c) `grafana/k6` Docker image avoids requiring a local k6 install; `--network host` lets it reach `localhost:8090` (on macOS use `host.docker.internal` if host networking is unavailable — note this in README).

- [ ] **Step 5: Make it runnable + chmod**

`chmod +x load/run.sh`. Add a Makefile target (optional):
```make
load: ## run the k6 load harness (capacity + shedding)
	bash load/run.sh all
```

- [ ] **Step 6: Run the harness, capture baseline**

Run (Docker, sandbox disabled): `bash load/run.sh all`
Expected: capacity scenario completes — record RPS + p50/p95/p99 for health and book; `http_req_failed` under threshold. shedding scenario shows 429s present, zero 5xx, health 200 after. If capacity exceeds the soft p95 thresholds, that is a finding — record the actual numbers (don't silently lower thresholds to pass; document reality).

- [ ] **Step 7: `load/README.md` — how to run + baseline numbers**

Document: prerequisites (Docker), `bash load/run.sh [capacity|shedding|all]`, what each scenario does, the `RATE_LIMIT_DISABLED` non-prod note, and a "Baseline (YYYY-MM-DD, <machine>)" section with the observed numbers from Step 6. Note it is on-demand and intentionally not a blocking CI gate.

- [ ] **Step 8: Commit**

```bash
git status --porcelain   # stage exactly the load/ files (+ Makefile if changed)
git add load/ Makefile
git commit -m "$(cat <<'EOF'
test(ws7b): k6 load harness — capacity baseline + rate-limit shedding

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- `RATE_LIMIT_DISABLED` config + non-prod guard + `app.go` wrapper (no middleware-signature change) → Task 1. ✓
- k6 harness with capacity (limits off) + shedding (limits on) scenarios → Task 2 Steps 2-3. ✓
- Deterministic SQL seed with fixed slug; self-verified by the 200-check → Task 2 Steps 1, 4. ✓
- On-demand runner, non-blocking, baseline documented → Task 2 Steps 4-7. ✓
- Out of scope (POST/auth under load, blocking gate, distributed gen, profiling) → not in plan. ✓

**Placeholder scan:** The config change, wrapper, k6 scripts, and run.sh are complete. Two items are explicitly flagged for implementer confirmation against ground truth rather than guessed: the `organizations` required columns in `seed.sql` (verified by the 200-check) and how the compose backend receives `RATE_LIMIT_DISABLED` (inline env vs a `docker-compose.load.yml` override) — these depend on the exact migration/compose details and the run fails loudly if wrong, so they are discovery-confirmed, not deferred work.

**Type consistency:** `rateLimit(max int, window time.Duration, prefix string) fiber.Handler` is defined once and used at all 5 call sites with the same (max, window, prefix) triples as the originals. `RateLimitDisabled` is the config field referenced by the wrapper. The k6 scripts and `run.sh` share `LOAD_BASE_URL`/`LOAD_SLUG` and the fixed slug `loadtest-intro` consistently with `seed.sql`.

**Execution note:** Task 1 is Docker-free (config/build/test); Task 2 requires Docker. If the environment cannot run Docker, Task 2 cannot be executed/verified there — flag and run where Docker is available.
