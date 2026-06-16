package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type reshapeRepo struct {
	unimplementedRepo
	org      model.Organization
	picked   model.Meeting
	occs     []model.Meeting
	starts   []time.Time
	created  []model.Meeting
	setUntil time.Time
	setCalls int
}

func (r *reshapeRepo) GetMeeting(context.Context, uuid.UUID, uuid.UUID) (model.Meeting, error) {
	return r.picked, nil
}
func (r *reshapeRepo) GetOrganization(context.Context, uuid.UUID) (model.Organization, error) {
	return r.org, nil
}
func (r *reshapeRepo) ListSeriesAllOccurrences(context.Context, uuid.UUID, uuid.UUID) ([]model.Meeting, error) {
	return r.occs, nil
}
func (r *reshapeRepo) ListSeriesOccurrenceStarts(context.Context, uuid.UUID, uuid.UUID) ([]time.Time, error) {
	return r.starts, nil
}
func (r *reshapeRepo) ListParticipants(context.Context, uuid.UUID) ([]model.MeetingParticipant, error) {
	return nil, nil
}
func (r *reshapeRepo) CreateMeetingSeries(_ context.Context, ms []model.Meeting, _ []model.MeetingParticipant) ([]model.Meeting, error) {
	r.created = append(r.created, ms...)
	return ms, nil
}
func (r *reshapeRepo) SetSeriesRecurrenceUntil(_ context.Context, _ uuid.UUID, _ uuid.UUID, until time.Time) error {
	r.setUntil = until
	r.setCalls++
	return nil
}

type reshapeCalProvider struct{}

func (reshapeCalProvider) For(context.Context, uuid.UUID) (CalendarService, error) {
	return reshapeCal{}, nil
}

type reshapeCal struct{}

func (reshapeCal) CreateEvent(context.Context, CalendarEvent) (CalendarResult, error) {
	return CalendarResult{EventID: "evt", MeetLink: "https://meet"}, nil
}
func (reshapeCal) UpdateEvent(context.Context, string, CalendarEvent) error { return nil }
func (reshapeCal) UpdateAttendees(context.Context, string, []string) error  { return nil }
func (reshapeCal) DeleteEvent(context.Context, string) error                { return nil }

func TestChangeSeriesEnd_NotASeries(t *testing.T) {
	owner := uuid.New()
	repo := &reshapeRepo{
		org:    model.Organization{OwnerUserID: &owner, TZ: "UTC"},
		picked: model.Meeting{ID: uuid.New(), Recurrence: "once"},
	}
	s := &Services{Store: repo, Calendar: reshapeCalProvider{}}
	if _, _, err := s.ChangeSeriesEnd(context.Background(), uuid.New(), owner, repo.picked.ID, "2026-07-01"); err == nil {
		t.Fatal("expected error for non-series")
	}
}

func TestChangeSeriesEnd_Extend(t *testing.T) {
	owner := uuid.New()
	seriesID := uuid.New()
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	until := start.AddDate(0, 0, 1)
	anchor := model.Meeting{
		ID: uuid.New(), OrganizerUserID: &owner, SeriesID: &seriesID,
		Dept: "Eng", Type: "Sync", Host: "h", Recurrence: "daily",
		StartsAt: start, EndsAt: start.Add(time.Hour), RecurrenceUntil: &until,
	}
	repo := &reshapeRepo{
		org:    model.Organization{OwnerUserID: &owner, TZ: "UTC"},
		picked: anchor, occs: []model.Meeting{anchor},
		starts: []time.Time{start},
	}
	s := &Services{Store: repo, Calendar: reshapeCalProvider{}}
	added, removed, err := s.ChangeSeriesEnd(context.Background(), uuid.New(), owner, anchor.ID, "2026-06-03")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if added != 2 || removed != 0 {
		t.Fatalf("added=%d removed=%d, want 2/0", added, removed)
	}
	if repo.setCalls == 0 {
		t.Fatal("expected SetSeriesRecurrenceUntil to be called")
	}
}

func TestChangeSeriesEnd_ExtendOverCancelledTailCreatesNothing(t *testing.T) {
	owner := uuid.New()
	seriesID := uuid.New()
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	d2 := start.AddDate(0, 0, 1)
	d3 := start.AddDate(0, 0, 2)
	until := start
	anchor := model.Meeting{
		ID: uuid.New(), OrganizerUserID: &owner, SeriesID: &seriesID,
		Dept: "Eng", Type: "Sync", Host: "h", Recurrence: "daily",
		StartsAt: start, EndsAt: start.Add(time.Hour), RecurrenceUntil: &until,
	}
	// Series was trimmed to d1: d2,d3 are cancelled but their rows still exist.
	repo := &reshapeRepo{
		org:    model.Organization{OwnerUserID: &owner, TZ: "UTC"},
		picked: anchor, occs: []model.Meeting{anchor},
		starts: []time.Time{start, d2, d3},
	}
	s := &Services{Store: repo, Calendar: reshapeCalProvider{}}
	added, removed, err := s.ChangeSeriesEnd(context.Background(), uuid.New(), owner, anchor.ID, "2026-06-03")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if added != 0 || removed != 0 {
		t.Fatalf("added=%d removed=%d, want 0/0 (no duplicate rows for cancelled tail)", added, removed)
	}
	if len(repo.created) != 0 {
		t.Fatalf("created %d rows, want 0", len(repo.created))
	}
}
