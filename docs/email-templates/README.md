# Lead Cat email templates

Standalone, email-client-safe HTML templates (table layout, inline styles, bulletproof CTAs, hidden preheader, unsubscribe). **No PostHog dependency** — render them with any Liquid-capable sender, or drop the rendered HTML into the backend SMTP sender (`internal/infrastructure/email/smtp`).

The reminder email is also ported to Go and wired into the backend — see [Production wiring](#production-wiring).

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

## Deliverability & client hardening
These templates are built to land in the inbox and render across the awkward clients:

- **multipart/alternative** — always send a `text/plain` part alongside the HTML. The `.txt` files are the plain-text bodies; spam filters penalize HTML-only mail and some clients show the text part. The Go sender does this via `SendMultipart` (below); for other senders, attach `welcome.txt` / `meeting-reminder.txt` as the text alternative.
- **Outlook-hardened CTAs** — the buttons carry a VML `<v:roundrect>` inside an `<!--[if mso]>` block so Word-rendering Outlook (desktop) shows a real rounded, padded button; every other client uses the styled `<a>` (wrapped in `<!--[if !mso]>` so Outlook ignores it — no double button).
- **List-Unsubscribe** — set the `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click` headers (RFC 8058) to the same URL as the in-body unsubscribe link. Gmail/Apple Mail surface a native unsubscribe affordance and reward it for reputation. The visible footer link is **not** a substitute. The Go sender sets these when given an unsubscribe URL.
- **Dark mode** — `color-scheme: light` + `supported-color-schemes: light` meta tags pin the design to its light palette so clients don't force-invert the cream/coral into muddy colors. (The layout is light-only by intent.)

Still on you (infra, not template): **SPF, DKIM, DMARC** on the sending domain — without them even perfect HTML lands in spam.

## Production wiring
The meeting reminder is the one path wired into the backend today. The Go port lives in `internal/platform/reminder_scheduler/email.go` (kept in sync with `meeting-reminder.html` / `.txt`) and renders both the HTML and plain-text parts in the recipient's language + timezone.

- Gated behind `REMINDER_EMAIL_ENABLED=true` (off by default); sends only to recipients with a known email.
- Delivered via `smtp.Sender.SendMultipart(ctx, to, subject, text, html, listUnsubscribe)` — multipart + RFC 2047-encoded subject + List-Unsubscribe headers.
- `smtp.Sender.Send(ctx, to, subject, html)` remains for the HTML-only web-auth magic-link path.

The Outlook VML for the Go template is injected as a trusted `template.HTML` value, because `html/template` strips HTML comments (and would otherwise drop the `<!--[if mso]>` conditionals).

The `welcome.*` templates are not wired to the backend yet — they're ready for whichever sender (PostHog or SMTP) sends onboarding mail.

## If you later want these in PostHog
Re-run the `designing-email-templates` skill after authorizing the PostHog MCP. Map variables to `person.properties.*` (e.g. `language` → `person.properties.language`, `first_name` → `person.properties.first_name`) and use the built-in `{{ unsubscribe_url }}`. The HTML here can seed an `html`-type block, or it can be rebuilt as native Unlayer blocks for visual editing.
