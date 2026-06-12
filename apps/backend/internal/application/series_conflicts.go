package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

type OccurrenceConflicts struct {
	Span      meeting.Span
	Conflicts []Conflict
}

func expandSeriesSpans(start, end time.Time, r meeting.Recurrence, days []int, until time.Time) ([]meeting.Span, error) {
	return meeting.Occurrences(start, end, r, days, until)
}

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
