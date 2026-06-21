package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type fakeBusyResolver struct {
	byEmail map[string]docalendar.BusyReader
}

func (f fakeBusyResolver) ReaderFor(_ context.Context, email string) (docalendar.BusyReader, bool) {
	r, ok := f.byEmail[email]
	return r, ok
}

type stubReader struct {
	busy map[string][]docalendar.Interval
	err  error
}

func (s stubReader) BusyTimes(_ context.Context, _ []string, _, _ time.Time) (map[string][]docalendar.Interval, error) {
	return s.busy, s.err
}

func TestGatherExternalBusy_UnionAndBestEffort(t *testing.T) {
	from := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	organizer := stubReader{busy: map[string][]docalendar.Interval{"a@x.com": {{Start: from, End: to}}}}
	ownB := stubReader{err: errors.New("boom")}
	s := &Services{Busy: fakeBusyResolver{byEmail: map[string]docalendar.BusyReader{
		"org@x.com": organizer, "b@x.com": ownB,
	}}}
	got := s.gatherExternalBusy(context.Background(), "org@x.com", []string{"a@x.com", "b@x.com"}, from, to)
	if len(got["a@x.com"]) != 1 {
		t.Fatalf("expected a busy from organizer view, got %v", got)
	}
}

func TestGatherExternalBusy_NilResolver(t *testing.T) {
	s := &Services{}
	got := s.gatherExternalBusy(context.Background(), "org@x.com", []string{"a@x.com"}, time.Now(), time.Now().Add(time.Hour))
	if len(got) != 0 {
		t.Fatalf("expected empty with nil resolver, got %v", got)
	}
}

type fakeRepo struct {
	Repository
	overlapping   []model.Meeting
	participants  map[uuid.UUID][]model.MeetingParticipant
	filterKnownFn func(context.Context, []string) ([]string, error)
}

func (r *fakeRepo) ListMeetingsOverlapping(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
	return r.overlapping, nil
}

func (r *fakeRepo) ListParticipants(_ context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error) {
	return r.participants[meetingID], nil
}

func (r *fakeRepo) GetUserByID(_ context.Context, _ uuid.UUID) (model.User, error) {
	return model.User{}, errors.New("no user")
}

func (r *fakeRepo) SearchEmployeesGlobal(_ context.Context, _ string) ([]model.Employee, error) {
	return nil, nil
}

func (r *fakeRepo) FilterKnownEmployeeEmails(ctx context.Context, emails []string) ([]string, error) {
	if r.filterKnownFn != nil {
		return r.filterKnownFn(ctx, emails)
	}
	return emails, nil
}

func TestMeetingConflicts_ExternalBusy(t *testing.T) {
	start := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	reader := stubReader{busy: map[string][]docalendar.Interval{
		"a@x.com": {{Start: start.Add(30 * time.Minute), End: end.Add(30 * time.Minute)}},
	}}
	s := &Services{
		Store: &fakeRepo{},
		Busy: fakeBusyResolver{byEmail: map[string]docalendar.BusyReader{
			"a@x.com": reader,
		}},
	}
	got, err := s.MeetingConflicts(context.Background(), "", []string{"a@x.com"}, start, end, uuid.Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one external conflict, got %v", got)
	}
	if got[0].Email != "a@x.com" || got[0].MeetingName != "" {
		t.Fatalf("expected empty MeetingName for external conflict, got %+v", got[0])
	}
}

func TestMeetingConflicts_NilBusyNoPanic(t *testing.T) {
	start := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	s := &Services{Store: &fakeRepo{}}
	got, err := s.MeetingConflicts(context.Background(), "org@x.com", []string{"a@x.com"}, start, end, uuid.Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no conflicts with nil Busy, got %v", got)
	}
}

func TestMeetingConflicts_UnknownParticipantRejected(t *testing.T) {
	start := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	repo := &fakeRepo{}
	repo.filterKnownFn = func(_ context.Context, emails []string) ([]string, error) {
		out := make([]string, 0, len(emails))
		for _, e := range emails {
			if e == "a@x.com" {
				out = append(out, e)
			}
		}
		return out, nil
	}
	s := &Services{Store: repo}
	_, err := s.MeetingConflicts(context.Background(), "", []string{"a@x.com", "stranger@evil.com"}, start, end, uuid.Nil)
	if !errors.Is(err, ErrUnknownParticipant) {
		t.Fatalf("expected ErrUnknownParticipant, got %v", err)
	}
}

func TestFreeSlots_ExternalBusyShrinksSlot(t *testing.T) {
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, almatyLoc)
	to := from.AddDate(0, 0, 1)
	busyStart := time.Date(2026, 6, 22, 9, 0, 0, 0, almatyLoc)
	busyEnd := time.Date(2026, 6, 22, 12, 0, 0, 0, almatyLoc)
	reader := stubReader{busy: map[string][]docalendar.Interval{
		"a@x.com": {{Start: busyStart, End: busyEnd}},
	}}
	withExt := &Services{
		Store: &fakeRepo{},
		Busy: fakeBusyResolver{byEmail: map[string]docalendar.BusyReader{
			"a@x.com": reader,
		}},
	}
	dbOnly := &Services{Store: &fakeRepo{}}

	gotExt, err := withExt.FreeSlots(context.Background(), "", []string{"a@x.com"}, from, to, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotDB, err := dbOnly.FreeSlots(context.Background(), "", []string{"a@x.com"}, from, to, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalFreeMins(gotExt) >= totalFreeMins(gotDB) {
		t.Fatalf("expected external busy to shrink free time: ext=%d db=%d", totalFreeMins(gotExt), totalFreeMins(gotDB))
	}
	for _, sl := range gotExt {
		if sl.Start.Before(busyEnd) && busyStart.Before(sl.End) {
			t.Fatalf("free slot overlaps external busy: %+v", sl)
		}
	}
}

func totalFreeMins(slots []FreeSlot) int {
	total := 0
	for _, s := range slots {
		total += s.Mins
	}
	return total
}
