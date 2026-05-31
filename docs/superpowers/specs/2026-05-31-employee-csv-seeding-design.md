# Employee Directory CSV Seeding — Design (§1.2 / §9.4)

**Status:** approved, ready for implementation plan.
**Spec source (ТЗ):** `docs/NEW-FEATURES.md` §1.2 (line 37–39), §9.4 (line 838–863). Feature status: `docs/MEETINGS.md`.

## Goal

On server startup, sync an **embedded** `employees.csv` into the `employees` table of **every Google-configured workspace**, so the meetings flows that search the directory — participant add (`/edit`), `/schedule`, `/checker`, and the Mini App `GET /api/workspaces/:id/employees` — work against real data instead of an empty table.

Per ТЗ §9.4 the CSV is the source of truth for the directory; columns are `full_name`, `email`, `department`; search is case-insensitive substring over `full_name`/`email` (already implemented in `SearchEmployeesGlobal`).

## Decisions (locked during brainstorming)

1. **Seed target = workspaces with Google configured.** Auto-target each workspace where `google_sa_json_enc IS NOT NULL`. No new "which workspace" env var. This matches where meetings actually work (a workspace without Google creds returns 400 on meeting create), so the directory seeds exactly where it is usable. Trade-off: 0 workspaces seeded until Google is configured — acceptable.
2. **CSV source = `//go:embed`.** The CSV is compiled into the binary and lives in the repo (PR-reviewed). Updating the directory = edit CSV → rebuild → redeploy. No `EMPLOYEES_CSV_PATH` override (YAGNI).
3. **Sync mode = full sync with a zero-row safety guard.** Per workspace the DB mirrors the CSV: delete rows whose email is no longer in the CSV, upsert the rest. This keeps the directory accurate (departed employees disappear from search/schedule/checker). **Guard:** if the parsed CSV has 0 records, skip the entire sync and log a warning — a truncated/empty CSV must never wipe the directory.
4. **Seeding is best-effort (non-fatal).** A directory-sync error logs at Error and the server continues to boot. A directory glitch should not take down notify-bot, auth, or the scenario engine.

## Codebase facts (verified)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`.
- `employees` table (`migrations/20260530120000_meetings.sql`): `id, workspace_id (NOT NULL, FK→workspaces ON DELETE CASCADE), full_name, email, dept (DEFAULT ''), has_telegram (DEFAULT false), created_at`, `UNIQUE (workspace_id, email)`.
- `has_telegram` is written **only** by `CreateEmployee` and read by `ListEmployees`/`SearchEmployeesGlobal`; never maintained dynamically (the CSV has no telegram column). The upsert therefore leaves `has_telegram` untouched on existing rows.
- `meeting_participants.email` is a plain column (no FK to `employees`), so deleting an employee row never cascades into meetings/participants — full-sync deletes are referentially safe.
- "Google configured" marker: `workspaces.google_sa_json_enc BYTEA` (`migrations/20260530130000_meetings_google.sql`); configured ⇔ `IS NOT NULL`.
- Existing repo (`internal/infrastructure/persistence/postgres/employee_repo.go`): `ListEmployees(ctx, workspaceID)`, `SearchEmployeesGlobal(ctx, query)` (ILIKE substring, cap 20), `CreateEmployee(ctx, workspaceID, fullName, email, dept, hasTelegram)`. `Store` holds `pool` (pgxpool) and exposes a `*zap.Logger`.
- Bootstrap (`cmd/server/main.go`): `pool` → `runMigrations` (if `AUTO_MIGRATE`) → `store := postgres.New(pool, logger)` (≈ line 59) → services/cipher/queue → `telegram.NewMultiHandler(...)` (≈ line 108) → HTTP listen + bot polling. The seed call slots in **after** `store` is created and **before** the server starts serving.
- Conventions: build/test/lint from repo root via `make test && make lint && make build` (lint includes a gofmt check); run Go directly as `env -u GOROOT go ...` from `backend/`. Pure logic is unit-tested; repo/IO/wiring is build-verified (no DB harness in the postgres package). Structured zap logging via `internal/platform/observability/log`; messages are stable snake_case + fields, no secrets.

## Architecture

Dependencies point inward; the seeder is a small `platform` orchestrator over a `domain`-free pure parser and two new repo queries.

### Component 1 — `internal/platform/employeedir` (new package)

Files:
- `employees.csv` — the embedded directory. Header `full_name,email,department`. A small starter set lands in the repo; real data is edited here.
- `employeedir.go` — `//go:embed employees.csv` `var csvData []byte`, plus `Seed`.
- `parse.go` — pure parser.
- `parse_test.go` — unit tests.

**`Record`**
```go
type Record struct {
    FullName string
    Email    string // normalized lower-case
    Dept     string
}
```

**`Parse(data []byte) ([]Record, error)`** — pure.
- `encoding/csv` reader; `FieldsPerRecord = -1` tolerated but validate the header.
- Require the header row to be exactly `full_name,email,department` (trim/lowercase comparison); error `invalid header` otherwise.
- For each data row: trim all fields; lower-case email; **skip** rows with empty email (log-free; the caller logs counts) and rows that are entirely blank.
- Returns records in file order. No DB, no I/O beyond the passed bytes. Handles `\r\n` (encoding/csv does this natively).

**`Seed(ctx context.Context, store *postgres.Store, log *zap.Logger)`** — orchestration.
1. `records, err := Parse(csvData)`; on error log `employee_csv_parse_failed` (Error) and return.
2. **Guard:** `if len(records) == 0 { log Warn "employee_csv_empty"; return }`.
3. `wsIDs, err := store.ListWorkspacesWithGoogle(ctx)`; on error log `employee_seed_failed` (Error) and return.
4. For each `wsID`: `added, updated, deleted, err := store.SyncEmployees(ctx, wsID, records)`. On per-workspace error, log `employee_sync_failed` (Error) with `workspace_id` and continue to the next workspace (one bad workspace must not skip the rest).
5. Log `employees_synced` (Info) per workspace with `workspace_id, added, updated, deleted` and a final `employee_seed_done` (Info) with `workspaces` count.

### Component 2 — repo queries (in `employee_repo.go`, build-verified)

**`ListWorkspacesWithGoogle(ctx) ([]uuid.UUID, error)`**
```sql
SELECT id FROM workspaces WHERE google_sa_json_enc IS NOT NULL ORDER BY id
```

**`SyncEmployees(ctx, workspaceID uuid.UUID, seeds []EmployeeSeed) (added, updated, deleted int, err error)`** where `EmployeeSeed{FullName, Email, Dept string}` is a small struct defined in the `postgres` package (see "Dependency direction").
- One `pgx` transaction (`pool.Begin` / `tx.Rollback` deferred / `tx.Commit`).
- Collect `emails := []string{...}` from seeds (already lower-cased).
- **Delete missing:** `DELETE FROM employees WHERE workspace_id=$1 AND email <> ALL($2)` → `deleted = rows affected`.
- **Upsert present:** for each record,
  ```sql
  INSERT INTO employees (workspace_id, full_name, email, dept)
  VALUES ($1,$2,$3,$4)
  ON CONFLICT (workspace_id, email) DO UPDATE
    SET full_name = EXCLUDED.full_name, dept = EXCLUDED.dept
  RETURNING (xmax = 0) AS inserted
  ```
  `xmax = 0` ⇒ inserted (count `added`), else `updated`. `has_telegram` is omitted on both INSERT (defaults false) and the DO UPDATE set list (preserved on existing rows).

**Dependency direction (avoid an import cycle).** `employeedir.Seed` imports `postgres` (to call the store). If `postgres.SyncEmployees` imported `employeedir.Record`, that is a cycle. Resolution: define the row shape the repo consumes as a small struct in the `postgres` package — `EmployeeSeed{FullName, Email, Dept string}` — and have `Seed` map `[]employeedir.Record → []postgres.EmployeeSeed`. The pure `Record` type stays in `employeedir`; the repo stays free of the platform package.

### Component 3 — wiring (`cmd/server/main.go`)

After `store := postgres.New(pool, logger)` and before the HTTP/bot start:
```go
employeedir.Seed(ctx, store, logger)
```
Synchronous (fast: a handful of workspaces × small CSV), best-effort, before serving so the directory is ready on first request. No new config, no env.

## Data flow

```
boot → migrations → store
     → employeedir.Seed
         Parse(embedded csv) ──(0 rows)──▶ warn + return
              │ records
              ▼
         store.ListWorkspacesWithGoogle()
              │ wsIDs
              ▼ (per workspace, own tx)
         store.SyncEmployees(wsID, records)
              ├─ DELETE emails not in CSV
              └─ UPSERT each record (full_name, dept)   [has_telegram untouched]
              ▼
         log employees_synced{added,updated,deleted}
     → HTTP listen + bot polling
```

## Error handling

| Failure | Handling |
| --- | --- |
| CSV parse error / bad header | Log `employee_csv_parse_failed` (Error); skip seeding; server continues. |
| 0 records parsed | Log `employee_csv_empty` (Warn); **skip sync entirely** (guard); server continues. |
| `ListWorkspacesWithGoogle` error | Log `employee_seed_failed` (Error); skip; server continues. |
| `SyncEmployees` error for one workspace | Log `employee_sync_failed` (Error, `workspace_id`); continue to next workspace. |
| No Google-configured workspaces | `wsIDs` empty; loop is a no-op; `employee_seed_done{workspaces:0}` (Info). |

No secrets in logs (emails are corporate directory data already stored in plaintext in the table; counts + workspace_id only at Info, full email never logged).

## Testing

- **Pure parser (`parse_test.go`, full unit coverage):** valid rows; header validation (wrong/missing header → error); blank lines skipped; row with empty email skipped; email lower-cased; surrounding whitespace trimmed; CRLF line endings; an all-blank file → 0 records (drives the guard).
- **Repo (`SyncEmployees`, `ListWorkspacesWithGoogle`):** build-verified only (project convention — no DB harness in the postgres package). The SQL is reviewed for the `xmax = 0` add/update split and the `<> ALL($2)` delete.
- **Seeder + wiring:** build-verified. Manual smoke (optional): configure Google on a workspace locally, boot, confirm `employees_synced` log and `GET /api/workspaces/:id/employees`.
- Full gate before merge: `make test && make lint && make build` from repo root.

## Docs to update

- `docs/MEETINGS.md` — add an "Employee directory CSV seeding (done)" line: embedded `internal/platform/employeedir/employees.csv`, full-sync into Google-configured workspaces on boot, rebuild-to-update.
- `docs/REQUIREMENTS.md` — flip the "planned" CSV line: the directory is **embedded** (no env var); to change it, edit the CSV and redeploy.

## Out of scope (YAGNI)

- Hot-reload without restart (ТЗ explicitly leaves this to the developer; restart/redeploy is the trigger).
- Bot/admin UI to manage the list (ТЗ §8 future).
- `EMPLOYEES_CSV_PATH` runtime override.
- Per-employee Telegram linkage / `has_telegram` maintenance from registration.
- Export, Active Directory integration (ТЗ open questions, not requested).
