# Security Policy

## Supported Versions

Security fixes are applied to the latest code on the `main` branch. Deploy from current `main` or the latest release tag.

| Version | Supported          |
| ------- | ------------------ |
| latest `main` | :white_check_mark: |
| older commits | :x:                |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report privately by:

1. Using [GitHub Security Advisories](https://github.com/luckyrogue/lead-cat/security/advisories/new) (preferred), or
2. Contacting the repository owner through GitHub.

Include:

- Description of the issue and potential impact
- Steps to reproduce (proof-of-concept if available)
- Affected components (backend, mini-app, admin, deploy)

We aim to acknowledge reports within **72 hours** and will coordinate disclosure once a fix is available.

## Sensitive data

This project handles Telegram auth, web sessions, Google Calendar integration, and SMTP. Never share `BOT_TOKEN`, `JWT_SECRET`, `MASTER_ENCRYPTION_KEY`, OAuth secrets, or service-account JSON in issues or PRs.
