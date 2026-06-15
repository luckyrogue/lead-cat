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
