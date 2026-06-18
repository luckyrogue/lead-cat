package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestCalendarConnection_UpsertGetDelete(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()

	conn := pg.CalendarConnection{
		Email: "Alice@Example.com", Provider: "google",
		AccessToken: "at-1", RefreshToken: "rt-1",
		Expiry: time.Now().Add(time.Hour).UTC().Truncate(time.Second), Scopes: "calendar.events",
	}
	if err := store.UpsertCalendarConnection(ctx, conn); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := store.GetCalendarConnection(ctx, "alice@example.com", "google")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccessToken != "at-1" || got.RefreshToken != "rt-1" {
		t.Fatalf("tokens not round-tripped: %+v", got)
	}
	if got.Scopes != "calendar.events" {
		t.Fatalf("scopes: %q", got.Scopes)
	}

	conn.AccessToken = "at-2"
	if err := store.UpsertCalendarConnection(ctx, conn); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	got, _ = store.GetCalendarConnection(ctx, "alice@example.com", "google")
	if got.AccessToken != "at-2" {
		t.Fatalf("upsert did not overwrite: %q", got.AccessToken)
	}

	list, err := store.ListCalendarConnections(ctx, "alice@example.com")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}

	if err := store.DeleteCalendarConnection(ctx, "alice@example.com", "google"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetCalendarConnection(ctx, "alice@example.com", "google"); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound after delete, got %v", err)
	}
}
