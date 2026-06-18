package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestGoogleReader_BusyTimes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendars":{"a@x.com":{"busy":[{"start":"2026-06-20T09:00:00Z","end":"2026-06-20T09:30:00Z"}]}}}`))
	}))
	defer srv.Close()
	svc, err := calendar.NewService(context.Background(), option.WithEndpoint(srv.URL), option.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	r := newGoogleReader(svc)
	busy, err := r.BusyTimes(context.Background(), []string{"a@x.com"},
		time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(busy["a@x.com"]) != 1 {
		t.Fatalf("expected 1 busy interval, got %v", busy)
	}
	if !busy["a@x.com"][0].Start.Equal(time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("bad start: %v", busy["a@x.com"][0].Start)
	}
}
