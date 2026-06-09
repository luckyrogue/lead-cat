# Lead Cat — Setup Guide

Lead Cat is a single-purpose Google Meet meetings-management Telegram Mini App.
Employees schedule, view, and manage meetings entirely inside the Mini App; no web
login or external scheduler is required.

**Operator setup** runs inside the Mini App: **Profile → Admin → Настройка workspace**
(`/profile/admin/setup`), backed by `/api/miniapp/admin/*`. Platform curl bootstrap is retired (HTTP 410).

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

| Config key         | Description                                                          |
| ------------------ | -------------------------------------------------------------------- |
| `google_sa_json`   | Full service account JSON (the file from Google Cloud)               |
| `calendar_subject` | Delegated user email the service account impersonates                |
| `calendar_id`      | Google Calendar id to create events on (usually the same user email) |

### Mini App admin (recommended)

1. Open the Mini App as a user with `role=admin`.
2. Go to **Profile → Admin → Настройка workspace**.
3. Paste the service account JSON, set subject and calendar id, tap **Сохранить**, then **Проверить**.

Backend routes: `PATCH /api/miniapp/admin/integrations`, `POST /api/miniapp/admin/integrations/verify`.

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

Registration happens entirely via the bot (not HTTP):

1. User opens the Telegram bot and sends `/start`.
2. Bot asks for full name (ФИО), then corporate email.
3. A `bot_users` row is created immediately (no email OTP).
4. User opens the Mini App from the bot menu; `POST /api/auth/miniapp` issues a JWT.

Unregistered users who open the Mini App first see a "register in the bot" screen.

---

## Step 5 — Admin bootstrap

Designate one or more Telegram users as admins by setting their numeric Telegram IDs
in the environment:

```
BOT_ADMIN_TELEGRAM_IDS=123456789,987654321
```

Comma-separated, no spaces. These users gain **Profile → Admin** and the **Настройка workspace**
screen for Google Calendar, chat linking, and member sync.

To find a Telegram user's numeric ID, have them message the bot — their id appears in
the `GET /api/miniapp/me` response field `telegram_id` once registered.

---

## Appendix — Retired platform bootstrap

`/api/workspaces/*` and platform auth (`/api/auth/email/*`, passkey, OAuth) return **410 Gone**.
All operator setup uses Mini App admin (`/api/miniapp/admin/*`) from Telegram.

See also: [API.md](API.md), [ARCHITECTURE.md](ARCHITECTURE.md), [MEETINGS.md](MEETINGS.md).
