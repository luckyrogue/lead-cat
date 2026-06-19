# WS5-rl — Rate-Limit Public Endpoints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Redis-backed per-IP rate limiter (429 on excess) on the public booking endpoints + the magic-link request.

**Architecture:** A small custom Fiber middleware (`INCR`+`EXPIRE` fixed window over the existing `*redis.Client`, XFF-aware client IP, fail-open) applied to three public routes with named-constant limits.

**Tech Stack:** Go 1.26 (Fiber, go-redis v9), miniredis (test-only).

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-19-ws5-rl-rate-limit-public-endpoints-design.md`.
- No code comments in new Go files. depguard: middleware lives in `internal/delivery/http/middleware`.
- **Fail-open** on Redis error (return `c.Next()`); **429 + `Retry-After`** on excess.
- gofmt; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green.
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference (verified):**
- `NewApp(cfg, store, cipher, rdb *redis.Client, tg, log *zap.Logger, services)` builds `app := fiber.New(...)`; `rdb` + `log` are in scope for wiring.
- Routes: `web.Post("/magic/request", api.WebMagicRequest)` (line 84, under the `web := app.Group("/api/auth/web")` group → full path `/api/auth/web/magic/request`); `app.Get("/api/book/:slug", api.PublicBooking)` (147); `app.Post("/api/book/:slug", api.PublicBookingSubmit)` (148).
- `fiber.Config` sets no `ProxyHeader` → key on `X-Forwarded-For` first hop, fallback `c.IP()`.
- go-redis v9: `rdb.Incr(ctx, key).Result() (int64, error)`, `rdb.Expire(ctx, key, d).Err()`.
- Neither miniredis nor a testcontainers-redis module is in go.mod yet (testcontainers-go postgres is). Add `github.com/alicebob/miniredis/v2` as a test dependency.

---

### Task 1: `RateLimit` middleware + test + wiring

**Files:**
- Create: `apps/backend/internal/delivery/http/middleware/ratelimit.go`
- Test: `apps/backend/internal/delivery/http/middleware/ratelimit_test.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — apply to the 3 routes
- Modify: `apps/backend/go.mod` / `go.sum` — add miniredis (test dep) via `go get`

**Interfaces:**
- Produces: `middleware.RateLimit(rdb *redis.Client, log *zap.Logger, max int, window time.Duration, prefix string) fiber.Handler`.

- [ ] **Step 1: Add the test dependency** — `cd apps/backend && env -u GOROOT go get github.com/alicebob/miniredis/v2@latest`. (Adds to go.mod/go.sum.)

- [ ] **Step 2: Write the failing test** — `ratelimit_test.go` (`package middleware_test` or `middleware`):
```go
func newRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func appWith(h fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Get("/x", h, func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func reqIP(app *fiber.App, ip string) int {
	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("X-Forwarded-For", ip)
	resp, _ := app.Test(r)
	return resp.StatusCode
}

func TestRateLimit_AllowsThenBlocks(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 3, time.Minute, "t"))
	for i := 0; i < 3; i++ {
		if code := reqIP(app, "1.1.1.1"); code != 200 {
			t.Fatalf("req %d: want 200 got %d", i, code)
		}
	}
	if code := reqIP(app, "1.1.1.1"); code != 429 {
		t.Fatalf("4th: want 429 got %d", code)
	}
}

func TestRateLimit_IndependentIPs(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t"))
	if reqIP(app, "1.1.1.1") != 200 || reqIP(app, "2.2.2.2") != 200 {
		t.Fatal("distinct IPs must each be allowed once")
	}
	if reqIP(app, "1.1.1.1") != 429 {
		t.Fatal("second hit from same IP must be 429")
	}
}

func TestRateLimit_FailOpenOnRedisError(t *testing.T) {
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close() // redis now dead -> Incr errors -> fail-open
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t"))
	if code := reqIP(app, "1.1.1.1"); code != 200 {
		t.Fatalf("fail-open: want 200 got %d", code)
	}
}
```

- [ ] **Step 3: Run; expect FAIL** — `env -u GOROOT go test ./internal/delivery/http/middleware/ -run TestRateLimit -v`

- [ ] **Step 4: Implement** — `ratelimit.go`:
```go
package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func clientIP(c *fiber.Ctx) string {
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return c.IP()
}

func RateLimit(rdb *redis.Client, log *zap.Logger, max int, window time.Duration, prefix string) fiber.Handler {
	windowSecs := int64(window.Seconds())
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		bucket := time.Now().Unix() / windowSecs
		key := fmt.Sprintf("ratelimit:%s:%s:%d", prefix, clientIP(c), bucket)
		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			if log != nil {
				log.Warn("ratelimit_redis_error", zap.String("prefix", prefix), zap.Error(err))
			}
			return c.Next()
		}
		if n == 1 {
			_ = rdb.Expire(ctx, key, window).Err()
		}
		if n > int64(max) {
			c.Set("Retry-After", strconv.FormatInt(windowSecs, 10))
			return fiber.NewError(fiber.StatusTooManyRequests, "rate_limited")
		}
		return c.Next()
	}
}
```

- [ ] **Step 5: Run; expect PASS** — `env -u GOROOT go test ./internal/delivery/http/middleware/ -run TestRateLimit -v`

- [ ] **Step 6: Wire `app.go`** — apply the limiter to the three routes (constants inline or as package vars near registration):
```go
app.Get("/api/book/:slug", middleware.RateLimit(rdb, log, 60, time.Minute, "book_get"), api.PublicBooking)
app.Post("/api/book/:slug", middleware.RateLimit(rdb, log, 10, time.Hour, "book_post"), api.PublicBookingSubmit)
```
and for the magic-link route on the `web` group:
```go
web.Post("/magic/request", middleware.RateLimit(rdb, log, 5, 15*time.Minute, "magic"), api.WebMagicRequest)
```
(Confirm `time` is imported in app.go — it is. `log` is the `*zap.Logger` param.)

- [ ] **Step 7: Build/vet/lint + full test** — `env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. All green.

- [ ] **Step 8: gofmt + commit**
```bash
gofmt -w internal/delivery/http/middleware/ratelimit.go internal/delivery/http/middleware/ratelimit_test.go internal/delivery/http/app.go
git add apps/backend/internal/delivery/http/middleware/ratelimit.go apps/backend/internal/delivery/http/middleware/ratelimit_test.go apps/backend/internal/delivery/http/app.go apps/backend/go.mod apps/backend/go.sum
git commit -m "feat(security): rate-limit public booking + magic-link endpoints (redis fixed-window)"
```

---

### Task 2: Whole-slice verification

**Files:** none

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green; the 3 RateLimit tests pass (miniredis, no Docker).
- [ ] **Step 2: Wiring check (documented)** — confirm the three routes (`GET`/`POST /api/book/:slug`, `POST /api/auth/web/magic/request`) each carry a `middleware.RateLimit(...)` handler before their final handler in `app.go`, with the limits 60/min, 10/hour, 5/15min respectively.
- [ ] **Step 3: Tree clean** — verify HEAD; `git status` no stray staged files; go.mod/go.sum changes are only the miniredis test dep.

---

## Notes for the executor

- **Fail-open is intentional:** a Redis error must `return c.Next()` (don't block traffic).
- **IP key:** XFF first hop (prod proxy sets it), fallback `c.IP()` — localized to the limiter; does NOT change global `c.IP()`/logging. (Trusted-proxy allowlist is a deferred follow-up; XFF is spoofable — acceptable for an abuse deterrent.)
- **miniredis** is a TEST-only dep (give a genuine `*redis.Client` pointed at `mr.Addr()`); the middleware code uses the real go-redis client.
- **Deferred:** env-configurable limits, trusted-proxy allowlist, sliding-window, limits on authed endpoints. E2E coverage (WS4) is the next, separate slice.
```
