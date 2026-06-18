package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type fakeLister struct{ conns []model.CalendarConnection }

func (f fakeLister) ListCalendarConnections(_ context.Context, _ string) ([]model.CalendarConnection, error) {
	return f.conns, nil
}

type stubService struct{ tag string }

func (stubService) CreateEvent(context.Context, docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	return docalendar.CalendarResult{}, nil
}
func (stubService) UpdateEvent(context.Context, string, docalendar.CalendarEvent) error { return nil }
func (stubService) UpdateAttendees(context.Context, string, []string) error             { return nil }
func (stubService) DeleteEvent(context.Context, string) error                           { return nil }

type fakeGoogle struct{ called bool }

func (g *fakeGoogle) For(context.Context, uuid.UUID, string) (docalendar.Service, error) {
	g.called = true
	return stubService{tag: "google"}, nil
}

type fakeMS struct{ built bool }

func (m *fakeMS) For(context.Context, model.CalendarConnection) (docalendar.Service, bool) {
	m.built = true
	return stubService{tag: "ms"}, true
}

func TestResolve_MicrosoftWins_MostRecent(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()
	lister := fakeLister{conns: []model.CalendarConnection{
		{Provider: "google", UpdatedAt: old}, {Provider: "microsoft", UpdatedAt: newer},
	}}
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(lister, g, m)
	svc, err := r.For(context.Background(), uuid.New(), "u@x.com")
	if err != nil || svc.(stubService).tag != "ms" {
		t.Fatalf("expected ms, got %+v err=%v", svc, err)
	}
	if g.called {
		t.Error("google should not be called when MS is most recent")
	}
}

func TestResolve_GoogleDelegate_WhenNoMS(t *testing.T) {
	lister := fakeLister{conns: []model.CalendarConnection{{Provider: "google", UpdatedAt: time.Now()}}}
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(lister, g, m)
	svc, err := r.For(context.Background(), uuid.New(), "u@x.com")
	if err != nil || svc.(stubService).tag != "google" || !g.called {
		t.Fatalf("expected google delegate, got %+v err=%v", svc, err)
	}
}

func TestResolve_NoConnections_Delegates(t *testing.T) {
	g, m := &fakeGoogle{}, &fakeMS{}
	r := New(fakeLister{}, g, m)
	if _, err := r.For(context.Background(), uuid.New(), "u@x.com"); err != nil || !g.called {
		t.Fatalf("expected google/SA delegate, err=%v called=%v", err, g.called)
	}
}
