package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

type SeriesReshape struct {
	Create    []meeting.Span
	CancelIDs []uuid.UUID
}

func planSeriesReshape(occs []model.Meeting, existingStarts []time.Time, candidate []meeting.Span, newUntil time.Time, loc *time.Location) SeriesReshape {
	var latest time.Time
	for _, o := range occs {
		if o.StartsAt.After(latest) {
			latest = o.StartsAt
		}
	}
	cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	var out SeriesReshape
	for _, sp := range candidate {
		if !sp.Start.After(latest) {
			continue
		}
		if startExists(existingStarts, sp.Start) {
			continue 
		}
		out.Create = append(out.Create, sp)
	}
	for _, o := range occs {
		if !o.StartsAt.Before(cutoff) {
			out.CancelIDs = append(out.CancelIDs, o.ID)
		}
	}
	return out
}

func startExists(starts []time.Time, t time.Time) bool {
	for _, s := range starts {
		if s.Equal(t) {
			return true
		}
	}
	return false
}
