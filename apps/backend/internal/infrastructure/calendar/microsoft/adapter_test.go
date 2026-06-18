package microsoft

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func newTestAdapter(t *testing.T, h http.HandlerFunc) (*adapter, *httptest.Server) {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newAdapter(srv.Client(), srv.URL), srv
}

func TestCreateEvent_Teams(t *testing.T) {
	var gotPath, gotBody string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"evt1","onlineMeeting":{"joinUrl":"https://teams.microsoft.com/l/xyz"}}`))
	})
	res, err := a.CreateEvent(context.Background(), docalendar.CalendarEvent{
		Title: "Sync", Description: "d",
		Start:          time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 20, 9, 30, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.EventID != "evt1" || res.MeetLink != "https://teams.microsoft.com/l/xyz" {
		t.Fatalf("bad result: %+v", res)
	}
	if gotPath != "/me/events" {
		t.Errorf("path=%q", gotPath)
	}
	for _, want := range []string{`"isOnlineMeeting":true`, `"teamsForBusiness"`, `"a@x.com"`, `"Sync"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body missing %q: %s", want, gotBody)
		}
	}
}

func TestCreateEvent_GraphError(t *testing.T) {
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":{"code":"ErrorAccessDenied","message":"no"}}`))
	})
	if _, err := a.CreateEvent(context.Background(), docalendar.CalendarEvent{}); err == nil {
		t.Fatal("expected error on 403")
	}
}

func TestUpdateEvent_Patch(t *testing.T) {
	var m, p string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) { m, p = r.Method, r.URL.Path })
	if err := a.UpdateEvent(context.Background(), "evt1", docalendar.CalendarEvent{Title: "X",
		Start: time.Now().UTC(), End: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if m != http.MethodPatch || p != "/me/events/evt1" {
		t.Fatalf("got %s %s", m, p)
	}
}

func TestDeleteEvent(t *testing.T) {
	var m, p string
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) { m, p = r.Method, r.URL.Path; w.WriteHeader(204) })
	if err := a.DeleteEvent(context.Background(), "evt1"); err != nil {
		t.Fatal(err)
	}
	if m != http.MethodDelete || p != "/me/events/evt1" {
		t.Fatalf("got %s %s", m, p)
	}
}

func TestBusyTimes(t *testing.T) {
	a, _ := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendar/getSchedule" {
			t.Errorf("path=%q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"value":[{"scheduleId":"a@x.com","scheduleItems":[{"status":"busy","start":{"dateTime":"2026-06-20T09:00:00.0000000","timeZone":"UTC"},"end":{"dateTime":"2026-06-20T09:30:00.0000000","timeZone":"UTC"}}]}]}`))
	})
	busy, err := a.BusyTimes(context.Background(), []string{"a@x.com"},
		time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(busy["a@x.com"]) != 1 {
		t.Fatalf("expected 1 busy block, got %v", busy)
	}
}
