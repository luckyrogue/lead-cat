# Redis

Redis serves two purposes in Lead Cat: backing the **asynq** job queue for meeting
notifications, and providing **scheduler leader locks** so only one replica runs
each background tick across a scaled deployment.

---

## Connection

Set `REDIS_URL` to the Redis instance address:

```
REDIS_URL=redis://user:pass@host:6379/0
```

For TLS-enabled instances (e.g. managed Redis with in-transit encryption), use the
`rediss://` scheme:

```
REDIS_URL=rediss://user:pass@host:6379/0
```

The URL is parsed by `asynq.ParseRedisURI` and also used directly by `go-redis` for
the scheduler lock clients.

---

## Queues — asynq job kinds

The meeting notification worker processes the following asynq task types:

| Task type             | Triggered when                                    |
| --------------------- | ------------------------------------------------- |
| `meeting:created`     | A new meeting is created; notifies participants   |
| `meeting:updated`     | Meeting details (time, link, name) are edited     |
| `meeting:cancelled`   | A meeting is cancelled; notifies participants     |
| `participant:added`   | A participant is added to an existing meeting     |
| `participant:removed` | A participant is removed from an existing meeting |

All tasks are enqueued with `MaxRetry(5)`. Payloads carry `workspace_id` and
`meeting_id` (plus `email` for participant tasks).

Reminder dispatch is handled in-process by the `reminder_scheduler` — it sends
Telegram DMs directly and does not go through the asynq queue.

---

## Scheduler leader locks

The reminder scheduler acquires a Redis `SET NX` lock (TTL 90 s, renewed every
tick) before doing any work. This ensures only one replica dispatches reminders at
a time in a multi-instance deployment.

| Lock key                   | Purpose                                            |
| -------------------------- | -------------------------------------------------- |
| `leadcat:reminders:leader` | Meeting reminder dispatch tick — runs every minute |

(An additional `leadcat:scheduler:leader` lock exists for the legacy cron-trigger
scheduler; that machinery is part of the deprecated alpha layer and not used by
the meetings product.)

If a lock cannot be acquired (another replica holds it, or Redis is unreachable),
the tick is silently skipped — no double-sends occur.

---

## Operations

- **Queue depth** — inspect pending / retry / failed tasks via `asynqmon` or the
  asynq CLI. See `OPERATIONS.md` for connection details.
- **Reminder not firing** — verify that exactly one replica holds
  `leadcat:reminders:leader` and that `REDIS_URL` is reachable from the app
  container.
- **Memory** — alert if Redis memory exceeds ~80 % of the configured `maxmemory`
  (Dokploy metrics). Meeting notification payloads are small; memory growth is
  mainly asynq internal structures.
- **Further guidance** — see `docs/OPERATIONS.md` for runbook steps, replica
  scaling notes, and health-check endpoints.
