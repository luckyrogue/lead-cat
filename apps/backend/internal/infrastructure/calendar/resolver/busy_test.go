package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type fakeBusyReader struct{ tag string }

func (fakeBusyReader) BusyTimes(context.Context, []string, time.Time, time.Time) (map[string][]docalendar.Interval, error) {
	return nil, nil
}

type msBusyService struct{}

func (msBusyService) CreateEvent(context.Context, docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	return docalendar.CalendarResult{}, nil
}
func (msBusyService) UpdateEvent(context.Context, string, docalendar.CalendarEvent) error { return nil }
func (msBusyService) UpdateAttendees(context.Context, string, []string) error             { return nil }
func (msBusyService) DeleteEvent(context.Context, string) error                           { return nil }
func (msBusyService) BusyTimes(context.Context, []string, time.Time, time.Time) (map[string][]docalendar.Interval, error) {
	return nil, nil
}

type fakeGoogleRF struct{ ok bool }

func (f fakeGoogleRF) For(context.Context, model.CalendarConnection) (docalendar.BusyReader, bool) {
	return fakeBusyReader{tag: "google"}, f.ok
}

type fakeMSRF struct{ ok bool }

func (f fakeMSRF) For(context.Context, model.CalendarConnection) (docalendar.Service, bool) {
	return msBusyService{}, f.ok
}

func TestReaderFor_Microsoft(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "microsoft", UpdatedAt: time.Now()}}}
	r := NewBusyResolver(lister, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	rd, ok := r.ReaderFor(context.Background(), "u@x.com")
	if !ok || rd == nil {
		t.Fatal("expected MS reader")
	}
}

func TestReaderFor_Google(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "google", UpdatedAt: time.Now()}}}
	r := NewBusyResolver(lister, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	rd, ok := r.ReaderFor(context.Background(), "u@x.com")
	if !ok || rd.(fakeBusyReader).tag != "google" {
		t.Fatalf("expected google reader, got %v ok=%v", rd, ok)
	}
}

func TestReaderFor_None(t *testing.T) {
	r := NewBusyResolver(fakeLister{}, fakeGoogleRF{ok: true}, fakeMSRF{ok: true})
	if _, ok := r.ReaderFor(context.Background(), "u@x.com"); ok {
		t.Fatal("expected no reader when no connections")
	}
}
