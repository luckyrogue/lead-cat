package application

import (
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
)

func TestExpandSeriesSpans_Once(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, loc).UTC()
	end := time.Date(2026, 6, 1, 11, 0, 0, 0, loc).UTC()
	spans, err := expandSeriesSpans(start, end, meeting.Once, nil, time.Time{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("once should expand to 1, got %d", len(spans))
	}
}

func TestExpandSeriesSpans_CustomThreeWeeks(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, loc).UTC() // Mon
	end := time.Date(2026, 6, 1, 11, 0, 0, 0, loc).UTC()
	until := time.Date(2026, 6, 21, 0, 0, 0, 0, loc)
	spans, err := expandSeriesSpans(start, end, meeting.Custom, []int{1, 3, 5}, until)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(spans) != 9 {
		t.Fatalf("expected 9 spans, got %d", len(spans))
	}
}
