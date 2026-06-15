package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

// SeriesReshape is the plan for changing a series' end date: spans to append and
// occurrence IDs to cancel.
type SeriesReshape struct {
	Create    []meeting.Span
	CancelIDs []uuid.UUID
}

// planSeriesReshape decides the delta for moving a series' end to newUntil.
// occs are the current scheduled occurrences (any order); candidate is the full
// on-cadence span list regenerated up to newUntil. Extend appends only spans
// after the latest existing occurrence (never resurrecting skipped gaps); trim
// cancels scheduled occurrences that start after the newUntil day.
func planSeriesReshape(occs []model.Meeting, candidate []meeting.Span, newUntil time.Time, loc *time.Location) SeriesReshape {
	var latest time.Time
	for _, o := range occs {
		if o.StartsAt.After(latest) {
			latest = o.StartsAt
		}
	}
	cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	var out SeriesReshape
	for _, sp := range candidate {
		if sp.Start.After(latest) {
			out.Create = append(out.Create, sp)
		}
	}
	for _, o := range occs {
		if !o.StartsAt.Before(cutoff) {
			out.CancelIDs = append(out.CancelIDs, o.ID)
		}
	}
	return out
}
