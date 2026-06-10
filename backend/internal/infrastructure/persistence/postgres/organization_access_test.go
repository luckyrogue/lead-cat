package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func TestUserCanAccessOrganization_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := New(pool, zap.NewNop())

	sub := "test-" + uuid.New().String()
	u, err := store.UpsertUser(ctx, sub, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	slug := "test-" + uuid.New().String()[:8]
	org, err := store.CreateOrganization(ctx, "Test Org", slug, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	strangerSub := "stranger-" + uuid.New().String()
	if _, err := store.UpsertUser(ctx, strangerSub, "stranger@example.com"); err != nil {
		t.Fatal(err)
	}
	stranger, err := store.GetUserBySub(ctx, strangerSub)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := store.UserCanAccessOrganization(ctx, u.ID, org.ID)
	if err != nil || !ok {
		t.Fatalf("owner access: ok=%v err=%v", ok, err)
	}

	ok, err = store.UserCanAccessOrganization(ctx, stranger.ID, org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("stranger should not access organization")
	}
}
