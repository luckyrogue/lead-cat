package application

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

func makeOcc(id uuid.UUID, start time.Time) model.Meeting {
	return model.Meeting{ID: id, StartsAt: start, EndsAt: start.Add(time.Hour)}
}

func TestPlanSeriesReshape_Extend(t *testing.T) {
	loc := time.UTC
	a, b := uuid.New(), uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 2, 9, 0, 0, 0, loc)
	occs := []model.Meeting{makeOcc(a, d1), makeOcc(b, d2)}
	candidate := []meeting.Span{
		{Start: d1, End: d1.Add(time.Hour)},
		{Start: d2, End: d2.Add(time.Hour)},
		{Start: d2.AddDate(0, 0, 1), End: d2.AddDate(0, 0, 1).Add(time.Hour)},
		{Start: d2.AddDate(0, 0, 2), End: d2.AddDate(0, 0, 2).Add(time.Hour)},
	}
	newUntil := time.Date(2026, 6, 4, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, []time.Time{d1, d2}, candidate, newUntil, loc)
	if len(r.Create) != 2 {
		t.Fatalf("create = %d, want 2", len(r.Create))
	}
	if len(r.CancelIDs) != 0 {
		t.Fatalf("cancel = %d, want 0", len(r.CancelIDs))
	}
}

func TestPlanSeriesReshape_ExtendSkipsGaps(t *testing.T) {
	loc := time.UTC
	a := uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	occs := []model.Meeting{makeOcc(a, d1)}
	candidate := []meeting.Span{
		{Start: d1, End: d1.Add(time.Hour)},
		{Start: d1.AddDate(0, 0, 1), End: d1.AddDate(0, 0, 1).Add(time.Hour)},
		{Start: d1.AddDate(0, 0, 2), End: d1.AddDate(0, 0, 2).Add(time.Hour)},
	}
	newUntil := time.Date(2026, 6, 3, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, []time.Time{d1}, candidate, newUntil, loc)
	if len(r.Create) != 2 || len(r.CancelIDs) != 0 {
		t.Fatalf("create=%d cancel=%d, want 2/0", len(r.Create), len(r.CancelIDs))
	}
}

func TestPlanSeriesReshape_Trim(t *testing.T) {
	loc := time.UTC
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 2, 9, 0, 0, 0, loc)
	d3 := time.Date(2026, 6, 3, 9, 0, 0, 0, loc)
	occs := []model.Meeting{makeOcc(a, d1), makeOcc(b, d2), makeOcc(c, d3)}
	candidate := []meeting.Span{{Start: d1, End: d1.Add(time.Hour)}}
	newUntil := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, []time.Time{d1, d2, d3}, candidate, newUntil, loc)
	if len(r.Create) != 0 {
		t.Fatalf("create = %d, want 0", len(r.Create))
	}
	if len(r.CancelIDs) != 2 {
		t.Fatalf("cancel = %d, want 2 (d2,d3)", len(r.CancelIDs))
	}
}

// Trim a series (cancelling its tail), then extend the end back over the
// cancelled dates: the cancelled slots must be neither resurrected nor
// duplicated — no new rows at starts that already exist in any status.
func TestPlanSeriesReshape_ExtendOverCancelledTail(t *testing.T) {
	loc := time.UTC
	a := uuid.New()
	d1 := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 2, 9, 0, 0, 0, loc)
	d3 := time.Date(2026, 6, 3, 9, 0, 0, 0, loc)
	// d2,d3 were cancelled by an earlier trim — only d1 remains scheduled,
	// but all three starts still exist as rows.
	occs := []model.Meeting{makeOcc(a, d1)}
	existing := []time.Time{d1, d2, d3}
	candidate := []meeting.Span{
		{Start: d1, End: d1.Add(time.Hour)},
		{Start: d2, End: d2.Add(time.Hour)},
		{Start: d3, End: d3.Add(time.Hour)},
	}
	newUntil := time.Date(2026, 6, 3, 0, 0, 0, 0, loc)
	r := planSeriesReshape(occs, existing, candidate, newUntil, loc)
	if len(r.Create) != 0 {
		t.Fatalf("create = %d, want 0 (cancelled slots must not be duplicated)", len(r.Create))
	}
	if len(r.CancelIDs) != 0 {
		t.Fatalf("cancel = %d, want 0", len(r.CancelIDs))
	}
}
