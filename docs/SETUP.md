# Lead Cat — Setup Guide

Lead Cat is a single-purpose Google Meet meetings-management Telegram Mini App.
Employees schedule, view, and manage meetings entirely inside the Mini App; no web
login or external scheduler is required.

**Setup today** uses curl against the platform REST API with a platform JWT (alpha
interim). **Target:** an Admin overlay inside the Mini App replaces all curl steps —
see the [TMA setup-replacement spec](superpowers/specs/2026-06-05-tma-setup-replacement-design.md).

---

## Step 1 — Create the Telegram bot

1. Open Telegram and message **@BotFather**.
2. Send `/newbot` and follow the prompts to choose a name and username.
3. Copy the token BotFather gives you — this is your `BOT_TOKEN`.
4. Set it in your environment / Dokploy config:
   ```
   BOT_TOKEN=<your-bot-token>
   ```

For full BotFather options (commands menu, description, photo) see [BOTFATHER.md](BOTFATHER.md).

---

## Step 2 — Configure Google Calendar integration

Lead Cat creates Google Meet links via a service account with domain-wide delegation.
You need three values:

| Config key | Description |
| --- | --- |
| `google_sa_json` | Full service account JSON (the file from Google Cloud) |
| `calendar_subject` | Delegated user email the service account impersonates |
| `calendar_id` | Google Calendar id to create events on (usually the same user email) |

### Target (not yet shipped)

Once the Admin overlay ships, you will paste the service account JSON and calendar
settings directly inside the Mini App under **Profile → Admin → Integrations** —
the backend route will be `PATCH /api/tma/admin/integrations`, no curl required.
See [TMA setup-replacement spec](superpowers/specs/2026-06-05-tma-setup-replacement-design.md).

### Interim alpha — curl (deprecated, will be removed)

> **Deprecated.** This path requires a platform JWT and direct REST access. It is the
> current only option while the TMA Admin overlay is in development.

Obtain a platform JWT (email OTP, passkey, or GitHub/GitLab OAuth — see [AUTH.md](AUTH.md)),
then apply the integration and verify it:

```bash
# PATCH integrations (deprecated alpha path)
curl -X PATCH https://<host>/api/workspaces/<workspace-id>/integrations \
  -H "Authorization: Bearer <PLATFORM_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "google_sa_json": "<SERVICE_ACCOUNT_JSON_PLACEHOLDER>",
    "calendar_subject": "<DELEGATED_USER_EMAIL_PLACEHOLDER>",
    "calendar_id": "<CALENDAR_ID_PLACEHOLDER>"
  }'

# Verify the integration
curl -X POST https://<host>/api/workspaces/<workspace-id>/integrations/verify \
  -H "Authorization: Bearer <PLATFORM_JWT>"
```

A `200` with `"ok": true` on verify confirms Google Calendar access is working. Any
error contains a human-readable `message` field.

---

## Step 3 — Employee directory

The employee directory drives the colleague-schedule view and participant search inside
the Mini App. It is an embedded CSV file:

```
backend/internal/platform/employeedir/employees.csv
```

**Format (header row required):**
```
id,full_name,department,position,telegram_username
```

Edit the CSV, then rebuild and redeploy the backend. No migration or API call needed —
the file is read at startup.

---

## Step 4 — Users self-register

Users do not create platform accounts. Registration happens entirely via the bot:

1. User opens the Telegram bot and sends `/start`.
2. The Mini App shows a **registration screen** (first launch only) asking for name
   confirmation.
3. On submit, the user is registered as a `bot_user` and can immediately access the
   meetings features.

There is no invite flow or email required — anyone who can find and start the bot can
register.

---

## Step 5 — Admin bootstrap

Designate one or more Telegram users as admins by setting their numeric Telegram IDs
in the environment:

```
BOT_ADMIN_TELEGRAM_IDS=123456789,987654321
```

Comma-separated, no spaces. These users gain the **Admin** section in **Profile** tab
inside the Mini App, which is where all future setup surfaces will live (per the
[setup-replacement spec](superpowers/specs/2026-06-05-tma-setup-replacement-design.md)).

To find a Telegram user's numeric ID, have them message the bot — their id appears in
the `GET /api/tma/me` response field `telegram_id` once registered.

---

## Appendix — Deprecated: alpha setup (curl)

The curl-based setup path (Step 2 above) uses platform JWT authentication against
`/api/workspaces/*`. This is an **interim alpha mechanism** and will be replaced by
TMA-authenticated `/api/tma/admin/*` routes so operators never need a platform account
or terminal access.

Deprecation schedule follows the phased rollout in the
[TMA setup-replacement spec](superpowers/specs/2026-06-05-tma-setup-replacement-design.md):

| Phase | Route deprecated for human use |
| --- | --- |
| Phase 1 (meetings unblock) | `PATCH /api/workspaces/:id/integrations` |
| Phase 2 (legacy notify-bot ops) | `…/chat/*`, `…/members/*` |
| Phase 3 (auto-tab ops) | `…/auto/*` read/toggle |

After all phases ship, platform setup routes will require a script-only token or be
removed from public deployments. Do not build new tooling against `/api/workspaces/*`
for setup — use `/api/tma/admin/*` when it ships.

See also: [API.md](API.md), [ARCHITECTURE.md](ARCHITECTURE.md), [MEETINGS.md](MEETINGS.md).
