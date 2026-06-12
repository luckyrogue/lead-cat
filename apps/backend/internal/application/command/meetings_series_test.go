package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

type fakeCal struct {
	mu      sync.Mutex
	created int
	deleted int
	failAt  int
}

func (f *fakeCal) CreateEvent(_ context.Context, _ docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created++
	if f.failAt != 0 && f.created == f.failAt {
		return docalendar.CalendarResult{}, errors.New("boom")
	}
	return docalendar.CalendarResult{EventID: "ev", MeetLink: "https://meet/x"}, nil
}
func (f *fakeCal) UpdateEvent(_ context.Context, _ string, _ docalendar.CalendarEvent) error { return nil }
func (f *fakeCal) UpdateAttendees(_ context.Context, _ string, _ []string) error          { return nil }
func (f *fakeCal) DeleteEvent(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted++
	return nil
}

func spans(n int) []meeting.Span {
	out := make([]meeting.Span, n)
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	for i := range out {
		s := base.AddDate(0, 0, 7*i)
		out[i] = meeting.Span{Start: s, End: s.Add(time.Hour)}
	}
	return out
}

func TestCreateSeriesEvents_OK(t *testing.T) {
	cal := &fakeCal{}
	names := []string{"a", "b", "c"}
	cmd := &Meetings{}
	evs, err := cmd.createSeriesEvents(context.Background(), cal, names, "desc", []string{"x@y"}, spans(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 || cal.created != 3 || cal.deleted != 0 {
		t.Fatalf("evs=%d created=%d deleted=%d", len(evs), cal.created, cal.deleted)
	}
	if evs[0].Name != "a" || evs[0].EventID != "ev" {
		t.Fatalf("bad event row: %+v", evs[0])
	}
}

func TestCreateSeriesEvents_CompensatesOnFailure(t *testing.T) {
	cal := &fakeCal{failAt: 3}
	names := []string{"a", "b", "c"}
	cmd := &Meetings{}
	_, err := cmd.createSeriesEvents(context.Background(), cal, names, "desc", nil, spans(3))
	if err == nil {
		t.Fatal("expected error")
	}
	if cal.created != 3 || cal.deleted != 2 {
		t.Fatalf("compensation: created=%d deleted=%d (want 3 created, 2 deleted)", cal.created, cal.deleted)
	}
}
