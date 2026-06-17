# WS2a — Test Foundation: harness + domain + persistence (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, workstream 2 (backend test coverage), sub-phase **a of d**.
WS2 was split into 4 sub-phases because "test everything incl. adapters" is too large for one plan:
- **WS2a (this doc):** test harness + domain layer + Postgres repos. *Foundation — unblocks the rest.*
- WS2b: application `command`/`query` handlers (in-memory fakes).
- WS2c: platform orchestration packages (checker, meetingedit, scheduleview, botreg, meeting_notifier).
- WS2d: external adapters (Google Calendar HTTP, OAuth/OIDC, Telegram dispatch).

## Goal

Stand up the backend test harness and use it to verify the two riskiest untested areas — the domain recurrence/series math and the Postgres persistence layer (~1,300 LOC of SQL) — against a real, ephemeral Postgres. The harness must work identically locally and in CI with no manual setup beyond Docker being available.

## Background — verified current state (2026-06-17)

- No DB test infrastructure exists (no testcontainers/dockertest/sqlmock in `go.mod`).
- Repos are constructed as `postgres.New(pool *pgxpool.Pool, log *zap.Logger)`; 15 repo files, ~1,300 LOC. Some methods manage their **own** transactions internally (e.g. `CreateMeetingSeries` writes meetings + participants atomically).
- Migrations are **filesystem-based, not embedded**: both `cmd/migrate/main.go` and `cmd/server/main.go` call goose against the `"migrations"` directory (relative path). There are 20 migrations under `apps/backend/migrations/`.
- Existing tests (scheduler_agent, reminder_scheduler, meetingrecipients, email/smtp) use hand-written in-memory fakes — no DB. WS2a adds the first real-DB tests.
- CI runners and local dev both have Docker (CI builds images; local dev runs a compose Postgres via `make up`). The CI gate from WS1 already runs `go test ./...`.
- Domain package `internal/domain/meeting/` has pure functions: `recurrence.Occurrences`, `Recurrence.Valid`, `conflict.Overlaps`, `conflict.FreeSlots`, `naming.GenerateName`, `validate.Input.Validate`.

## Decisions (resolved during brainstorming)

1. **testcontainers-go** for the repo layer (ephemeral Postgres); in-memory fakes for the application layer (WS2b, not here). Chosen over a CI service container for identical local/CI behavior and ephemeral ports (sidesteps the local 5432 conflict with other projects).
2. **Isolation = truncate-all-tables before each test**, NOT transaction-rollback — because repo methods open their own transactions, which a per-test wrapping transaction would conflict with.
3. **One container per repo-test package** (via `TestMain`), migrations applied once at boot. Boot cost amortized across the package's tests.
4. **Migrations located via `runtime.Caller`**, not embedded — no production code changes. The harness computes the absolute path to `apps/backend/migrations` from the `pgtest` package's own source location. (Embedding migrations via `//go:embed` is a possible later improvement but is deferred: it would touch the two prod entrypoints, which is out of WS2a's "add tests, don't change behavior" intent.)
5. WS2a covers domain + repos only. Application/platform/adapter tests are WS2b–2d.

## Design

### 1. Test harness — `internal/testsupport/pgtest`

New package `apps/backend/internal/testsupport/pgtest` exposing:

- `func Start(ctx context.Context) (*DB, error)` — called from each repo-test package's `TestMain(m *testing.M)`. Boots a `postgres:16-alpine` container via testcontainers-go, waits for readiness (the testcontainers postgres module's wait strategy), opens a `*pgxpool.Pool`, applies all goose migrations once. Returns a `*DB` handle. Usage pattern: `TestMain` calls `Start`, stashes the `*DB` in a package var, runs `m.Run()`, then `db.Close()`.
- `func DockerAvailable() bool` — probes for a usable Docker endpoint. `TestMain` uses it to decide whether to run repo tests: if false, it logs a skip notice and exits 0 (so `go test ./...` degrades gracefully without Docker — CI has Docker, so the gate still enforces coverage there).
- `type DB struct { Pool *pgxpool.Pool; ... }` with:
  - `func (d *DB) Truncate(t *testing.T)` — truncates every application table (`TRUNCATE ... RESTART IDENTITY CASCADE`) so each test starts clean. Discovers table names from `information_schema` (excluding `goose_db_version`) so it stays correct as migrations evolve. Call at the top of each test.
  - `func (d *DB) Store(log *zap.Logger) *postgres.Store` — convenience constructor returning a repo bound to the pool.
  - `func (d *DB) Close()` — terminates the container.
- Migrations are applied by resolving the migrations dir absolutely: `filepath.Join(callerDir, "..", "..", "..", "migrations")` derived via `runtime.Caller`, then `goose.SetDialect("postgres")` + `goose.Up(sqlDB, migrationsDir)` against a `database/sql` handle opened on the container DSN (mirrors `cmd/migrate`).

The harness is the single source of DB-test setup; repo test files call it, never duplicate container/migration logic.

### 2. Domain tests (no DB)

`internal/domain/meeting/*_test.go` — pure unit tests:
- **recurrence_test.go:** `Occurrences` for `none`/`daily`/`weekly`/`custom`(weekday mask)/`monthly`; the `until` boundary (inclusive/exclusive as implemented); the 100-occurrence safety cap; custom-weekday with an empty/!valid mask; monthly month-end rollover.
- **conflict_test.go:** `Overlaps` truth table (touching edges, containment, disjoint); `FreeSlots` with no busy spans, fully-busy window, and `minDur` filtering out short gaps.
- **naming_test.go:** `GenerateName` format `[Dept] | [Type] | [Host] | [Date] | [Frequency]` across recurrence types.
- **validate_test.go:** `Input.Validate` — end-after-start, required fields, recurring requires `until`, custom requires weekdays; plus `Recurrence.Valid`.

### 3. Postgres repo tests (real DB via harness)

`internal/infrastructure/persistence/postgres/*_test.go` — each test gets a clean DB via `Truncate`:
- **meetings:** create → get round-trip (all fields incl. recurrence + meet link); update; cancel; list by org/scope window.
- **series:** `CreateMeetingSeries` writes N meetings + their participants atomically; a forced mid-batch failure leaves zero rows (transaction rollback).
- **participants:** add then remove; idempotent add (no duplicate).
- **orgs + members:** create org, add member with role, list members, remove.
- **bot_users:** upsert/get by telegram id; reminder-minutes + language + timezone persistence.
- **web sessions:** create, lookup by token, expiry honored.
- **audit log:** insert + list ordering.
- **reminder claim:** `TryClaimReminder(meeting, telegramID, offset)` returns true once, false on the second identical claim (idempotency that prevents double reminders).

Test data is built with small local helpers (e.g. `seedOrg`, `seedMeeting`) kept in the test files — no shared fixture framework (YAGNI).

### New dependency

`github.com/testcontainers/testcontainers-go` + its `modules/postgres`. Added to `apps/backend/go.mod`. CI already has Docker; the WS1 `go test ./...` gate will exercise these.

## Testing / verification

- `make test` (and `go test ./...`) green locally with Docker running.
- With Docker stopped, repo tests `t.Skip` cleanly (domain tests still run and pass).
- Confirm the harness truncate-isolation actually isolates: tests pass in any order and when run with `-count=2`.
- Confirm in CI (the WS1 gate runs `go test ./...` on the runner, which has Docker) that the new tests execute and pass — watch the run after push.

## Risks & mitigations

- **Container boot time** adds seconds to the repo-test packages. Mitigation: one container per package via `TestMain`, migrations applied once; domain tests stay DB-free and fast.
- **Docker required.** Mitigation: `TestMain` checks `DockerAvailable()` and exits 0 with a skip notice when the Docker socket is absent locally; CI always has Docker so coverage is still enforced where it counts. Domain tests have no `TestMain` dependency and always run.
- **Migration path resolution** via `runtime.Caller` is location-coupled. Mitigation: a single helper computes it; a self-check in `Start` fails loudly if the resolved dir has no `.sql` files.
- **Flaky readiness waits.** Mitigation: use the testcontainers postgres module's built-in wait strategy (waits for the "ready to accept connections" log + a successful ping), not a fixed sleep.

## Done criteria

- `internal/testsupport/pgtest` exists with `Start`/`Truncate`/`Store`/`Close`, resolving migrations via `runtime.Caller` and skipping cleanly without Docker.
- Domain tests cover the functions listed in §2; repo tests cover the surfaces in §3, each isolated via truncate.
- `go test ./...` passes locally (Docker up) and in CI; `-count=2` passes (isolation holds).
- `go.mod`/`go.sum` updated with testcontainers-go; no production (non-test) code behavior changed.
