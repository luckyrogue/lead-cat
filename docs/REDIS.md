# Redis

## Uses

- **asynq** — `scenario:run` tasks
- **scheduler lock** — key `leadcat:scheduler:leader` TTL 90s

## Env

`REDIS_URL=redis://host:6379/0`

## Monitoring

- Failed tasks: asynq retry logs + `scenario_runs.status=failed`
- Memory: alert if Redis > 80% (Dokploy metrics)

## Failures

| Symptom              | Action                                              |
| -------------------- | --------------------------------------------------- |
| health redis down    | check Redis service, `REDIS_URL`                    |
| duplicate cron fires | verify single scheduler leader; scale app carefully |
| stuck runs           | inspect asynq pending; re-run Test run              |
