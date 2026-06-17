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

// DockerAvailable reports whether a usable Docker endpoint exists.
func DockerAvailable() bool {
	p, err := testcontainers.NewDockerProvider()
	if err != nil {
		return false
	}
	defer func() { _ = p.Close() }()
	return p.Health(context.Background()) == nil
}

// Start boots Postgres, applies migrations, and returns a connected DB.
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

// migrationsDir resolves apps/backend/migrations from this file's location.
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
