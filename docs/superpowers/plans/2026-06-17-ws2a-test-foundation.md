# WS2a — Test Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a `testcontainers-go` Postgres harness and use it (plus DB-free domain tests) to verify the recurrence/series math and the Postgres persistence layer against a real, ephemeral Postgres.

**Architecture:** A new `internal/testsupport/pgtest` package boots one `postgres:16-alpine` container per repo-test package (via `TestMain`), applies the 20 goose migrations (dir resolved by `runtime.Caller`), and exposes a pool + per-test `Truncate`. Domain tests are pure and DB-free. Repo tests seed via the real Store methods (`UpsertUserIdentity` → `CreateOrganization` → meeting).

**Tech Stack:** Go 1.26, testcontainers-go (+ `modules/postgres`), pgx v5, goose v3, stdlib `testing`.

**Standing constraints (every task):**
- Work directly on `main`; no feature branches. Commit per task; the human pushes on request.
- Run Go with `env -u GOROOT`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- Stage only the explicit paths each task lists; never `git add -A`. The human commits other files in parallel — `git status` before staging.
- Requires Docker running locally to execute repo tests. Domain tests (Task 2) need no Docker.
- **testcontainers-go API note:** the code below targets the current module API (`postgres.Run`, `wait.ForLog`, `provider.Health`). If `go get` resolves a version whose API differs, adapt the calls — the required behavior (boot Postgres, get a DSN, wait until ready) is what matters, not the exact symbol names.

---

## File Structure

| File | Responsibility | Task |
|---|---|---|
| `apps/backend/internal/testsupport/pgtest/pgtest.go` | Container lifecycle, migrations, pool, `Truncate`, `DockerAvailable` | 1 |
| `apps/backend/internal/domain/meeting/recurrence_test.go` | `Occurrences` + `nextStep`/`isoWeekday` behavior | 2 |
| `apps/backend/internal/domain/meeting/conflict_test.go` | `Overlaps`, `FreeSlots` | 2 |
| `apps/backend/internal/domain/meeting/naming_test.go` | `GenerateName` | 2 |
| `apps/backend/internal/domain/meeting/validate_test.go` | `Input.Validate`, `Recurrence.Valid` | 2 |
| `apps/backend/internal/infrastructure/persistence/postgres/main_test.go` | `TestMain` + shared `*pgtest.DB` + seed helpers | 3 |
| `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo_test.go` | meetings CRUD, series tx, participants | 3 |
| `apps/backend/internal/infrastructure/persistence/postgres/org_user_repo_test.go` | users, orgs, members | 4 |
| `apps/backend/internal/infrastructure/persistence/postgres/bot_session_audit_repo_test.go` | bot_users, web sessions, audit, reminder claim | 4 |

---

### Task 1: testcontainers Postgres harness

**Files:**
- Create: `apps/backend/internal/testsupport/pgtest/pgtest.go`
- Modify: `apps/backend/go.mod`, `apps/backend/go.sum` (new dep)

- [ ] **Step 1: Add the dependency**

Run: `cd apps/backend && env -u GOROOT go get github.com/testcontainers/testcontainers-go github.com/testcontainers/testcontainers-go/modules/postgres`
Expected: `go.mod`/`go.sum` updated, no error.

- [ ] **Step 2: Write the harness**

Create `apps/backend/internal/testsupport/pgtest/pgtest.go`:

```go
// Package pgtest provides an ephemeral Postgres for repository tests.
// It boots a throwaway container, applies the goose migrations once, and
// hands back a pgx pool. One container per test package (via TestMain).
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// DB is a ready, migrated ephemeral Postgres.
type DB struct {
	Pool      *pgxpool.Pool
	container testcontainers.Container
}

// DockerAvailable reports whether a usable Docker endpoint exists, so callers
// can skip DB tests gracefully on machines without Docker.
func DockerAvailable() bool {
	p, err := testcontainers.NewDockerProvider()
	if err != nil {
		return false
	}
	defer func() { _ = p.Close() }()
	return p.Health(context.Background()) == nil
}

// Start boots Postgres, applies migrations, and returns a connected DB.
// Call from TestMain; Close it when m.Run() returns.
func Start(ctx context.Context) (*DB, error) {
	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("leadcat_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("connection string: %w", err)
	}
	if err := migrate(dsn); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	return &DB{Pool: pool, container: ctr}, nil
}

func migrate(dsn string) error {
	dir := migrationsDir()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = db.Close() }()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("goose up (%s): %w", dir, err)
	}
	return nil
}

// migrationsDir resolves apps/backend/migrations from this file's location,
// so tests work regardless of the working directory.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// pgtest.go -> pgtest -> testsupport -> internal -> backend
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations")
}

// Store returns a repository bound to the pool.
func (d *DB) Store(log *zap.Logger) *pg.Store { return pg.New(d.Pool, log) }

// Truncate wipes every application table so each test starts clean.
func (d *DB) Truncate(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	rows, err := d.Pool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if len(tables) == 0 {
		return
	}
	stmt := "TRUNCATE "
	for i, tn := range tables {
		if i > 0 {
			stmt += ", "
		}
		stmt += `"` + tn + `"`
	}
	stmt += " RESTART IDENTITY CASCADE"
	if _, err := d.Pool.Exec(ctx, stmt); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// Close terminates the container.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
	if d.container != nil {
		_ = d.container.Terminate(context.Background())
	}
}
```

- [ ] **Step 3: Verify it compiles and migrations resolve**

Run: `cd apps/backend && env -u GOROOT go build ./internal/testsupport/pgtest/`
Expected: builds clean.

Run: `cd apps/backend && env -u GOROOT go vet ./internal/testsupport/pgtest/`
Expected: no issues.

(The harness is exercised end-to-end by Task 3's `TestMain`; no standalone test here — a self-test would just duplicate Task 3.)

- [ ] **Step 4: Commit**

```bash
git add apps/backend/internal/testsupport/pgtest/pgtest.go apps/backend/go.mod apps/backend/go.sum
git commit -m "$(cat <<'EOF'
test(backend): testcontainers Postgres harness (pgtest)

Boots an ephemeral postgres:16, applies goose migrations (dir via
runtime.Caller), exposes a pool + per-test Truncate. Foundation for WS2a
repo tests.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Domain unit tests (no DB)

**Files:**
- Create: `apps/backend/internal/domain/meeting/recurrence_test.go`
- Create: `apps/backend/internal/domain/meeting/conflict_test.go`
- Create: `apps/backend/internal/domain/meeting/naming_test.go`
- Create: `apps/backend/internal/domain/meeting/validate_test.go`

Reference (already in the package): `Occurrences(start, end, r Recurrence, days []int, until) ([]Span, error)`; recurrences `Once/Daily/Weekly/Custom/Monthly`; sentinels `ErrRecurrenceWindow`, `ErrRecurrenceDays`, `ErrTooManyOccurrences`; `isoWeekday` (Mon=1…Sun=7); `Overlaps(aStart,aEnd,bStart,bEnd)`; `FreeSlots(busy, winStart, winEnd, minDur)`; `GenerateName(dept,type,host,date,r)`; `Input{Dept,Type,Host,StartsAt,EndsAt,Recurrence,RecurrenceDays,...}`; `Recurrence.Valid()`.

- [ ] **Step 1: Write recurrence tests**

Create `apps/backend/internal/domain/meeting/recurrence_test.go`:

```go
package meeting

import (
	"errors"
	"testing"
	"time"
)

func ts(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

func TestOccurrences_Once_SingleSpan(t *testing.T) {
	start, end := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 11, 0)
	got, err := Occurrences(start, end, Once, nil, time.Time{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || !got[0].Start.Equal(start) || !got[0].End.Equal(end) {
		t.Fatalf("got %+v", got)
	}
}

func TestOccurrences_Daily_InclusiveCount(t *testing.T) {
	start, end := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 10, 30)
	until := ts(2026, 6, 5, 0, 0) // 5 days: Jun 1..5
	got, err := Occurrences(start, end, Daily, nil, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("want 5 occurrences, got %d", len(got))
	}
	if got[0].End.Sub(got[0].Start) != 30*time.Minute {
		t.Fatalf("duration not preserved: %v", got[0].End.Sub(got[0].Start))
	}
}

func TestOccurrences_Weekly_StepsBySevenDays(t *testing.T) {
	start, end := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 9, 30)
	until := ts(2026, 6, 22, 0, 0) // Jun 1, 8, 15, 22
	got, err := Occurrences(start, end, Weekly, nil, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4, got %d", len(got))
	}
	if got[1].Start.Sub(got[0].Start) != 7*24*time.Hour {
		t.Fatalf("not weekly: %v", got[1].Start.Sub(got[0].Start))
	}
}

func TestOccurrences_Custom_OnlyAllowedWeekdays(t *testing.T) {
	start, end := ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 8, 30) // Jun 1 2026 is a Monday
	until := ts(2026, 6, 14, 0, 0)
	days := []int{1, 3} // Mon, Wed
	got, err := Occurrences(start, end, Custom, days, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected some occurrences")
	}
	allowed := map[int]bool{1: true, 3: true}
	for _, s := range got {
		if !allowed[isoWeekday(s.Start)] {
			t.Fatalf("occurrence on disallowed weekday: %v (iso %d)", s.Start, isoWeekday(s.Start))
		}
	}
}

func TestOccurrences_Custom_NoDays_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 9, 0), Custom, nil, ts(2026, 6, 30, 0, 0))
	if !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}

func TestOccurrences_MissingUntil_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 9, 0), Daily, nil, time.Time{})
	if !errors.Is(err, ErrRecurrenceWindow) {
		t.Fatalf("want ErrRecurrenceWindow, got %v", err)
	}
}

func TestOccurrences_UntilBeforeStart_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 10, 8, 0), ts(2026, 6, 10, 9, 0), Daily, nil, ts(2026, 6, 1, 0, 0))
	if !errors.Is(err, ErrRecurrenceWindow) {
		t.Fatalf("want ErrRecurrenceWindow, got %v", err)
	}
}

func TestOccurrences_TooMany_Errors(t *testing.T) {
	start, end := ts(2026, 1, 1, 8, 0), ts(2026, 1, 1, 9, 0)
	until := start.AddDate(0, 0, 150) // 151 daily occurrences > 100 cap
	_, err := Occurrences(start, end, Daily, nil, until)
	if !errors.Is(err, ErrTooManyOccurrences) {
		t.Fatalf("want ErrTooManyOccurrences, got %v", err)
	}
}

func TestIsoWeekday_SundayIsSeven(t *testing.T) {
	sunday := ts(2026, 6, 7, 0, 0) // Jun 7 2026 is a Sunday
	if got := isoWeekday(sunday); got != 7 {
		t.Fatalf("want 7, got %d", got)
	}
}
```

- [ ] **Step 2: Write conflict tests**

Create `apps/backend/internal/domain/meeting/conflict_test.go`:

```go
package meeting

import (
	"testing"
	"time"
)

func TestOverlaps(t *testing.T) {
	a1, a2 := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 11, 0)
	cases := []struct {
		name               string
		b1, b2             time.Time
		want               bool
	}{
		{"contained", ts(2026, 6, 1, 10, 15), ts(2026, 6, 1, 10, 45), true},
		{"partial-overlap", ts(2026, 6, 1, 10, 30), ts(2026, 6, 1, 11, 30), true},
		{"touching-edge", ts(2026, 6, 1, 11, 0), ts(2026, 6, 1, 12, 0), false},
		{"disjoint", ts(2026, 6, 1, 12, 0), ts(2026, 6, 1, 13, 0), false},
	}
	for _, c := range cases {
		if got := Overlaps(a1, a2, c.b1, c.b2); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestFreeSlots_FiltersShortGaps(t *testing.T) {
	win0, win1 := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 17, 0)
	busy := []Span{
		{Start: ts(2026, 6, 1, 10, 0), End: ts(2026, 6, 1, 10, 20)},
		{Start: ts(2026, 6, 1, 10, 40), End: ts(2026, 6, 1, 12, 0)},
	}
	free := FreeSlots(busy, win0, win1, 30*time.Minute)
	for _, f := range free {
		if f.End.Sub(f.Start) < 30*time.Minute {
			t.Fatalf("slot shorter than minDur survived: %v", f)
		}
	}
	// the 20-minute gap (10:20-10:40) must be excluded
	for _, f := range free {
		if f.Start.Equal(ts(2026, 6, 1, 10, 20)) {
			t.Fatal("short gap should have been filtered")
		}
	}
}

func TestFreeSlots_NoBusy_ReturnsWholeWindow(t *testing.T) {
	win0, win1 := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 10, 0)
	free := FreeSlots(nil, win0, win1, 15*time.Minute)
	if len(free) != 1 || !free[0].Start.Equal(win0) || !free[0].End.Equal(win1) {
		t.Fatalf("got %+v", free)
	}
}
```

- [ ] **Step 3: Write naming + validate tests**

Create `apps/backend/internal/domain/meeting/naming_test.go`:

```go
package meeting

import (
	"strings"
	"testing"
)

func TestGenerateName_WithFrequencyLabel(t *testing.T) {
	got := GenerateName("Eng", "Standup", "Mia", ts(2026, 6, 1, 9, 0), Daily)
	if !strings.HasPrefix(got, "Eng | Standup | Mia | 2026-06-01") {
		t.Fatalf("bad prefix: %q", got)
	}
	if !strings.HasSuffix(got, "| "+Daily.Label()) {
		t.Fatalf("missing frequency label: %q", got)
	}
}

func TestGenerateName_OnceHasNoFrequencySuffix(t *testing.T) {
	got := GenerateName("Eng", "Sync", "Mia", ts(2026, 6, 1, 9, 0), Once)
	if got != "Eng | Sync | Mia | 2026-06-01" {
		t.Fatalf("unexpected: %q", got)
	}
}
```

Create `apps/backend/internal/domain/meeting/validate_test.go`:

```go
package meeting

import (
	"errors"
	"testing"
	"time"
)

func validInput() Input {
	return Input{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt: ts(2026, 6, 1, 10, 0), EndsAt: ts(2026, 6, 1, 11, 0),
		Recurrence: Once,
	}
}

func TestValidate_OK(t *testing.T) {
	if err := validInput().Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidate_EndBeforeStart(t *testing.T) {
	in := validInput()
	in.EndsAt = in.StartsAt.Add(-time.Minute)
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for end<=start")
	}
}

func TestValidate_MissingFields(t *testing.T) {
	in := validInput()
	in.Dept = ""
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for missing dept")
	}
}

func TestValidate_CustomNeedsDays(t *testing.T) {
	in := validInput()
	in.Recurrence = Custom
	if err := in.Validate(); !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}

func TestValidate_WeekdayRange(t *testing.T) {
	in := validInput()
	in.Recurrence = Custom
	in.RecurrenceDays = []int{0}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for weekday out of 1..7")
	}
}

func TestRecurrence_Valid(t *testing.T) {
	if !Daily.Valid() || Recurrence("nope").Valid() {
		t.Fatal("Valid() wrong")
	}
}
```

- [ ] **Step 4: Run domain tests**

Run: `cd apps/backend && env -u GOROOT go test ./internal/domain/meeting/ -v`
Expected: all tests PASS (no Docker needed).

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/domain/meeting/recurrence_test.go \
  apps/backend/internal/domain/meeting/conflict_test.go \
  apps/backend/internal/domain/meeting/naming_test.go \
  apps/backend/internal/domain/meeting/validate_test.go
git commit -m "$(cat <<'EOF'
test(domain): cover recurrence, conflict, naming, validation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Repo tests — TestMain, seed helpers, meetings

**Files:**
- Create: `apps/backend/internal/infrastructure/persistence/postgres/main_test.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo_test.go`

Reference signatures (from the package): `New(pool,log) *Store`; `UpsertUserIdentity(ctx, authSub, email) (User,error)`; `CreateOrganization(ctx, name, slug, ownerUserID uuid.UUID) (Organization,error)`; `CreateMeeting(ctx, Meeting) (Meeting,error)`; `CreateMeetingSeries(ctx, []Meeting, []MeetingParticipant) ([]Meeting,error)`; `GetMeeting(ctx, orgID, id) (Meeting,error)`; `UpdateMeeting(ctx, orgID, id, Meeting) error`; `CancelMeeting(ctx, orgID, id) error`; `ListMeetings(ctx, orgID) ([]Meeting,error)`; `AddParticipants/RemoveParticipant/ListParticipants`. `Meeting` fields: `OrganizationID, Dept, Type, Host, StartsAt, EndsAt, Recurrence(string), Name, Status, SeriesID *uuid.UUID, RecurrenceUntil *time.Time, RecurrenceDays []int`. `MeetingParticipant{EmployeeID *uuid.UUID, Email string}`.

- [ ] **Step 1: Write TestMain + shared DB + seed helpers**

Create `apps/backend/internal/infrastructure/persistence/postgres/main_test.go`:

```go
package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/testsupport/pgtest"
)

var testDB *pgtest.DB

func TestMain(m *testing.M) {
	if !pgtest.DockerAvailable() {
		fmt.Fprintln(os.Stderr, "pgtest: Docker unavailable — skipping postgres repo tests")
		os.Exit(0)
	}
	db, err := pgtest.Start(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgtest start: %v\n", err)
		os.Exit(1)
	}
	testDB = db
	code := m.Run()
	db.Close()
	os.Exit(code)
}

func newStore() *Store { return New(testDB.Pool, zap.NewNop()) }

// seedOrg creates a user + organization and returns the org id.
func seedOrg(t *testing.T, s *Store) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	u, err := s.UpsertUserIdentity(ctx, "sub-"+uuid.NewString(), uuid.NewString()+"@x.io")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	org, err := s.CreateOrganization(ctx, "Org", "org-"+uuid.NewString()[:8], u.ID)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}
	return org.ID
}

// seedMeeting inserts a single scheduled meeting in the org.
func seedMeeting(t *testing.T, s *Store, orgID uuid.UUID) Meeting {
	t.Helper()
	m, err := s.CreateMeeting(context.Background(), Meeting{
		OrganizationID: orgID,
		Dept:           "Eng", Type: "Sync", Host: "Mia",
		StartsAt:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		EndsAt:     time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
		Recurrence: "once", Name: "Eng | Sync | Mia | 2026-06-01",
	})
	if err != nil {
		t.Fatalf("seed meeting: %v", err)
	}
	return m
}
```

- [ ] **Step 2: Write meeting repo tests**

Create `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo_test.go`:

```go
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMeeting_CreateGetRoundTrip(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	created := seedMeeting(t, s, org)

	got, err := s.GetMeeting(context.Background(), org, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Dept != "Eng" || got.Type != "Sync" || got.Host != "Mia" {
		t.Fatalf("fields not persisted: %+v", got)
	}
	if !got.StartsAt.Equal(created.StartsAt) {
		t.Fatalf("starts_at mismatch: %v vs %v", got.StartsAt, created.StartsAt)
	}
}

func TestMeeting_Update(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	m := seedMeeting(t, s, org)

	m.Dept = "Sales"
	m.Name = "Sales | Sync | Mia | 2026-06-01"
	if err := s.UpdateMeeting(context.Background(), org, m.ID, m); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := s.GetMeeting(context.Background(), org, m.ID)
	if got.Dept != "Sales" {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestMeeting_Cancel(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	m := seedMeeting(t, s, org)
	if err := s.CancelMeeting(context.Background(), org, m.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	got, _ := s.GetMeeting(context.Background(), org, m.ID)
	if got.Status != "cancelled" {
		t.Fatalf("want cancelled, got %q", got.Status)
	}
}

func TestMeeting_CreateSeries_Atomic(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	series := uuid.New()
	until := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	ms := []Meeting{
		{OrganizationID: org, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "d1",
			StartsAt: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
			SeriesID: &series, RecurrenceUntil: &until},
		{OrganizationID: org, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "d2",
			StartsAt: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 2, 10, 30, 0, 0, time.UTC),
			SeriesID: &series, RecurrenceUntil: &until},
	}
	ps := []MeetingParticipant{{Email: "a@x.io"}, {Email: "b@x.io"}}
	out, err := s.CreateMeetingSeries(context.Background(), ms, ps)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 meetings, got %d", len(out))
	}
	all, _ := s.ListMeetings(context.Background(), org)
	if len(all) != 2 {
		t.Fatalf("want 2 persisted, got %d", len(all))
	}
	parts, _ := s.ListParticipants(context.Background(), out[0].ID)
	if len(parts) != 2 {
		t.Fatalf("want 2 participants on first meeting, got %d", len(parts))
	}
}

func TestMeeting_CreateSeries_RollsBackOnBadRow(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	bogusOrg := uuid.New() // no such organization -> FK violation on the 2nd row
	ms := []Meeting{
		{OrganizationID: org, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "ok",
			StartsAt: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)},
		{OrganizationID: bogusOrg, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "bad",
			StartsAt: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 2, 10, 30, 0, 0, time.UTC)},
	}
	if _, err := s.CreateMeetingSeries(context.Background(), ms, nil); err == nil {
		t.Fatal("expected error on bogus org FK")
	}
	all, _ := s.ListMeetings(context.Background(), org)
	if len(all) != 0 {
		t.Fatalf("transaction did not roll back: %d rows persist", len(all))
	}
}

func TestMeeting_Participants_AddRemove(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	org := seedOrg(t, s)
	m := seedMeeting(t, s, org)
	ctx := context.Background()
	if err := s.AddParticipants(ctx, m.ID, []MeetingParticipant{{Email: "a@x.io"}, {Email: "b@x.io"}}); err != nil {
		t.Fatalf("add: %v", err)
	}
	// idempotent add
	_ = s.AddParticipants(ctx, m.ID, []MeetingParticipant{{Email: "a@x.io"}})
	parts, _ := s.ListParticipants(ctx, m.ID)
	if len(parts) != 2 {
		t.Fatalf("want 2 unique participants, got %d", len(parts))
	}
	if err := s.RemoveParticipant(ctx, m.ID, "a@x.io"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	parts, _ = s.ListParticipants(ctx, m.ID)
	if len(parts) != 1 || parts[0].Email != "b@x.io" {
		t.Fatalf("after remove: %+v", parts)
	}
}
```

- [ ] **Step 3: Run the repo tests (needs Docker)**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -v`
Expected: container boots once, all tests PASS. (Confirmed: `meetings.organization_id` is `NOT NULL REFERENCES organizations(id)`, so the bogus-org row in `TestMeeting_CreateSeries_RollsBackOnBadRow` triggers an FK violation mid-transaction, exercising the rollback.)

- [ ] **Step 4: Verify isolation holds**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -count=2`
Expected: PASS (truncate-per-test makes re-runs clean).

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/infrastructure/persistence/postgres/main_test.go \
  apps/backend/internal/infrastructure/persistence/postgres/meeting_repo_test.go
git commit -m "$(cat <<'EOF'
test(postgres): meetings CRUD, series tx atomicity, participants

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Repo tests — orgs/users/members, bot_users, sessions, audit, reminder claim

**Files:**
- Create: `apps/backend/internal/infrastructure/persistence/postgres/org_user_repo_test.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/bot_session_audit_repo_test.go`

Reference signatures: `UpsertUserIdentity(ctx, authSub, email) (User,error)`; `GetUserByID(ctx,id) (User,error)`; `CreateOrganization(ctx,name,slug,ownerUserID) (Organization,error)`; `GetOrganization(ctx,id) (Organization,error)`; `ListOrgMembers(ctx,orgID) ([]Member,error)`; `GetOrgMember(ctx,orgID,userID) (Member,bool,error)`; `UpdateMemberRole(ctx,orgID,userID,role) error`; `RemoveMember(ctx,orgID,userID) error`; `CreateBotUser(ctx,telegramID,fullName,email,role) (BotUser,error)`; `GetBotUserByTelegramID(ctx,telegramID) (BotUser,error)`; `SetReminderMinutes(ctx,telegramID,csv) error`; `SetBotUserPrefs(ctx,telegramID,timezone,language) error`; `CreateWebSession(ctx,tokenHash,userID,expiresAt,ua,ip) (WebSession,error)`; `ResolveWebSession(ctx,tokenHash,now) (WebSession,bool,error)`; `RevokeWebSession(ctx,tokenHash,now) error`; `InsertAuditEntry(ctx,AuditEntry) error`; `ListAuditEntries(ctx,AuditFilter) ([]AuditEntry,error)`; `TryClaimReminder(ctx,meetingID,telegramID,offset) (bool,error)`.

- [ ] **Step 1: Write org/user/member tests**

Create `apps/backend/internal/infrastructure/persistence/postgres/org_user_repo_test.go`:

```go
package postgres

import (
	"context"
	"testing"
)

func TestUser_UpsertIdempotentByEmail(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	u1, err := s.UpsertUserIdentity(ctx, "sub-1", "same@x.io")
	if err != nil {
		t.Fatalf("upsert1: %v", err)
	}
	u2, err := s.UpsertUserIdentity(ctx, "sub-1", "same@x.io")
	if err != nil {
		t.Fatalf("upsert2: %v", err)
	}
	if u1.ID != u2.ID {
		t.Fatalf("upsert created a second user: %v vs %v", u1.ID, u2.ID)
	}
	got, err := s.GetUserByID(ctx, u1.ID)
	if err != nil || got.Email != "same@x.io" {
		t.Fatalf("get user: %v / %+v", err, got)
	}
}

func TestOrg_CreateAndGet(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "owner", "owner@x.io")
	org, err := s.CreateOrganization(ctx, "Acme", "acme", owner.ID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	got, err := s.GetOrganization(ctx, org.ID)
	if err != nil || got.Name != "Acme" || got.Slug != "acme" {
		t.Fatalf("get org: %v / %+v", err, got)
	}
}

func TestOrgMembers_RoleAndRemoval(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "owner", "owner@x.io")
	org, _ := s.CreateOrganization(ctx, "Acme", "acme2", owner.ID)

	// The owner should be a member after org creation.
	members, err := s.ListOrgMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) == 0 {
		t.Fatal("expected at least the owner as a member")
	}

	if err := s.UpdateMemberRole(ctx, org.ID, owner.ID, "admin"); err != nil {
		t.Fatalf("update role: %v", err)
	}
	m, ok, err := s.GetOrgMember(ctx, org.ID, owner.ID)
	if err != nil || !ok || m.Role != "admin" {
		t.Fatalf("role not updated: %v ok=%v %+v", err, ok, m)
	}

	if err := s.RemoveMember(ctx, org.ID, owner.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, ok, _ := s.GetOrgMember(ctx, org.ID, owner.ID); ok {
		t.Fatal("member still present after removal")
	}
}
```

(Confirmed: `CreateOrganization` inserts the owner into `organization_members` with role `owner` in the same transaction, so the "owner is a member" assertion holds.)

- [ ] **Step 2: Write bot_user / session / audit / reminder tests**

Create `apps/backend/internal/infrastructure/persistence/postgres/bot_session_audit_repo_test.go`:

```go
package postgres

import (
	"context"
	"testing"
	"time"
)

func TestBotUser_CreateGetAndPrefs(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	if _, err := s.CreateBotUser(ctx, 12345, "Mia", "mia@x.io", "user"); err != nil {
		t.Fatalf("create bot user: %v", err)
	}
	got, err := s.GetBotUserByTelegramID(ctx, 12345)
	if err != nil || got.FullName != "Mia" || got.Email != "mia@x.io" {
		t.Fatalf("get bot user: %v / %+v", err, got)
	}
	if err := s.SetReminderMinutes(ctx, 12345, "15,60"); err != nil {
		t.Fatalf("set reminders: %v", err)
	}
	if err := s.SetBotUserPrefs(ctx, 12345, "Asia/Almaty", "ru"); err != nil {
		t.Fatalf("set prefs: %v", err)
	}
	got, _ = s.GetBotUserByTelegramID(ctx, 12345)
	if got.ReminderMinutes != "15,60" || got.Timezone != "Asia/Almaty" || got.Language != "ru" {
		t.Fatalf("prefs not persisted: %+v", got)
	}
}

func TestWebSession_CreateResolveRevoke(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "ws-owner", "ws@x.io")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	hash := []byte("token-hash-bytes")
	if _, err := s.CreateWebSession(ctx, hash, owner.ID, now.Add(time.Hour), "ua", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	sess, ok, err := s.ResolveWebSession(ctx, hash, now)
	if err != nil || !ok {
		t.Fatalf("resolve: %v ok=%v", err, ok)
	}
	if sess.UserID != owner.ID {
		t.Fatalf("wrong user: %v", sess.UserID)
	}
	// expired lookup
	if _, ok, _ := s.ResolveWebSession(ctx, hash, now.Add(2*time.Hour)); ok {
		t.Fatal("expired session should not resolve")
	}
	// revoke
	if err := s.RevokeWebSession(ctx, hash, now); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, ok, _ := s.ResolveWebSession(ctx, hash, now); ok {
		t.Fatal("revoked session should not resolve")
	}
}

func TestAudit_InsertAndList(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	// admin_audit_log.actor_user_id is NOT NULL REFERENCES bot_users(id),
	// so seed a bot_user and use its id as the actor.
	actor, err := s.CreateBotUser(ctx, 777, "Admin", "admin@x.io", "admin")
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	if err := s.InsertAuditEntry(ctx, AuditEntry{
		ActorUserID: actor.ID, ActorTelegramID: 777, ActorEmail: "admin@x.io",
		ActorKind: "bot", Action: "meeting.create",
		TargetKind: "meeting", TargetID: "m1",
	}); err != nil {
		t.Fatalf("insert audit: %v", err)
	}
	entries, err := s.ListAuditEntries(ctx, AuditFilter{Limit: 10})
	if err != nil || len(entries) != 1 || entries[0].Action != "meeting.create" {
		t.Fatalf("list audit: %v / %+v", err, entries)
	}
}

func TestReminderClaim_Idempotent(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	org := seedOrg(t, s)
	m := seedMeeting(t, s, org)
	first, err := s.TryClaimReminder(ctx, m.ID, 999, 15)
	if err != nil || !first {
		t.Fatalf("first claim should succeed: %v ok=%v", err, first)
	}
	second, err := s.TryClaimReminder(ctx, m.ID, 999, 15)
	if err != nil || second {
		t.Fatalf("second identical claim should be false: %v ok=%v", err, second)
	}
}
```

(Confirmed: `InsertAuditEntry` defaults empty `Details` to `{}`, but `actor_user_id` is `NOT NULL REFERENCES bot_users(id)` — hence the seeded `actor.ID` above. `reminder` test relies on the `meeting_reminders` PK `(meeting_id, telegram_id, offset_minutes)` for `ON CONFLICT DO NOTHING` idempotency, with `meeting_id` FK to a real seeded meeting.)

- [ ] **Step 3: Run all repo tests**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -v`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/backend/internal/infrastructure/persistence/postgres/org_user_repo_test.go \
  apps/backend/internal/infrastructure/persistence/postgres/bot_session_audit_repo_test.go
git commit -m "$(cat <<'EOF'
test(postgres): orgs/members, bot_users, web sessions, audit, reminder claim

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Full-suite + lint gate verification

**Files:** none (verification only)

- [ ] **Step 1: Full backend suite**

Run: `cd apps/backend && env -u GOROOT go test ./...`
Expected: all packages pass (the 4 pre-existing + domain + postgres). Container boots once for the postgres package.

- [ ] **Step 2: Lint stays clean (WS1 gate)**

Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...`
Expected: `0 issues.` (new test files must pass the same lint gate.)

- [ ] **Step 3: Confirm clean tree**

Run: `git status --short`
Expected: only the human's pre-existing items, if any. No stray files.

- [ ] **Step 4: Post-push CI observation (informational)**

After the human pushes, the WS1 CI gate runs `go test ./...` on the runner (which has Docker), so the new repo tests execute in CI. Confirm the Actions run is green (`gh run watch`). If the testcontainers boot fails in CI (e.g. Docker-in-Docker quirk), that surfaces here and becomes the next fix.

---

## Notes on execution order
Task 1 (harness) must come first — Tasks 3 and 4 import `pgtest`. Task 2 (domain) is independent and Docker-free. Tasks 3 and 4 both add files to the postgres test package and share `main_test.go`'s `TestMain`/helpers (created in Task 3), so Task 3 precedes Task 4. Task 5 verifies the whole.
