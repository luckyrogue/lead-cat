# Lead Cat email templates

Standalone, email-client-safe HTML templates (table layout, inline styles, bulletproof CTAs, hidden preheader). **No PostHog dependency** — render them with any Liquid-capable sender, or use the Go ports in `apps/backend/internal/platform/emailtemplates/`.

| Email | HTML | Plain text | Go renderer |
|---|---|---|---|
| Welcome (day 0) | `welcome.html` | `welcome.txt` | `RenderWelcome` |
| Magic link sign-in | `magic-link.html` | `magic-link.txt` | `RenderMagicLink` |
| Organization invite | `org-invite.html` | `org-invite.txt` | `RenderOrgInvite` |
| Meeting reminder | `meeting-reminder.html` | `meeting-reminder.txt` | `RenderReminder` |

**Sync rule:** when you change a `.html` / `.txt` design file, update the matching Go port in `emailtemplates/` so production mail stays in sync. The backend does **not** read these files at runtime.

## Brand
Cozy Lead Cat. Palette repeated literally throughout (no CSS vars in email):
cream `#FBF7F0` · page `#F1EADF` · coral accent `#E8714C` · sunny `#F6C453` · ink `#3A332E` · muted `#8A7F75`. Headlines `Georgia` serif, body `Arial` sans.

## Localization (ru / en / kk)
Each file is **one template, branched with Liquid** on a `language` variable (`ru` | `en` | `kk`); **default `ru`**. The Go renderers mirror the same strings via `NormalizeLang`.

**How the language is chosen at send time:** set `language` to the recipient's stored preference (`platform_users.language`). Fallback chain: explicit request param (magic link) → DB lookup by email → `ru`.

## Variables

**welcome** — `language`, `first_name`, `app_url`, `unsubscribe_url`

**magic-link** — `language`, `sign_in_url`, `expires_minutes`, `first_name` (optional)

**org-invite** — `language`, `org_name`, `role_label`, `login_url`, `inviter_name` (optional)

**meeting-reminder** — `language`, `first_name`, `meeting_title`, `meeting_date`, `meeting_time`, `meeting_tz`, `participants`, `meet_link`, `unsubscribe_url`

Format date/time on the sender side (`02.01.2006`, `15:04`, `UTC±H`).

## Subjects
Also Liquid-branched (see comment at the top of each `.html`):
- **welcome** — ru `Добро пожаловать в Lead Cat 🐾` · en `Welcome to Lead Cat 🐾` · kk `Lead Cat-қа қош келдің 🐾`
- **magic-link** — ru `Войти в Lead Cat` · en `Sign in to Lead Cat` · kk `Lead Cat-қа кіру`
- **org-invite** — ru `Приглашение в {org}` · en `Invitation to {org}` · kk `{org} ұйымына шақыру`
- **reminder** — ru `Напоминание: «{title}» в {time}` · en `Reminder: "{title}" at {time}` · kk `Еске салу: «{title}» {time}`

## Deliverability & client hardening
- **multipart/alternative** — always send text + HTML via `EmailSender.SendMultipart`.
- **Outlook-hardened CTAs** — VML `<v:roundrect>` injected as trusted `template.HTML` in Go.
- **List-Unsubscribe** — set for welcome and reminder (`WEBAPP_URL` profile link); transactional magic-link and invite omit it.
- **Dark mode** — `color-scheme: light` meta tags pin the light palette.

Still on you: **SPF, DKIM, DMARC** on the sending domain.

## Production wiring
All four emails render in `internal/platform/emailtemplates/` and send via `smtp.Sender.SendMultipart`.

| Email | Trigger | List-Unsubscribe |
|---|---|---|
| Magic link | `POST /api/auth/web/magic/request` | — |
| Org invite | `InviteToOrg` (admin) | — |
| Welcome | `CreateOrganizationForOwner` (onboarding) | `WEBAPP_URL` |
| Meeting reminder | `reminder_scheduler` tick | `WEBAPP_URL` |

Meeting reminder email is gated behind `REMINDER_EMAIL_ENABLED=true` (off by default); SMTP alone is not enough — enable this flag for email reminders.

## If you later want these in PostHog
Re-run the `designing-email-templates` skill after authorizing the PostHog MCP. Map variables to `person.properties.*` and use the built-in `{{ unsubscribe_url }}`.
