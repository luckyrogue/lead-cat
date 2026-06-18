package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
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

func newStore() *pg.Store {
	cipher, err := crypto.NewTokenCipher("pgtest-fixed-32-byte-key-for-tests!")
	if err != nil {
		panic("newStore: cipher: " + err.Error())
	}
	return pg.New(testDB.Pool, zap.NewNop(), cipher)
}

func seedOrg(t *testing.T, s *pg.Store) uuid.UUID {
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

func seedMeeting(t *testing.T, s *pg.Store, orgID uuid.UUID) pg.Meeting {
	t.Helper()
	m, err := s.CreateMeeting(context.Background(), pg.Meeting{
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
