# SPA content Content-Security-Policy (design)

## Context

Follow-up to WS7c (DAST). The frontends already send most security headers (nosniff,
Referrer-Policy, Permissions-Policy, X-Frame-Options DENY on admin/landing, HSTS,
mini-app `frame-ancestors` for Telegram), and ZAP allowlisted the absent **content**
CSP (`default-src`/`script-src`/…) as a deferred item (rule 10038). This adds an
enforcing content CSP to close it.

CSP on a real SPA is risky — a wrong policy blocks legitimate scripts/styles and breaks
the app. So it is built **discovery-driven** (measure actual violations against the
running apps), and the per-app policy is verified by loading the apps in a browser
before it ships enforcing.

## What the apps actually load (from source)

- **Ackee analytics**: `<script src="${ACKEE_BASE_PATH}/tracker.js">` — `ACKEE_BASE_PATH`
  is same-origin (`/ackee/`, nginx-proxied to analytics.rysdavletov.org). External
  `<script src>`, same-origin → covered by `script-src 'self'`; its POSTs are same-origin
  → `connect-src 'self'`.
- **Telegram SDK (mini-app only)**: `<script src="https://telegram.org/js/telegram-web-app.js">`
  — external → mini-app `script-src` needs `https://telegram.org`.
- **Fonts**: `@fontsource-variable/inter` is bundled → served same-origin (`font-src 'self'`).
- **Scripts**: the apps are served as static client builds (`COPY build/client` via nginx,
  no React-Router server), so hydration is via the external JS bundle (`script-src 'self'`),
  not an inline hydration script — to be confirmed by discovery.
- **Styles**: Tailwind v4 + React inline `style=` attributes → `style-src` needs
  `'unsafe-inline'` (style injection is low-risk and standard to allow).

## Goal

Ship an enforcing `Content-Security-Policy` on admin, landing, and mini-app that the
discovery run proves produces zero CSP violations while keeping `script-src` strict
(no `'unsafe-inline'` for scripts — hash any inline script instead), and preserves the
mini-app's Telegram frameability.

## Design

### Candidate policy (per app, in nginx)

- **admin / landing**:
  `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; object-src 'none'`
- **mini-app** (extends, keeping Telegram framing):
  `default-src 'self'; script-src 'self' https://telegram.org; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' https://telegram.org; frame-ancestors https://web.telegram.org https://*.telegram.org; base-uri 'self'; object-src 'none'`

`style-src 'unsafe-inline'` is an accepted residual (documented). `script-src` stays
strict; if discovery finds an inline script, add its `'sha256-…'` hash (NOT
`'unsafe-inline'`) — and if that hash proves build-unstable (changes per build), that is
a recorded finding to resolve (e.g. externalize the script) rather than weaken to
`'unsafe-inline'`.

### Discovery-then-enforce methodology

1. Add the candidate policy as **`Content-Security-Policy-Report-Only`** first.
2. Bring up the stack; with Playwright, load each app's key pages (landing `/`, `/en`;
   admin authed: dashboard/meetings/booking-config + `/book/:slug`; mini-app via the
   `stubTelegramWebApp` seam: home/meetings/profile) and collect CSP violation events
   (`page.on('console')` / the `securitypolicyviolation` event / Report-Only reports).
3. Tighten the policy until zero violations (add a hash for any inline script; never
   broaden `script-src` to `'unsafe-inline'`).
4. **Flip Report-Only → enforcing** (`Content-Security-Policy`), re-run, confirm zero
   violations AND the apps still render/function (no blocked scripts).

### Where it lives (nginx)

- **admin / landing** (`spa.conf`): add the `Content-Security-Policy` header (a new
  `add_header` or a shared `csp-web.conf` include alongside the existing
  `security-headers.conf` + `X-Frame-Options DENY`).
- **mini-app** (`security-headers-miniapp.conf`): replace the current frame-ancestors-only
  CSP line with the full mini-app policy above (one CSP header, frame-ancestors retained)
  — a response should carry a single coherent CSP.

## Out of scope

- Backend API CSP beyond the `frame-ancestors 'none'` already covered by WS7c's
  `X-Frame-Options: DENY` (a JSON API doesn't render content; a content CSP is moot).
- CSP violation **reporting endpoint** (`report-uri`/`report-to` collection) — the
  discovery uses browser-side violation capture; a production report collector is a
  separate follow-up.
- Nonce-based CSP (impractical for statically-served SPA HTML; hashes are the strict
  mechanism here).

## Error handling / fallbacks

- The Report-Only phase cannot break the apps (browsers don't enforce it) — it only
  surfaces what the enforcing policy would block, so the enforcing flip is safe.
- mini-app must remain frameable by Telegram — the `frame-ancestors https://*.telegram.org`
  directive is preserved verbatim in its merged policy.

## Testing / verification

- Playwright: with each app's enforcing CSP active, load the key pages and assert
  **zero** `securitypolicyviolation`/CSP-blocked-resource events, and that the page
  rendered (a known element is visible) — proving the policy doesn't break the app.
  Reuse the WS7a a11y harness surfaces + the `stubTelegramWebApp` seam for mini-app.
- A short note records the final per-app policy and any accepted residual
  (`style-src 'unsafe-inline'`, any inline-script hash) and where it's set.
- Docker-dependent, on-demand verification (the enforcing headers themselves ship in the
  nginx configs and take effect in every deploy).
