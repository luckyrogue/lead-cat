package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
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
	ms := []pg.Meeting{
		{OrganizationID: org, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "d1",
			StartsAt: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
			SeriesID: &series, RecurrenceUntil: &until},
		{OrganizationID: org, Dept: "Eng", Type: "Sync", Host: "Mia", Recurrence: "daily", Name: "d2",
			StartsAt: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 6, 2, 10, 30, 0, 0, time.UTC),
			SeriesID: &series, RecurrenceUntil: &until},
	}
	ps := []pg.MeetingParticipant{{Email: "a@x.io"}, {Email: "b@x.io"}}
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
	ms := []pg.Meeting{
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
	if err := s.AddParticipants(ctx, m.ID, []pg.MeetingParticipant{{Email: "a@x.io"}, {Email: "b@x.io"}}); err != nil {
		t.Fatalf("add: %v", err)
	}
	_ = s.AddParticipants(ctx, m.ID, []pg.MeetingParticipant{{Email: "a@x.io"}})
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
