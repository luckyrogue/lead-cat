package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type bookingFakeStore struct {
	Repository
	user        model.PlatformUser
	userOK      bool
	userErr     error
	overlapping []model.Meeting
	overlapErr  error
}

func (f *bookingFakeStore) GetPlatformUserByID(_ context.Context, _ uuid.UUID) (model.PlatformUser, bool, error) {
	return f.user, f.userOK, f.userErr
}

func (f *bookingFakeStore) ListMeetingsOverlapping(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
	return f.overlapping, f.overlapErr
}

func monFriEvent() model.BookingEventType {
	return model.BookingEventType{
		HostUserID:       uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"),
		OrganizationID:   uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002"),
		Title:            "Intro Call",
		DurationMins:     30,
		Timezone:         "Asia/Almaty",
		AvailWeekdays:    []int{1, 2, 3, 4, 5},
		AvailStartMinute: 540,
		AvailEndMinute:   1020,
		Active:           true,
	}
}

func hostStore() *bookingFakeStore {
	return &bookingFakeStore{
		user:   model.PlatformUser{Email: "host@example.com"},
		userOK: true,
	}
}

func TestBookingAvailability_WeekdayHasSlots(t *testing.T) {
	s := &Services{Store: hostStore()}
	et := monFriEvent()
	// Monday 2026-06-22 00:00 Almaty = 2026-06-21 19:00 UTC
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almatyLoc)
	to := from.AddDate(0, 0, 1)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected slots on a Mon-Fri weekday, got none")
	}
	// All slots should be at least DurationMins long
	for _, sl := range slots {
		if sl.End.Sub(sl.Start) < 30*time.Minute {
			t.Errorf("slot too short (< 30m): %v – %v", sl.Start, sl.End)
		}
	}
}

func TestBookingAvailability_BusyMeetingExcludesSlot(t *testing.T) {
	almaty, _ := time.LoadLocation("Asia/Almaty")
	busyStart := time.Date(2026, 6, 22, 9, 0, 0, 0, almaty)
	busyEnd := time.Date(2026, 6, 22, 9, 30, 0, 0, almaty)
	store := hostStore()
	store.overlapping = []model.Meeting{
		{StartsAt: busyStart, EndsAt: busyEnd},
	}
	s := &Services{Store: store}
	et := monFriEvent()
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almaty)
	to := from.AddDate(0, 0, 1)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sl := range slots {
		if sl.Start.Before(busyEnd) && busyStart.Before(sl.End) {
			t.Errorf("slot overlaps busy block: %v – %v", sl.Start, sl.End)
		}
	}
	// First slot should start at 09:30
	if len(slots) == 0 {
		t.Fatal("expected some slots after busy block")
	}
	first := slots[0].Start.In(almaty)
	if first.Hour() != 9 || first.Minute() != 30 {
		t.Errorf("expected first slot at 09:30, got %v", first)
	}
}

func TestBookingAvailability_NonAllowedWeekdayYieldsNone(t *testing.T) {
	s := &Services{Store: hostStore()}
	et := monFriEvent()
	et.AvailWeekdays = []int{6, 7} // Sat, Sun only
	// Monday 2026-06-22
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almatyLoc)
	to := from.AddDate(0, 0, 1)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected no slots for weekday-excluded day, got %d", len(slots))
	}
}

func TestBookingAvailability_PastSlotsDropped(t *testing.T) {
	almaty, _ := time.LoadLocation("Asia/Almaty")
	s := &Services{Store: hostStore()}
	et := monFriEvent()
	// from is set to 09:45 on a Monday — so the 09:00 and 09:30 slots are in the past
	from := time.Date(2026, 6, 22, 9, 45, 0, 0, almaty)
	to := time.Date(2026, 6, 22, 23, 59, 0, 0, almaty)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sl := range slots {
		if !sl.Start.After(from) {
			t.Errorf("past slot not dropped: %v (from=%v)", sl.Start, from)
		}
	}
}

func TestBookingAvailability_ZeroDuration_ReturnsNil(t *testing.T) {
	s := &Services{Store: hostStore()}
	et := monFriEvent()
	et.DurationMins = 0
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almatyLoc)
	to := from.AddDate(0, 0, 1)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slots != nil {
		t.Fatalf("expected nil for zero-duration event, got %v", slots)
	}
}

func TestBookingAvailability_HostNotFound_ReturnsNil(t *testing.T) {
	store := &bookingFakeStore{
		user:   model.PlatformUser{},
		userOK: false,
	}
	s := &Services{Store: store}
	et := monFriEvent()
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almatyLoc)
	to := from.AddDate(0, 0, 1)

	slots, err := s.BookingAvailability(context.Background(), et, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slots != nil {
		t.Fatalf("expected nil when host not found, got %v", slots)
	}
}
