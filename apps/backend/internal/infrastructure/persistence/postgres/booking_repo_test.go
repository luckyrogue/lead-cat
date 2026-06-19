package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestBookingEventType_CRUD(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, _ := seedOrg(t, store)
	hostID := seedUser(t, store, "host@x.com")

	et := pg.BookingEventType{
		HostUserID: hostID, OrganizationID: orgID, Slug: "intro-abc123",
		Title: "Intro call", DurationMins: 30, Active: true, Timezone: "Asia/Almaty",
		AvailWeekdays: []int{1, 2, 3, 4, 5}, AvailStartMinute: 540, AvailEndMinute: 1020,
	}
	created, err := store.CreateBookingEventType(ctx, et)
	if err != nil || created.ID == uuid.Nil {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := store.GetBookingEventTypeBySlug(ctx, "intro-abc123")
	if err != nil || got.Title != "Intro call" || len(got.AvailWeekdays) != 5 {
		t.Fatalf("by slug: %v %+v", err, got)
	}
	list, err := store.ListBookingEventTypesForUser(ctx, hostID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	created.Title = "Intro (30m)"
	created.AvailEndMinute = 1080
	if err := store.UpdateBookingEventType(ctx, created); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = store.GetBookingEventType(ctx, created.ID)
	if got.Title != "Intro (30m)" || got.AvailEndMinute != 1080 {
		t.Fatalf("update not applied: %+v", got)
	}
	if err := store.DeleteBookingEventType(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetBookingEventType(ctx, created.ID); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound after delete, got %v", err)
	}
}
