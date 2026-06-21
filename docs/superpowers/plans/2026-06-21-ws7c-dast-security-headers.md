# WS7c — DAST baseline + security headers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a backend security-headers middleware + HSTS on the nginx front ends, and a passive OWASP ZAP baseline scan harness (on-demand) that confirms the header posture and regression-checks it.

**Architecture:** A Fiber `SecurityHeaders(prod bool)` middleware sets nosniff / anti-framing / referrer (+ prod HSTS, + `no-store` on `/api/auth/*`) on every API response, wired early in `app.go`. `Strict-Transport-Security` is added to the shared nginx `security-headers.conf` (always-send, inert over HTTP; the existing X-Frame-Options/CSP split and mini-app frameability are untouched). A `security/` harness runs `zap-baseline.py` (passive) over the compose stack with a baseline allowlist, producing a report. On-demand, not a CI gate.

**Tech Stack:** Go (Fiber middleware), nginx config, OWASP ZAP (`zaproxy/zap-stable` Docker image), docker-compose stack.

## Global Constraints

- **Headers only, no behavior change:** `SecurityHeaders` must not alter status, body, or handler logic — only add response headers.
- **HSTS gating:** backend HSTS only when `cfg.IsProduction()`; nginx HSTS is `always` (browsers ignore it over plain HTTP, so it's safe in dev/e2e).
- **Preserve mini-app frameability:** do NOT add `X-Frame-Options`/restrictive frame CSP to the shared nginx file or anything that breaks mini-app embedding in Telegram. Only HSTS goes in the shared file. (Backend `X-Frame-Options: DENY` is fine — the JSON API is never framed.)
- **ZAP baseline is passive only** (no active attacks), on-demand, **not** wired into the PR CI gate. Consciously-accepted findings (e.g. absent SPA content-CSP) go in an allowlist with comments; the scan's signal is *unexpected* findings.
- **Docker prerequisite** for Task 2 (compose stack + ZAP image); run docker commands with the sandbox disabled.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Backend `SecurityHeaders` middleware

Docker-free; verified by a unit test + build.

**Files:**
- Create: `apps/backend/internal/delivery/http/middleware/security_headers.go`
- Create: `apps/backend/internal/delivery/http/middleware/security_headers_test.go`
- Modify: `apps/backend/internal/delivery/http/app.go` (one `app.Use` line)

**Interfaces:**
- Produces: `func SecurityHeaders(prod bool) fiber.Handler`.

- [ ] **Step 1: Write the failing test `security_headers_test.go`**

```go
package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
)

func appWithSec(prod bool) *fiber.App {
	app := fiber.New()
	app.Use(middleware.SecurityHeaders(prod))
	app.Get("/api/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Post("/api/auth/web/magic/request", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func TestSecurityHeaders_Common(t *testing.T) {
	resp, err := appWithSec(false).Test(httptest.NewRequest("GET", "/api/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("nosniff = %q", got)
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if resp.Header.Get("Referrer-Policy") == "" {
		t.Error("Referrer-Policy missing")
	}
	if resp.Header.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must be absent in non-prod")
	}
}

func TestSecurityHeaders_ProdHSTS(t *testing.T) {
	resp, _ := appWithSec(true).Test(httptest.NewRequest("GET", "/api/x", nil))
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("HSTS must be present in prod")
	}
}

func TestSecurityHeaders_AuthNoStore(t *testing.T) {
	resp, _ := appWithSec(false).Test(httptest.NewRequest("POST", "/api/auth/web/magic/request", nil))
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("auth Cache-Control = %q, want no-store", got)
	}
	// non-auth path should NOT be forced no-store
	resp2, _ := appWithSec(false).Test(httptest.NewRequest("GET", "/api/x", nil))
	if resp2.Header.Get("Cache-Control") == "no-store" {
		t.Error("non-auth path should not be no-store")
	}
}
```

Run: `cd apps/backend && go test ./internal/delivery/http/middleware/ -run TestSecurityHeaders 2>&1 | head` — Expected: FAIL (`SecurityHeaders` undefined).

- [ ] **Step 2: Implement `security_headers.go`**

```go
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders sets baseline security response headers on every API response.
// HSTS is emitted only in production (it requires TLS to be meaningful); auth
// endpoints are additionally marked non-cacheable.
func SecurityHeaders(prod bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if prod {
			c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		if strings.HasPrefix(c.Path(), "/api/auth/") {
			c.Set("Cache-Control", "no-store")
		}
		return c.Next()
	}
}
```

- [ ] **Step 3: Wire it in `app.go`**

After `app.Use(requestid.New())` (and before the routes), add:
```go
	app.Use(middleware.SecurityHeaders(cfg.IsProduction()))
```
(`cfg.IsProduction()` exists; `middleware` is already imported.)

- [ ] **Step 4: Run tests + build + vet**

Run: `cd apps/backend && go test ./internal/delivery/http/middleware/ && go build ./... && go vet ./internal/delivery/http/...`
Expected: the 3 new tests PASS, existing middleware tests still pass, build + vet clean.

- [ ] **Step 5: Commit**

```bash
git status --porcelain
git add apps/backend/internal/delivery/http/middleware/security_headers.go apps/backend/internal/delivery/http/middleware/security_headers_test.go apps/backend/internal/delivery/http/app.go
git commit -m "$(cat <<'EOF'
feat(security): backend security-headers middleware (nosniff/frame/referrer/HSTS/no-store)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: nginx HSTS + ZAP baseline harness + run

Docker-dependent.

**Files:**
- Modify: `deploy/nginx/security-headers.conf` (add HSTS)
- Create: `security/run.sh`, `security/zap-baseline.conf`, `security/README.md`
- Modify: `Makefile` (optional `dast` target)

**Interfaces:**
- Consumes (Task 1): the backend headers (so the ZAP report reflects the closed gaps).

- [ ] **Step 1: Add HSTS to the shared nginx headers**

Append to `deploy/nginx/security-headers.conf` (do NOT touch the X-Frame-Options/CSP split):
```
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;
```
(This file is included by admin, landing, AND mini-app — HSTS is safe for all; the mini-app's frame-ancestors CSP and the admin/landing X-Frame-Options stay as they are.)

- [ ] **Step 2: Create `security/zap-baseline.conf` (allowlist skeleton)**

ZAP rule config: `<ruleId>\tIGNORE\t<comment>` lines for consciously-accepted findings.
Start with the known accepted item (SPA content-CSP absent) and let the run reveal exact
IDs to confirm:
```
# OWASP ZAP baseline rule config. Format: <ruleId>\t(IGNORE|WARN|FAIL)\t<comment>
# 10038 = Content Security Policy (Header Not Set). SPA content-CSP is a tracked
# follow-up (frame protection already via X-Frame-Options / mini-app frame-ancestors).
10038	IGNORE	SPA content-CSP deferred (anti-framing already covered)
```
> Adjust/extend during the run: any genuinely-accepted warning gets an IGNORE line with a comment; unexpected findings stay WARN/FAIL and must be triaged (or fixed).

- [ ] **Step 3: Create `security/run.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-dast"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
$compose up -d --build
for i in $(seq 1 60); do
  curl -fsS http://localhost:8090/api/health >/dev/null 2>&1 && break
  sleep 3
  [ "$i" = 60 ] && { echo "stack not healthy"; $compose logs --tail=50; exit 1; }
done

mkdir -p security/report
net="--network host"
host="localhost"
if [ "$(uname)" = "Darwin" ]; then net=""; host="host.docker.internal"; fi

status=0
for app in "admin:8090" "landing:8091" "mini-app:8092"; do
  name="${app%%:*}"; port="${app##*:}"
  echo "[dast] scanning $name (http://$host:$port)"
  docker run --rm $net -v "$PWD/security:/zap/wrk:rw" zaproxy/zap-stable \
    zap-baseline.py -t "http://$host:$port" \
    -c zap-baseline.conf -r "report/zap-$name.html" -w "report/zap-$name.md" \
    || status=$?   # ZAP exits non-zero on WARN/FAIL; report, don't gate
done
echo "[dast] done (combined ZAP exit signal: $status). Reports in security/report/."
```

> Notes: `zaproxy/zap-stable` `zap-baseline.py` reads `-c` config and writes reports into the mounted `/zap/wrk`. On macOS use `host.docker.internal` (handled above). The script reports ZAP's WARN/FAIL exit but does not fail the run (on-demand). If `zap-baseline.py` can't resolve the host, confirm the networking flag for this machine and note it in the README.

- [ ] **Step 4: Run the harness, triage findings**

`chmod +x security/run.sh` then run (Docker, sandbox disabled): `bash security/run.sh`
Expected: ZAP baseline completes for all three apps; reports land in `security/report/`. Review the WARN/FAIL list:
- Header gaps that Task 1 + Step 1 should have closed (nosniff/frame/referrer/HSTS) must NOT appear for the relevant surfaces — if they do, investigate (e.g. the backend `/api/*` via the app proxy vs direct).
- Consciously-accepted findings (SPA content-CSP, any benign info) → add an `IGNORE` line to `zap-baseline.conf` with a comment and re-run until the report is clean of *unexpected* findings.
- Any genuine new finding → triage; fix if cheap/in-scope, else record in README as known/accepted with rationale. Do NOT blanket-IGNORE to force a clean report.

- [ ] **Step 5: `security/README.md`**

Document: prerequisite (Docker), `bash security/run.sh`, that it's a passive baseline (no active attacks) and on-demand (not a CI gate), the allowlist rationale (each IGNORE'd rule + why), and the observed result (clean of unexpected findings as of the run date). Optional Makefile target:
```make
dast: ## run the OWASP ZAP baseline scan over the app stack
	bash security/run.sh
```

- [ ] **Step 6: Commit**

```bash
git status --porcelain
git add deploy/nginx/security-headers.conf security/ Makefile
git commit -m "$(cat <<'EOF'
test(ws7c): HSTS on nginx + OWASP ZAP baseline DAST harness

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- Backend `SecurityHeaders` middleware (nosniff/frame/referrer + prod HSTS + auth no-store), wired in app.go → Task 1. ✓
- HSTS on nginx shared config, mini-app frameability preserved (only HSTS added; X-Frame/CSP split untouched) → Task 2 Step 1 + constraint. ✓
- ZAP baseline passive harness with allowlist + report, on-demand non-gating → Task 2 Steps 2-5. ✓
- Content-CSP on SPA out of scope, allowlisted with comment → Task 2 Step 2. ✓
- Active pentest / CI gate / auth-flow DAST out of scope → not in plan. ✓

**Placeholder scan:** Middleware, test, app.go line, nginx line, run.sh, and the allowlist skeleton are complete. The ZAP rule-IDs to allowlist beyond 10038 are intentionally discovered at run time (ZAP output names each rule) — Step 4 says add IGNORE-with-comment for *accepted* findings and triage the rest, which is the honest shape of baseline triage, not a deferred TODO.

**Type consistency:** `SecurityHeaders(prod bool) fiber.Handler` defined in Task 1, called in app.go with `cfg.IsProduction()`, exercised by the test with `true`/`false`. The harness uses the same compose file/ports as the e2e/load harnesses (`:8090/:8091/:8092`) and a distinct project name `leadcat-dast`.

**Execution note:** Task 1 is Docker-free (test/build); Task 2 needs Docker (compose + ZAP image). If Docker is unavailable, Task 2 cannot be run/verified there — flag and run where Docker is available.
