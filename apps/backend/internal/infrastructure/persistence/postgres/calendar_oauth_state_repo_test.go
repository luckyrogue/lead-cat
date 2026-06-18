package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestCalendarOAuthState_CreateConsume(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()

	st := pg.CalendarOAuthState{
		State: "st-123", Email: "bob@example.com", Provider: "google",
		Verifier: "ver-xyz", ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := store.CreateCalendarOAuthState(ctx, st); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.ConsumeCalendarOAuthState(ctx, "st-123")
	if err != nil || got.Email != "bob@example.com" || got.Verifier != "ver-xyz" {
		t.Fatalf("consume: %v %+v", err, got)
	}
	if _, err := store.ConsumeCalendarOAuthState(ctx, "st-123"); !model.IsNotFound(err) {
		t.Fatalf("second consume should be IsNotFound, got %v", err)
	}

	expired := pg.CalendarOAuthState{State: "st-exp", Email: "c@x.com", Provider: "google", Verifier: "v", ExpiresAt: time.Now().Add(-time.Minute)}
	_ = store.CreateCalendarOAuthState(ctx, expired)
	if _, err := store.ConsumeCalendarOAuthState(ctx, "st-exp"); !model.IsNotFound(err) {
		t.Fatalf("expired consume should be IsNotFound, got %v", err)
	}
}
