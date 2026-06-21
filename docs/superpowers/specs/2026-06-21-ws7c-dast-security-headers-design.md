# WS7c — DAST baseline + security headers (design)

## Context: WS7 hardening, sub-project c of 3 (final)

WS7 = accessibility (7a, done) + load/perf (7b, done) + this. 7c adds a passive DAST
(OWASP ZAP baseline) harness as the team's web-security scanner, and closes the
response-header gaps it surfaces. Active/full penetration testing is **out of scope** —
intrusive, specialized, and largely redundant with existing controls (govulncheck for
deps, golangci-lint, per-IP rate limiting, the scoped CSRF guard, the earlier
X-Real-IP/metrics security fixes).

## Problem

There is no web-security scanner in the project, and the response-header posture is
uneven:
- **Frontend nginx is mostly hardened already** (verified): admin/landing/mini-app all
  send `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`,
  `Permissions-Policy`; admin/landing send `X-Frame-Options: DENY`; mini-app sends
  `Content-Security-Policy: frame-ancestors https://*.telegram.org` (correctly frameable
  by Telegram, not the others). Session/CSRF cookies already use `Secure`+`HttpOnly`(session)+`SameSite=Lax`.
- **The backend Fiber API sends NO security headers** — its middleware chain is
  recover/requestid/context/prometheus/logger/cors only. API responses lack `nosniff`,
  anti-framing, and `Referrer-Policy`, and auth responses are not marked non-cacheable.
- **HSTS (`Strict-Transport-Security`) is absent everywhere** — relevant in production
  behind TLS.

## Goal

1. A passive **ZAP baseline** scan harness (on-demand, over the compose stack) that
   crawls the front ends, reports missing-header / info-leak findings, and supports a
   baseline allowlist for accepted items — the reusable DAST tool.
2. Close the concrete header gaps the scan confirms (fix-first): a backend security-
   headers middleware, and `Strict-Transport-Security` on the nginx front ends — while
   **preserving the mini-app's Telegram frameability**.

## Design

### 1. Backend security-headers middleware (Fiber)

Add `middleware.SecurityHeaders()` applied early in the `app.Use(...)` chain in `app.go`
(after recover/requestid, before routes). It sets on every API response:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY` (the JSON API is never framed) — and/or
  `Content-Security-Policy: frame-ancestors 'none'`
- `Referrer-Policy: strict-origin-when-cross-origin`
- In production only (`cfg.IsProduction()`), `Strict-Transport-Security: max-age=63072000; includeSubDomains`.

Auth responses additionally get `Cache-Control: no-store` so tokens/sessions are never
cached — applied in the auth handlers (or the middleware can set `Cache-Control: no-store`
for `/api/auth/*` paths). The middleware takes the `isProduction` flag (or reads it from
a small config field passed at construction) so HSTS is prod-gated.

### 2. HSTS on the nginx front ends

Add `Strict-Transport-Security` to the shared `deploy/nginx/security-headers.conf`
(`add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;`).
Browsers ignore HSTS over plain HTTP, so it is safe to always-send; in production behind
TLS it takes effect. This covers admin, landing, AND mini-app (all include the shared
file). No change to the existing `X-Frame-Options`/CSP split (mini-app stays frameable).

### 3. Content-Security-Policy (content directives) — out of scope, allowlisted

A full content CSP (`default-src`/`script-src`/`style-src`) on the SPAs risks breaking
inline/external assets and needs careful per-app tuning + testing. It is **out of scope
for 7c** and recorded as an accepted ZAP baseline finding (allowlisted) with a follow-up
note, rather than added blindly. (The anti-framing `frame-ancestors` protection is
already present via X-Frame-Options / the mini-app CSP.)

### 4. ZAP baseline harness (`security/` dir, on-demand)

Mirrors the `e2e/`/`load/` pattern:
- `security/run.sh`: brings up the compose stack (reusing `deploy/docker-compose.e2e.yml`),
  waits for readiness, runs the OWASP ZAP **baseline** scan (`zaproxy/zap-stable`
  `zap-baseline.py`, passive only — no active attacks) against the admin (`:8090`),
  landing (`:8091`), and mini-app (`:8092`) base URLs, writes an HTML/MD report under
  `security/report/`, and tears the stack down. Non-zero ZAP exit (WARN/FAIL) is reported
  but does NOT gate (on-demand).
- `security/zap-baseline.conf`: the rule allowlist — items consciously accepted (e.g. the
  absent content-CSP on the SPAs) are set to `IGNORE` with a comment, so the scan's
  signal is the *unexpected* findings. New genuine findings show as WARN/FAIL.
- `security/README.md`: how to run (`bash security/run.sh`), the Docker prerequisite, the
  accepted-findings rationale, and the observed baseline result.

### Out of scope (7c)

- Active/intrusive penetration testing (ZAP active scan, fuzzing, exploit attempts).
- A full content `Content-Security-Policy` for the SPAs (allowlisted; follow-up).
- A blocking CI security gate (on-demand harness; ZAP baselines are noisy as hard gates).
- Auth/session-flow DAST (would need authenticated ZAP contexts) — the passive baseline
  over the public/app surfaces is the right-sized launch check.

## Error handling / fallbacks

- `run.sh` fails loudly if the stack isn't healthy before scanning.
- HSTS is always-send but inert over HTTP, and the backend variant is prod-gated — no
  effect on local/dev/e2e over plain HTTP.
- The `SecurityHeaders` middleware only adds response headers; it must not alter status,
  body, or existing handler behavior.

## Testing / verification

- A unit test for `middleware.SecurityHeaders()` asserting the headers are present on a
  response (and HSTS only when the prod flag is set). `go build ./... && go vet` + the
  existing suites stay green.
- A manual `bash security/run.sh` run: ZAP baseline completes; the report shows the
  expected reduced finding set (header gaps closed; only allowlisted items remain). The
  observed result + the accepted-findings list are recorded in `security/README.md`.
- On-demand and Docker-dependent; not part of the blocking CI suite.
