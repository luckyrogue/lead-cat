# Employee Directory CSV Seeding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On startup, full-sync an embedded `employees.csv` into the `employees` table of every Google-configured workspace, so the meetings directory search (`/edit` participant add, `/schedule`, `/checker`, Mini App `/employees`) works against real data.

**Architecture:** A new `internal/platform/employeedir` package holds the embedded CSV, a pure `Parse` function, and a best-effort `Seed` orchestrator. Two new build-verified repo queries (`ListWorkspacesWithGoogle`, transactional `SyncEmployees`) do the per-workspace full-sync. `Seed` is called once in `cmd/server/main.go` after the store is built. No new env/config.

**Tech Stack:** Go, `//go:embed`, `encoding/csv`, pgx/Postgres, zap.

**Spec:** `docs/superpowers/specs/2026-05-31-employee-csv-seeding-design.md`

## Codebase facts (verified — rely on these, but confirm before editing)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`. Imports are `github.com/Jaryq-Lab/notify-bot/internal/...`.
- **`Store`** (`internal/infrastructure/persistence/postgres/store.go`): `type Store struct { pool *pgxpool.Pool; log *zap.Logger }`, constructed by `New(pool, log)`. Repo methods use receiver `s` and `s.pool`.
- **Transaction idiom** (see `meeting_repo.go:72`): `tx, err := s.pool.Begin(ctx)` → `defer func() { _ = tx.Rollback(ctx) }()` → `tx.Exec` / `tx.QueryRow` → `tx.Commit(ctx)`.
- **`employees` table** (`migrations/20260530120000_meetings.sql`): `id, workspace_id (NOT NULL, FK), full_name, email, dept (DEFAULT ''), has_telegram (DEFAULT false), created_at`, `UNIQUE (workspace_id, email)`. `has_telegram` is only ever set by `CreateEmployee` — never maintained dynamically; the upsert must NOT touch it.
- **Google marker:** `workspaces.google_sa_json_enc BYTEA` — configured ⇔ `IS NOT NULL`.
- **Existing repo file:** `internal/infrastructure/persistence/postgres/employee_repo.go` (has `ListEmployees`, `SearchEmployeesGlobal`, `CreateEmployee`; imports `context`, `github.com/google/uuid`). New queries go here.
- **Bootstrap** (`cmd/server/main.go`): `pool` → `runMigrations` (if `AUTO_MIGRATE`) → `store := postgres.New(pool, logger)` → `cipher, err := crypto.NewTokenCipher(...)`. Insert the seed call on the line immediately **after** `store := postgres.New(pool, logger)`. `ctx` and `logger` are already in scope.
- **pgx arrays:** passing a Go `[]string` binds to a Postgres `text[]`; `email <> ALL($2)` works directly.

## Conventions

- Run checks from the **repo root**: `make test && make lint && make build`. Run Go directly as `env -u GOROOT go ...` from `backend/`. **`make lint` includes a gofmt check** — run it before committing; if it flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/... ./cmd/...`.
- Backend convention: **pure logic is unit-tested; repo/IO/wiring is build-verified** (no DB harness in the postgres package).
- Do **not** touch `frontend/vite.config.ts` (long-standing local-only edit).

## File structure (created/modified)

- Create `backend/internal/platform/employeedir/parse.go` + `parse_test.go` — pure `Record` + `Parse`.
- Create `backend/internal/platform/employeedir/employees.csv` — embedded starter directory.
- Create `backend/internal/platform/employeedir/employeedir.go` — `//go:embed` + `Seed`.
- Modify `backend/internal/infrastructure/persistence/postgres/employee_repo.go` — `EmployeeSeed`, `ListWorkspacesWithGoogle`, `SyncEmployees`.
- Modify `backend/cmd/server/main.go` — call `employeedir.Seed`.
- Modify `docs/MEETINGS.md`, `docs/REQUIREMENTS.md` — status/docs.

---

## Task 1: Pure CSV parser (`Record`, `Parse`)

**Files:**

- Create: `backend/internal/platform/employeedir/parse.go`
- Test: `backend/internal/platform/employeedir/parse_test.go`

Pure logic — full TDD. No embed, no DB; this task's package compiles and tests on its own.

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/platform/employeedir/parse_test.go`:

```go
package employeedir

import "testing"

const goodCSV = "full_name,email,department\n" +
	"Иванов Иван Иванович,I.Ivanov@Company.kz,Разработка\n" +
	"\n" + // blank line between rows is skipped
	"  Петров Пётр  , p.petrov@company.kz ,  Маркетинг \n" +
	",noname@company.kz,Без имени\n" + // empty full_name is allowed
	"Без Почты,,Отдел\n" // empty email row is skipped

func TestParse_Good(t *testing.T) {
	recs, err := Parse([]byte(goodCSV))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("want 3 records, got %d: %+v", len(recs), recs)
	}
	// email lower-cased, fields trimmed
	if recs[0].Email != "i.ivanov@company.kz" || recs[0].FullName != "Иванов Иван Иванович" || recs[0].Dept != "Разработка" {
		t.Fatalf("rec0 wrong: %+v", recs[0])
	}
	if recs[1].FullName != "Петров Пётр" || recs[1].Email != "p.petrov@company.kz" || recs[1].Dept != "Маркетинг" {
		t.Fatalf("rec1 not trimmed/lowered: %+v", recs[1])
	}
}

func TestParse_CRLF(t *testing.T) {
	recs, err := Parse([]byte("full_name,email,department\r\nИ И,i@c.kz,Dev\r\n"))
	if err != nil || len(recs) != 1 || recs[0].Email != "i@c.kz" {
		t.Fatalf("CRLF parse failed: %v %+v", err, recs)
	}
}

func TestParse_BadHeader(t *testing.T) {
	if _, err := Parse([]byte("name,mail,dep\nA,a@c.kz,D\n")); err == nil {
		t.Fatal("expected header error")
	}
}

func TestParse_HeaderOnlyIsEmpty(t *testing.T) {
	recs, err := Parse([]byte("full_name,email,department\n"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("want 0 records, got %d", len(recs))
	}
}

func TestParse_AllBlankIsEmpty(t *testing.T) {
	recs, err := Parse([]byte("\n\n\n"))
	if err != nil || len(recs) != 0 {
		t.Fatalf("want 0 records no error, got %d / %v", len(recs), err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env -u GOROOT go test ./internal/platform/employeedir/ -v`
Expected: FAIL — `undefined: Parse` / `undefined: Record`.

- [ ] **Step 3: Implement the parser**

Create `backend/internal/platform/employeedir/parse.go`:

```go
// Package employeedir loads the corporate employee directory from an embedded
// CSV and syncs it into Google-configured workspaces. §1.2 / §9.4
package employeedir

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

// Record is one parsed directory entry. Email is normalized to lower-case.
type Record struct {
	FullName string
	Email    string
	Dept     string
}

var wantHeader = []string{"full_name", "email", "department"}

// Parse reads employees.csv bytes into records. The header row must be exactly
// full_name,email,department (case- and whitespace-insensitive). Blank lines and
// rows with an empty email are skipped; remaining fields are trimmed and the
// email is lower-cased. §9.4
func Parse(data []byte) ([]Record, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // tolerate ragged rows; we validate the header ourselves
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse employees csv: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	head := rows[0]
	if len(head) < len(wantHeader) {
		return nil, fmt.Errorf("invalid header: want full_name,email,department")
	}
	for i, want := range wantHeader {
		if strings.ToLower(strings.TrimSpace(head[i])) != want {
			return nil, fmt.Errorf("invalid header: want full_name,email,department")
		}
	}
	var out []Record
	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(row[1]))
		if email == "" {
			continue
		}
		out = append(out, Record{
			FullName: strings.TrimSpace(row[0]),
			Email:    email,
			Dept:     strings.TrimSpace(row[2]),
		})
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && env -u GOROOT go test ./internal/platform/employeedir/ -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/employeedir/parse.go backend/internal/platform/employeedir/parse_test.go
git commit -m "feat(meetings): pure employees.csv parser §9.4"
```

---

## Task 2: Repo — `EmployeeSeed`, `ListWorkspacesWithGoogle`, `SyncEmployees`

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/employee_repo.go`

Build-verified (no DB harness). Read the existing file first to match style; it already imports `context` and `github.com/google/uuid`.

- [ ] **Step 1: Append the type and the two queries**

Append to the end of `backend/internal/infrastructure/persistence/postgres/employee_repo.go`:

```go
// EmployeeSeed is one directory row to sync. Email is pre-normalized to
// lower-case by the caller. Defined here (not in employeedir) so the repo stays
// free of the platform package and there is no import cycle. §9.4
type EmployeeSeed struct {
	FullName string
	Email    string
	Dept     string
}

// ListWorkspacesWithGoogle returns IDs of workspaces that have Google
// service-account credentials configured (google_sa_json_enc IS NOT NULL).
func (s *Store) ListWorkspacesWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM workspaces WHERE google_sa_json_enc IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SyncEmployees makes workspaceID's employees rows mirror seeds in one
// transaction: rows whose email is not in seeds are deleted; present rows are
// upserted (full_name, dept). has_telegram is left untouched on existing rows.
// Returns per-op counts. Empty seeds is a no-op (caller guards against an empty
// CSV; this is belt-and-suspenders so a stray empty call never wipes the table).
// §9.4
func (s *Store) SyncEmployees(ctx context.Context, workspaceID uuid.UUID, seeds []EmployeeSeed) (added, updated, deleted int, err error) {
	if len(seeds) == 0 {
		return 0, 0, 0, nil
	}
	emails := make([]string, 0, len(seeds))
	for _, sd := range seeds {
		emails = append(emails, sd.Email)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `DELETE FROM employees WHERE workspace_id = $1 AND email <> ALL($2)`, workspaceID, emails)
	if err != nil {
		return 0, 0, 0, err
	}
	deleted = int(ct.RowsAffected())

	for _, sd := range seeds {
		var inserted bool
		if err = tx.QueryRow(ctx, `
			INSERT INTO employees (workspace_id, full_name, email, dept)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (workspace_id, email) DO UPDATE
				SET full_name = EXCLUDED.full_name, dept = EXCLUDED.dept
			RETURNING (xmax = 0)`,
			workspaceID, sd.FullName, sd.Email, sd.Dept).Scan(&inserted); err != nil {
			return 0, 0, 0, err
		}
		if inserted {
			added++
		} else {
			updated++
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, 0, err
	}
	return added, updated, deleted, nil
}
```

> `RETURNING (xmax = 0)` is the standard Postgres trick: on a fresh INSERT the system column `xmax` is 0 (true ⇒ inserted), on a conflict-update it is non-zero (false ⇒ updated).

- [ ] **Step 2: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/employee_repo.go
git commit -m "feat(meetings): employee directory sync repo queries §9.4"
```

---

## Task 3: Embedded CSV + `Seed` orchestrator

**Files:**

- Create: `backend/internal/platform/employeedir/employees.csv`
- Create: `backend/internal/platform/employeedir/employeedir.go`

The `//go:embed` directive requires `employees.csv` to exist in the package directory at build time — create it in this task before building.

- [ ] **Step 1: Create the embedded CSV**

Create `backend/internal/platform/employeedir/employees.csv` (starter rows — real data is edited here and the binary rebuilt):

```
full_name,email,department
Иванов Иван Иванович,i.ivanov@company.kz,Разработка
Петрова Анна Сергеевна,a.petrova@company.kz,Маркетинг
```

- [ ] **Step 2: Create the seeder**

Create `backend/internal/platform/employeedir/employeedir.go`:

```go
package employeedir

import (
	"context"
	_ "embed"

	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

//go:embed employees.csv
var csvData []byte

// Seed parses the embedded directory CSV and full-syncs it into every
// Google-configured workspace. Best-effort: all failures are logged, never
// fatal — a directory glitch must not take the server down. If the CSV parses
// to zero records the sync is skipped entirely (guard against an empty/truncated
// CSV wiping the directory). §9.4
func Seed(ctx context.Context, store *postgres.Store, log *zap.Logger) {
	records, err := Parse(csvData)
	if err != nil {
		log.Error("employee_csv_parse_failed", zap.Error(err))
		return
	}
	if len(records) == 0 {
		log.Warn("employee_csv_empty")
		return
	}
	seeds := make([]postgres.EmployeeSeed, 0, len(records))
	for _, r := range records {
		seeds = append(seeds, postgres.EmployeeSeed{FullName: r.FullName, Email: r.Email, Dept: r.Dept})
	}
	wsIDs, err := store.ListWorkspacesWithGoogle(ctx)
	if err != nil {
		log.Error("employee_seed_failed", zap.Error(err))
		return
	}
	for _, wsID := range wsIDs {
		added, updated, deleted, serr := store.SyncEmployees(ctx, wsID, seeds)
		if serr != nil {
			log.Error("employee_sync_failed", zap.String("workspace_id", wsID.String()), zap.Error(serr))
			continue
		}
		log.Info("employees_synced",
			zap.String("workspace_id", wsID.String()),
			zap.Int("added", added), zap.Int("updated", updated), zap.Int("deleted", deleted))
	}
	log.Info("employee_seed_done", zap.Int("workspaces", len(wsIDs)))
}
```

- [ ] **Step 3: Build + vet + test the package**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/platform/employeedir/ && env -u GOROOT go test ./internal/platform/employeedir/`
Expected: builds, vets, parser tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/platform/employeedir/employees.csv backend/internal/platform/employeedir/employeedir.go
git commit -m "feat(meetings): embedded employees.csv + Seed orchestrator §9.4"
```

---

## Task 4: Wire `Seed` into server startup

**Files:**

- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add the import**

In `cmd/server/main.go`, add to the import block (with the other `internal/platform/...` imports):

```go
	"github.com/Jaryq-Lab/notify-bot/internal/platform/employeedir"
```

- [ ] **Step 2: Call `Seed` after the store is built**

Find the line `store := postgres.New(pool, logger)` and insert the seed call immediately after it:

```go
	store := postgres.New(pool, logger)
	employeedir.Seed(ctx, store, logger)
```

> Synchronous and best-effort, before HTTP/bot start, so the directory is ready on the first request. No new config.

- [ ] **Step 3: Build + vet**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./cmd/server/`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(meetings): seed employee directory on startup §9.4"
```

---

## Task 5: Docs + final verification

**Files:**

- Modify: `docs/MEETINGS.md`
- Modify: `docs/REQUIREMENTS.md`

- [ ] **Step 1: Update `docs/MEETINGS.md`**

In the Backend (planned) block, after the last `> **...(done):**` line, add:

```markdown
> **Employee directory CSV seeding (§1.2/§9.4, done):** on startup the server full-syncs an **embedded** `internal/platform/employeedir/employees.csv` (columns `full_name,email,department`) into every Google-configured workspace (`google_sa_json_enc IS NOT NULL`): rows missing from the CSV are deleted, present rows upserted (`has_telegram` untouched). Pure `employeedir.Parse` is unit-tested; the per-workspace sync is one transaction (`SyncEmployees`). Best-effort (logs `employees_synced` / `employee_seed_failed`, never fatal); an empty CSV is skipped (guard). To change the directory: edit the CSV, rebuild, redeploy. Hot-reload and a bot/admin management UI remain out of scope.
```

- [ ] **Step 2: Update `docs/REQUIREMENTS.md`**

Edit `docs/REQUIREMENTS.md` line ~128. Replace this exact line:

```markdown
- **New prerequisites (planned, when backend lands):** Google service-account credentials + employees CSV — to be added to `deploy/.env.example` and §1–2 above.
```

with:

```markdown
- **New prerequisites:** Google service-account credentials are configured **per workspace** via `PATCH /api/workspaces/:id/integrations` (no env var). The employee directory is an **embedded CSV** (`backend/internal/platform/employeedir/employees.csv`), full-synced into Google-configured workspaces on boot — there is **no** env var; to change it, edit the CSV and redeploy.
```

- [ ] **Step 3: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/... ./cmd/...` and re-run.

- [ ] **Step 4: Commit**

```bash
git add docs/MEETINGS.md docs/REQUIREMENTS.md
git commit -m "docs(meetings): document employee directory CSV seeding §9.4"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** embedded CSV + columns → Task 3 (`employees.csv`) + Task 1 (`Parse` header `full_name,email,department`); seed into Google-configured workspaces → Task 2 (`ListWorkspacesWithGoogle`) + Task 3 (`Seed` loop); full-sync with delete-missing + upsert, `has_telegram` preserved → Task 2 (`SyncEmployees`); zero-row guard → Task 1 (`Parse` returns 0) + Task 3 (`len(records)==0` warn+return) + Task 2 (defensive `len(seeds)==0`); best-effort non-fatal + structured logs → Task 3; wiring after store, before serving → Task 4; docs → Task 5. No new env/config (intentional, per spec).
- **Type consistency:** `employeedir.Record{FullName,Email,Dept}` (Task 1) maps to `postgres.EmployeeSeed{FullName,Email,Dept}` (Task 2) in `Seed` (Task 3). `SyncEmployees(ctx, workspaceID uuid.UUID, seeds []postgres.EmployeeSeed) (added, updated, deleted int, err error)` consumed in Task 3. `ListWorkspacesWithGoogle(ctx) ([]uuid.UUID, error)` consumed in Task 3. `Seed(ctx, store *postgres.Store, log *zap.Logger)` consumed in Task 4.
- **Out of scope (do not implement):** `EMPLOYEES_CSV_PATH` env override, hot-reload, bot/admin directory UI, `has_telegram` maintenance, export/AD integration.
- **Known approximations:** `SyncEmployees` does one `QueryRow` per row (directory is small — fine). The delete uses `email <> ALL($2)`; the empty-seeds guard (both in `Seed` and `SyncEmployees`) prevents an all-rows delete. `has_telegram` defaults false for CSV-inserted rows and is never set elsewhere, so it is effectively always false today.

```

```
