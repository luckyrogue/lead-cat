package command

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

// Compile-time proof the fakes satisfy the real ports.
var (
	_ Store              = (*fakeStore)(nil)
	_ CalendarProvider   = (*fakeCalProvider)(nil)
	_ JobQueue           = (*fakeQueue)(nil)
	_ docalendar.Service = (*fakeCalService)(nil)
)

type fakeStore struct {
	org       model.Organization
	orgErr    error
	meetings  map[uuid.UUID]model.Meeting
	created   []model.Meeting
	series    [][]model.Meeting
	updated   []model.Meeting
	cancelled []uuid.UUID
}

func newFakeStore() *fakeStore { return &fakeStore{meetings: map[uuid.UUID]model.Meeting{}} }

func (f *fakeStore) GetOrganization(_ context.Context, _ uuid.UUID) (model.Organization, error) {
	if f.orgErr != nil {
		return model.Organization{}, f.orgErr
	}
	return f.org, nil
}

func (f *fakeStore) GetUserByID(_ context.Context, id uuid.UUID) (model.User, error) {
	return model.User{ID: id}, nil
}

func (f *fakeStore) GetMeeting(_ context.Context, _, id uuid.UUID) (model.Meeting, error) {
	m, ok := f.meetings[id]
	if !ok {
		return model.Meeting{}, errors.New("not found")
	}
	return m, nil
}

func (f *fakeStore) CreateMeeting(_ context.Context, m model.Meeting) (model.Meeting, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	f.meetings[m.ID] = m
	f.created = append(f.created, m)
	return m, nil
}

func (f *fakeStore) CreateMeetingSeries(_ context.Context, ms []model.Meeting, ps []model.MeetingParticipant) ([]model.Meeting, error) {
	out := make([]model.Meeting, 0, len(ms))
	for _, m := range ms {
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		m.Participants = ps
		f.meetings[m.ID] = m
		out = append(out, m)
	}
	f.series = append(f.series, out)
	return out, nil
}

func (f *fakeStore) UpdateMeeting(_ context.Context, _, id uuid.UUID, m model.Meeting) error {
	f.meetings[id] = m
	f.updated = append(f.updated, m)
	return nil
}

func (f *fakeStore) CancelMeeting(_ context.Context, _, id uuid.UUID) error {
	if m, ok := f.meetings[id]; ok {
		m.Status = "cancelled"
		f.meetings[id] = m
	}
	f.cancelled = append(f.cancelled, id)
	return nil
}

func (f *fakeStore) AddParticipants(_ context.Context, _ uuid.UUID, _ []model.MeetingParticipant) error {
	return nil
}

type fakeCalProvider struct {
	svc    *fakeCalService
	forErr error
}

func (p *fakeCalProvider) For(_ context.Context, _ uuid.UUID, _ string) (docalendar.Service, error) {
	if p.forErr != nil {
		return nil, p.forErr
	}
	return p.svc, nil
}

// fakeCalService is concurrency-safe: the real series path fans out CreateEvent
// across goroutines (fanio.All), so the recording slices are guarded by a mutex.
type fakeCalService struct {
	failCreate bool

	mu      sync.Mutex
	created []docalendar.CalendarEvent
	updated []string
	deleted []string
}

func (s *fakeCalService) CreateEvent(_ context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	if s.failCreate {
		return docalendar.CalendarResult{}, errors.New("boom")
	}
	s.mu.Lock()
	s.created = append(s.created, e)
	s.mu.Unlock()
	id := "evt-" + uuid.NewString()[:8]
	return docalendar.CalendarResult{EventID: id, MeetLink: "https://meet.google.com/" + id}, nil
}

func (s *fakeCalService) UpdateEvent(_ context.Context, eventID string, _ docalendar.CalendarEvent) error {
	s.mu.Lock()
	s.updated = append(s.updated, eventID)
	s.mu.Unlock()
	return nil
}

func (s *fakeCalService) UpdateAttendees(_ context.Context, _ string, _ []string) error { return nil }

func (s *fakeCalService) DeleteEvent(_ context.Context, eventID string) error {
	s.mu.Lock()
	s.deleted = append(s.deleted, eventID)
	s.mu.Unlock()
	return nil
}

type fakeQueue struct {
	createdEnq   []uuid.UUID
	updatedEnq   []uuid.UUID
	cancelledEnq []uuid.UUID
}

func (q *fakeQueue) EnqueueMeetingCreated(_ context.Context, _, meetingID uuid.UUID) error {
	q.createdEnq = append(q.createdEnq, meetingID)
	return nil
}

func (q *fakeQueue) EnqueueMeetingUpdated(_ context.Context, _, meetingID uuid.UUID) error {
	q.updatedEnq = append(q.updatedEnq, meetingID)
	return nil
}

func (q *fakeQueue) EnqueueMeetingCancelled(_ context.Context, _, meetingID uuid.UUID) error {
	q.cancelledEnq = append(q.cancelledEnq, meetingID)
	return nil
}
