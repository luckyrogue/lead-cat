# Migrations (goose)

## Naming

`YYYYMMDDHHMMSS_description.sql` in `backend/migrations/`.

## Commands

```bash
cd backend
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate status
```

## Seed

`20260520000000_member_vcs_logins.sql` splits GitHub/GitLab logins per team member (no env seed).

## Production

Set `AUTO_MIGRATE=true` so `lead-cat` runs goose on start, or run `go run ./cmd/migrate up` / `/app/migrate up` before deploy.
