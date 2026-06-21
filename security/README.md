# OWASP ZAP Baseline DAST Harness

Passive-only DAST scan of the full app stack (admin / landing / mini-app) using
[OWASP ZAP](https://www.zaproxy.org/) baseline. **On-demand — not a CI gate.**

## Prerequisites

- Docker 29+ (`docker --version`)
- All three app images buildable (same as `e2e/run.sh`)

## Running

```bash
bash security/run.sh
# or via Make:
make dast
```

The script:
1. Spins up the compose stack (`deploy/docker-compose.e2e.yml`, project `leadcat-dast`)
2. Waits for `/api/health` to respond
3. Runs `zap-baseline.py` (passive only — no active attacks) against each app
4. Writes HTML + Markdown reports to `security/report/`
5. Tears down the stack on exit (trap)

ZAP exits non-zero on WARN/FAIL findings; the script captures the signal and reports
it but **does not gate** the run — the exit code of `run.sh` is always 0 after
reporting.

## Observed Baseline (2026-06-21)

| App      | FAIL-NEW | WARN-NEW | IGNORE | PASS |
|----------|----------|----------|--------|------|
| admin    | 0        | 0        | 5      | 62   |
| landing  | 0        | 0        | 5      | 62   |
| mini-app | 0        | 0        | 7      | 60   |

All three apps exit ZAP with **0 unexpected findings** after triage.

Headers confirmed present (PASS) by ZAP:
- `X-Content-Type-Options: nosniff` (10021)
- `Anti-clickjacking Header` / X-Frame-Options or frame-ancestors CSP (10020)
- `Strict-Transport-Security` (10035) — added by this task
- `Permissions-Policy` (10063)

## Allowlist Rationale (`zap-baseline.conf`)

| Rule ID | Rule Name | Apps | Rationale |
|---------|-----------|------|-----------|
| 10038 | Content Security Policy Not Set | admin, landing | SPA content-CSP is a tracked follow-up. Anti-framing is already covered by X-Frame-Options (admin/landing) and frame-ancestors CSP (mini-app). |
| 10036 | Server leaks version via Server header | admin, mini-app | nginx static-asset responses include the default nginx version string. `server_tokens off` would suppress it; low-risk informational leak. Accepted; tracked as a follow-up nginx hardening item. |
| 10049 | Storable and Cacheable Content | all | Static assets (images, fonts, manifests, JS bundles) are intentionally cached. This is correct nginx behavior for SPA builds. |
| 10109 | Modern Web Application | all | Informational — ZAP notes it cannot fully spider a JS SPA. Not a vulnerability. |
| 90004 | Cross-Origin-Embedder-Policy Missing | all | COEP is required only for SharedArrayBuffer / cross-origin isolation. This app does not use SharedArrayBuffer; COEP is unnecessary. |
| 10010 | Cookie No HttpOnly Flag | landing | The `leadcat_locale` cookie stores the user's UI language preference and must be readable by JavaScript for locale switching. It carries no session or auth data; adding HttpOnly would break the locale feature. |
| 10017 | Cross-Domain JS File Inclusion | mini-app | Telegram Mini Apps load Telegram's official SDK from telegram.org CDN. This is architecturally required by the Telegram platform. |
| 10055 | CSP Failure to Define Directive | mini-app | The mini-app's CSP is intentionally frame-ancestors-only (`deploy/nginx/security-headers-miniapp.conf`) to permit Telegram iframe embedding. A full content-CSP is a tracked follow-up (same as 10038). |
| 90003 | Sub Resource Integrity Missing | mini-app | The Telegram SDK is loaded from Telegram's CDN. Pinning SRI would break on Telegram SDK updates outside our control. |

## Report Artifacts

`security/report/` is gitignored. Reports are generated fresh on each run.
Sample report filenames: `zap-admin.html`, `zap-landing.html`, `zap-mini-app.html`
(plus `.md` equivalents).

## Follow-up Items (not in scope for WS7c)

- **nginx `server_tokens off`** — suppress nginx version from `Server` header on
  static asset responses (rule 10036).
- **Full content-CSP** on all three apps (rule 10038 / 10055) — requires JS app
  inventory and nonce/hash strategy; tracked separately.
