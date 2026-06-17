package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type capturedReq struct {
	method string
	path   string
	query  url.Values
	body   map[string]any
}

func newTestAdapter(t *testing.T, status int, respJSON string) (*adapter, *capturedReq) {
	t.Helper()
	cap := &capturedReq{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			cap.method, cap.path = r.Method, r.URL.Path
			w.WriteHeader(204)
			return
		}
		cap.method, cap.path, cap.query = r.Method, r.URL.Path, r.URL.Query()
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			cap.body = map[string]any{}
			_ = json.Unmarshal(b, &cap.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if respJSON != "" {
			_, _ = io.WriteString(w, respJSON)
		}
	}))
	t.Cleanup(srv.Close)

	svc, err := calendar.NewService(context.Background(), option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return &adapter{svc: svc, calendarID: "primary"}, cap
}

func sampleEvent() docalendar.CalendarEvent {
	return docalendar.CalendarEvent{
		Title: "Sync", Description: "d",
		Start:          time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.io", "b@x.io"},
	}
}

func TestCreateEvent_HangoutLink(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1","hangoutLink":"https://meet.google.com/abc"}`)
	res, err := a.CreateEvent(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.EventID != "evt1" || res.MeetLink != "https://meet.google.com/abc" {
		t.Fatalf("result: %+v", res)
	}
	if cap.method != "POST" || cap.path != "/calendars/primary/events" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
	if cap.query.Get("conferenceDataVersion") != "1" {
		t.Fatalf("conferenceDataVersion = %q", cap.query.Get("conferenceDataVersion"))
	}
}

func TestCreateEvent_ConferenceEntryPoint(t *testing.T) {
	a, _ := newTestAdapter(t, 200, `{"id":"evt2","conferenceData":{"entryPoints":[{"entryPointType":"phone","uri":"tel:+1"},{"entryPointType":"video","uri":"https://meet.google.com/xyz"}]}}`)
	res, err := a.CreateEvent(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.MeetLink != "https://meet.google.com/xyz" {
		t.Fatalf("want video entrypoint uri, got %q", res.MeetLink)
	}
}

func TestCreateEvent_APIError(t *testing.T) {
	a, _ := newTestAdapter(t, 500, `{"error":{"code":500,"message":"boom"}}`)
	if _, err := a.CreateEvent(context.Background(), sampleEvent()); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestUpdateEvent_PatchAllUpdates(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1"}`)
	if err := a.UpdateEvent(context.Background(), "evt1", sampleEvent()); err != nil {
		t.Fatalf("update: %v", err)
	}
	if cap.method != "PATCH" || cap.path != "/calendars/primary/events/evt1" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
	if cap.query.Get("sendUpdates") != "all" {
		t.Fatalf("sendUpdates = %q", cap.query.Get("sendUpdates"))
	}
	if cap.body["summary"] != "Sync" {
		t.Fatalf("body summary = %v", cap.body["summary"])
	}
}

func TestUpdateAttendees_PatchWithAttendees(t *testing.T) {
	a, cap := newTestAdapter(t, 200, `{"id":"evt1"}`)
	if err := a.UpdateAttendees(context.Background(), "evt1", []string{"c@x.io"}); err != nil {
		t.Fatalf("update attendees: %v", err)
	}
	if cap.method != "PATCH" || cap.query.Get("sendUpdates") != "all" {
		t.Fatalf("request: %s sendUpdates=%q", cap.method, cap.query.Get("sendUpdates"))
	}
	att, ok := cap.body["attendees"].([]any)
	if !ok || len(att) != 1 {
		t.Fatalf("attendees body = %v", cap.body["attendees"])
	}
}

func TestDeleteEvent(t *testing.T) {
	a, cap := newTestAdapter(t, 200, ``)
	if err := a.DeleteEvent(context.Background(), "evt1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if cap.method != "DELETE" || cap.path != "/calendars/primary/events/evt1" {
		t.Fatalf("request: %s %s", cap.method, cap.path)
	}
}
