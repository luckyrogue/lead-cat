package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
)

// OccurrenceConflicts is the per-occurrence result of a series conflict check.
type OccurrenceConflicts struct {
	Span      meeting.Span
	Conflicts []Conflict
}

// expandSeriesSpans is the pure expansion helper — a thin wrapper over the
// domain Occurrences function so tests can hit the math without a Services
// receiver. Kept here for locality with MeetingSeriesConflicts.
func expandSeriesSpans(start, end time.Time, r meeting.Recurrence, days []int, until time.Time) ([]meeting.Span, error) {
	return meeting.Occurrences(start, end, r, days, until)
}

// MeetingSeriesConflicts expands a hypothetical series and runs the existing
// per-occurrence conflict check against each. Only occurrences with ≥1 conflict
// are returned. Spans are in chronological order.
func (s *Services) MeetingSeriesConflicts(ctx context.Context, emails []string, start, end time.Time, r meeting.Recurrence, days []int, until time.Time) ([]OccurrenceConflicts, error) {
	spans, err := expandSeriesSpans(start, end, r, days, until)
	if err != nil {
		return nil, err
	}
	out := make([]OccurrenceConflicts, 0, len(spans))
	for _, sp := range spans {
		cs, err := s.MeetingConflicts(ctx, emails, sp.Start, sp.End, uuid.Nil)
		if err != nil {
			return nil, err
		}
		if len(cs) == 0 {
			continue
		}
		out = append(out, OccurrenceConflicts{Span: sp, Conflicts: cs})
	}
	return out, nil
}
