# Lead Cat — docs

**Lead Cat** is a single-purpose **Google Meet meetings-management Telegram Mini App**.
Employees register via the bot's `/start`; inside the Mini App they create, edit, and
delete meetings, get conflict warnings, find common free time, view colleague schedules,
and receive Telegram reminders. Google Meet links are created through a corporate Google
service account. Admins configure the integration inside the Mini App. Stack: Go (Fiber,
asynq, Postgres) + React Telegram Mini App.

Start with the **ТЗ**: [NEW-FEATURES.md](NEW-FEATURES.md).

| Doc                                    | Purpose (one line)                                         | Audience       |
| -------------------------------------- | ---------------------------------------------------------- | -------------- |
| [NEW-FEATURES.md](NEW-FEATURES.md)     | the canonical technical spec (ТЗ)                          | Dev/Product    |
| [REQUIREMENTS.md](REQUIREMENTS.md)     | product overview, actors, feature set, stack               | Dev/Product    |
| [ARCHITECTURE.md](ARCHITECTURE.md)     | backend layers, meeting domain, identity, request flow     | Dev            |
| [API.md](API.md)                       | HTTP API (Mini App stack; platform deprecated appendix)    | Dev            |
| [AUTH.md](AUTH.md)                     | Telegram Mini App auth (platform auth deprecated appendix) | Dev            |
| [DEPLOY-DOKPLOY.md](DEPLOY-DOKPLOY.md) | Dokploy deploy + env vars                                  | Operator       |
| [SETUP.md](SETUP.md)                   | bot/Google/employee-CSV/admin setup path                   | Operator/Admin |
| [BOTFATHER.md](BOTFATHER.md)           | bot token + /start + menu commands                         | Operator       |
| [OPERATIONS.md](OPERATIONS.md)         | logs, health, metrics, backups, smoke checklist            | Operator       |
| [REDIS.md](REDIS.md)                   | asynq reminders + scheduler lock                           | Dev/Operator   |
| [MEETINGS.md](MEETINGS.md)             | engineering status of the meetings feature                 | Dev            |
| [DESIGN-CATS.md](DESIGN-CATS.md)       | cat design system                                          | Dev/Design     |
| [LOCAL_DEV.md](LOCAL_DEV.md)           | local development workflow                                 | Dev            |
| [MIGRATIONS.md](MIGRATIONS.md)         | DB migration conventions                                   | Dev            |
