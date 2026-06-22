package query

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type Store interface {
	ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error)
	SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error)
	ListScheduleForEmail(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error)
	ListMeetingsFiltered(ctx context.Context, organizationID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error)
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)
	ListMeetingsByOrganizerTelegram(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error)
}

type Meetings struct {
	Store Store
}

func NewMeetings(store Store) *Meetings {
	return &Meetings{Store: store}
}

func (m *Meetings) ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error) {
	return m.Store.ListEmployees(ctx, organizationID)
}

func (m *Meetings) SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]model.Employee, error) {
	all, err := m.Store.ListEmployees(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return filterEmployees(all, query), nil
}

func (m *Meetings) SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error) {
	return m.Store.SearchEmployeesGlobal(ctx, query)
}

func (m *Meetings) EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error) {
	return m.Store.ListScheduleForEmail(ctx, email, from, to)
}

func (m *Meetings) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error) {
	return m.Store.ListParticipants(ctx, meetingID)
}

func (m *Meetings) ListMeetings(ctx context.Context, organizationID, userID uuid.UUID) ([]model.Meeting, error) {
	return m.ListMeetingsFiltered(ctx, organizationID, userID, model.MeetingFilter{})
}

func (m *Meetings) ListMeetingsFiltered(ctx context.Context, organizationID, userID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	w, err := m.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if w.OwnerUserID == nil || *w.OwnerUserID != userID {
		f.Organizer = &userID
	}
	return m.Store.ListMeetingsFiltered(ctx, organizationID, f)
}

func (m *Meetings) GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error) {
	mt, err := m.Store.GetMeeting(ctx, organizationID, id)
	if err != nil {
		return mt, err
	}
	mt.Participants, err = m.Store.ListParticipants(ctx, id)
	return mt, err
}

func (m *Meetings) ListEditableMeetings(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error) {
	return m.Store.ListMeetingsByOrganizerTelegram(ctx, telegramID)
}

func filterEmployees(all []model.Employee, query string) []model.Employee {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []model.Employee
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.FullName), q) || strings.Contains(strings.ToLower(e.Email), q) {
			out = append(out, e)
		}
	}
	return out
}
