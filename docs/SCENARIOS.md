# Scenarios (n8n-like)

## Definition JSON

```json
{
  "nodes": [
    { "id": "t1", "type": "trigger.cron", "parameters": { "hour": 18, "minute": 30, "weekdays": [1,2,3,4,5], "tz": "Asia/Almaty" } },
    { "id": "a1", "type": "action.telegram.message", "parameters": { "text": "Пора коммитить! 🐾" } },
    { "id": "a2", "type": "action.telegram.cat_photo", "parameters": {} }
  ],
  "edges": [
    { "source": "t1", "target": "a1" },
    { "source": "a1", "target": "a2" }
  ]
}
```

## Node types

| type | Description |
|------|-------------|
| `trigger.cron` | Schedule |
| `trigger.manual` | Test run button |
| `action.telegram.message` | Text to notify chat |
| `action.telegram.cat_photo` | Evil cat image |
| `action.vcs.commits_report` | Commits report to owner |

## Presets (create scenario in Mini App)

- **commits** — weekdays 18:30
- **meet** — Mon/Wed/Fri 10:15
- **commits_report** — weekdays 18:35

## Test run

`POST /api/workspaces/:id/scenarios/:sid/run` enqueues asynq job; see runs at `GET .../runs`.
