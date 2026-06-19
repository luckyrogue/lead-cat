# WS5-rl — Rate-Limit Public Endpoints (design)

**Date:** 2026-06-19
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, **WS5 (security)** — the rate-limiting item flagged when the public booking endpoint (slice 3-3) shipped without it.

## Goal

Protect the public, unauthenticated endpoints — the booking submit (creates calendar events)
and the magic-link request (sends emails) — from abuse, via a Redis-backed per-IP rate limiter
returning `429 Too Many Requests` when a limit is exceeded.

## Decisions (from brainstorming)

- **Custom Redis fixed-window** middleware (the HTTP layer already has `*redis.Client`; no new
  dependency). `INCR` + `EXPIRE` per `(prefix, ip, window-bucket)`.
- **Scope:** `POST /api/book/:slug` (strict), `GET /api/book/:slug` (lenient), and
  `POST /api/auth/web/magic/request` (email-spam vector).
- **Fail-open** on a Redis error (availability over strictness; logged).
- **IP source:** first hop of `X-Forwarded-For` (set by the prod proxy), fallback `c.IP()`;
  localized to the limiter. Trusted-proxy allowlist is a deferred follow-up (XFF spoofable).

## Background — verified current state

- `NewApp(cfg, store, cipher, rdb *redis.Client, tg, log, services)` (`internal/delivery/http/app.go:26`)
  — the Fiber app is constructed with a live `*redis.Client` (`rdb`); `handlers.API.RDB = rdb`.
- `fiber.New(fiber.Config{ ErrorHandler: … })` sets NO `ProxyHeader` — so `c.IP()` is the proxy's
  IP. The prod SPA containers reverse-proxy `/api/*`→backend (nginx) and set `X-Forwarded-For`.
- Public routes (no middleware): `GET /api/book/:slug` (148-ish), `POST /api/book/:slug`,
  `POST /api/auth/web/magic/request`, plus auth callbacks. No rate limiting exists today.
- Existing middleware lives in `internal/delivery/http/middleware/` (e.g. `RequestContext`,
  `web_auth`, `require_org_member`). New limiter middleware joins them.

## Design

### A. Limiter middleware

`internal/delivery/http/middleware/ratelimit.go`:
```go
func RateLimit(rdb *redis.Client, max int, window time.Duration, prefix string) fiber.Handler
```
- Key the client IP: `ip := clientIP(c)` where `clientIP` returns the first comma-split token of
  `c.Get("X-Forwarded-For")` (trimmed) if non-empty, else `c.IP()`.
- Fixed window bucket: `bucket := now.Unix() / int64(window.Seconds())`; `key :=
  "ratelimit:" + prefix + ":" + ip + ":" + strconv.FormatInt(bucket, 10)`.
- `n, err := rdb.Incr(ctx, key)`; on the first increment (`n == 1`) `rdb.Expire(ctx, key, window)`.
  - `err != nil` → **fail-open**: log `ratelimit_redis_error` (Warn) once + `return c.Next()`.
  - `n > int64(max)` → set `Retry-After` (seconds to bucket end) + `return fiber.NewError(429,
    "rate_limited")`.
  - else `return c.Next()`.
- `now` via `time.Now()`; ctx via `c.UserContext()`. No PII beyond the IP in logs (IP is
  operationally necessary; acceptable — or hash it if the project's logging rules forbid raw IP,
  the plan checks AGENTS.md and hashes if needed).

### B. Wiring (`app.go`)

Apply the middleware to the three routes with named-constant limits:
- `POST /api/book/:slug` → `RateLimit(rdb, 10, time.Hour, "book_post")`.
- `GET /api/book/:slug` → `RateLimit(rdb, 60, time.Minute, "book_get")`.
- `POST /api/auth/web/magic/request` → `RateLimit(rdb, 5, 15*time.Minute, "magic")`.
Constants declared near the route registration (or in the middleware package). Env-overridable
limits are deferred.

### C. Behavior

- Within a window, the (max+1)-th request from an IP → 429 with `Retry-After`. The next window
  resets. Distinct IPs are independent. Multi-instance correct (shared Redis).
- A Redis outage degrades to no limiting (fail-open) rather than blocking traffic.

## Testing / verification

- **Middleware unit test** (`ratelimit_test.go`): mount `RateLimit(rdb, 3, time.Minute, "t")` on
  a tiny Fiber app; N≤3 → 200, 4th → 429 with `Retry-After`; a fresh IP (different
  `X-Forwarded-For`) is independent; a Redis error → fail-open (200). Use **miniredis**
  (`github.com/alicebob/miniredis/v2`, in-memory Redis fake — runs in CI without Docker) if
  available/addable; else a testcontainers Redis (Docker, like the postgres tests). The plan
  picks based on go.mod. Use `app.Test(req)` with the `X-Forwarded-For` header to drive IPs;
  simulate a redis error by closing miniredis / pointing at a dead addr.
- **Wiring:** `go build/vet`, `golangci-lint`, `go test -race ./...` green; the three routes carry
  the limiter (assert by code inspection / an httptest hitting the booking POST limit if a redis
  is wired in the test app).
- No frontend change.

## Risks & mitigations

- **XFF spoofing** (no trusted-proxy allowlist) → a client could forge `X-Forwarded-For` to evade
  or poison another IP's limit. *Mitigation:* documented; a `TrustedProxies`/`ProxyHeader` +
  allowlist config is a follow-up. For an abuse-deterrent (not a security boundary) this is
  acceptable now.
- **Fail-open** means a Redis outage disables limiting. *Mitigation:* intentional (availability);
  Redis is already a hard dependency (asynq) so an outage is already degraded mode.
- **Fixed-window burst at boundaries** (2×max across a window edge). *Mitigation:* acceptable for
  abuse deterrence; sliding-window deferred.
- **New test dep (miniredis)** if chosen. *Mitigation:* test-only; or use testcontainers Redis to
  avoid a new dep — the plan decides from go.mod.

## Done criteria

- `middleware.RateLimit` (Redis fixed-window, XFF-aware IP, fail-open, 429+Retry-After).
- Applied to `POST /api/book/:slug` (10/h), `GET /api/book/:slug` (60/min),
  `POST /api/auth/web/magic/request` (5/15min).
- Unit test (allow→429→reset→independent-IP→fail-open); `-race` + lint green.
- Trusted-proxy allowlist, env-configurable limits, and limits on authed endpoints explicitly
  deferred. (E2E coverage — WS4 — is the next, separate slice.)
