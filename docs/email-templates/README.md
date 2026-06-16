# Lead Cat email templates

Standalone, email-client-safe HTML templates (table layout, inline styles, bulletproof CTAs, hidden preheader, unsubscribe). **No PostHog dependency** — render them with any Liquid-capable sender, or drop the rendered HTML into the backend SMTP sender (`internal/infrastructure/email/smtp` → `Send(to, subject, htmlBody)`).

| Email | HTML | Plain text |
|---|---|---|
| Welcome (day 0) | `welcome.html` | `welcome.txt` |
| Meeting reminder | `meeting-reminder.html` | `meeting-reminder.txt` |

## Brand
Cozy Lead Cat. Palette repeated literally throughout (no CSS vars in email):
cream `#FBF7F0` · page `#F1EADF` · coral accent `#E8714C` · sunny `#F6C453` · ink `#3A332E` · muted `#8A7F75`. Headlines `Georgia` serif, body `Arial` sans.

## Localization (ru / en / kk)
Each file is **one template, branched with Liquid** on a `language` variable (`ru` | `en` | `kk`); **default `ru`** (matches the app's default locale). The localized strings are defined once in a `{% case language %}` block at the top, then referenced in the markup — so there's one template per email, not three.

**How the language is chosen at send time:** set the `language` variable to the recipient's stored preference (the per-user `language` field from settings). Fallback chain: explicit pref → Telegram `language_code` → `ru`. If the sender doesn't pass `language`, everyone gets `ru`.

## Variables
All have `| default:` fallbacks, so a missing value never renders blank.

**welcome**
| Variable | Example | Notes |
|---|---|---|
| `language` | `ru` | `ru`\|`en`\|`kk`; default `ru` |
| `first_name` | `Mia` | default `there`/`друг`/`дос` |
| `app_url` | `https://t.me/lead_cat_bot/app` | Mini App deep link |
| `unsubscribe_url` | … | required for marketing sends |

**meeting-reminder**
| Variable | Example | Notes |
|---|---|---|
| `language` | `ru` | default `ru` |
| `first_name` | `Mia` | |
| `meeting_title` | `Design sync` | |
| `meeting_date` | `16.06.2026` | pre-formatted by the sender |
| `meeting_time` | `10:00–10:30` | pre-formatted |
| `meeting_tz` | `UTC+5` | optional; hidden if blank |
| `participants` | `mia@co.com, alex@co.com` | optional; row hidden if blank |
| `meet_link` | `https://meet.google.com/…` | optional; **Join** button hidden if blank |
| `unsubscribe_url` | … | |

Format `meeting_date`/`meeting_time`/`meeting_tz` on the sender side — mirror the backend's existing notification format (`02.01.2006`, `15:04`, `UTC±H` from `meeting_notifier/message.go`).

## Subjects
Also Liquid-branched (set on the send, not in the HTML body — see the comment at the top of each `.html`):
- **welcome** — ru `Добро пожаловать в Lead Cat 🐾` · en `Welcome to Lead Cat 🐾` · kk `Lead Cat-қа қош келдің 🐾`
- **reminder** — ru `Напоминание: «{{ meeting_title }}» в {{ meeting_time }}` · en `Reminder: "{{ meeting_title }}" at {{ meeting_time }}` · kk `Еске салу: «{{ meeting_title }}» {{ meeting_time }}`

## If you later want these in PostHog
Re-run the `designing-email-templates` skill after authorizing the PostHog MCP. Map variables to `person.properties.*` (e.g. `language` → `person.properties.language`, `first_name` → `person.properties.first_name`) and use the built-in `{{ unsubscribe_url }}`. The HTML here can seed an `html`-type block, or it can be rebuilt as native Unlayer blocks for visual editing.
