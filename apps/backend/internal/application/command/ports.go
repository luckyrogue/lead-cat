package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type Store interface {
	GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error)
	CreateMeeting(ctx context.Context, m model.Meeting) (model.Meeting, error)
	CreateMeetingWithParticipants(ctx context.Context, m model.Meeting, ps []model.MeetingParticipant) (model.Meeting, error)
	CreateMeetingSeries(ctx context.Context, ms []model.Meeting, ps []model.MeetingParticipant) ([]model.Meeting, error)
	UpdateMeeting(ctx context.Context, organizationID, id uuid.UUID, m model.Meeting) error
	UpdateMeetingsTx(ctx context.Context, organizationID uuid.UUID, ms []model.Meeting) error
	CancelMeeting(ctx context.Context, organizationID, id uuid.UUID) error
	AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []model.MeetingParticipant) error
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)
	ListSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) ([]model.Meeting, error)
	ListSeriesAllOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) ([]model.Meeting, error)
	ListSeriesOccurrenceStarts(ctx context.Context, organizationID, seriesID uuid.UUID) ([]time.Time, error)
	CancelSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) (int, error)
	CancelAllSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) (int, error)
	ReshapeSeriesTx(ctx context.Context, organizationID, seriesID uuid.UUID, newRows []model.Meeting, ps []model.MeetingParticipant, cancelFrom, until time.Time) (added, removed int, err error)
}

type CalendarProvider interface {
	For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error)
}

type JobQueue interface {
	EnqueueMeetingCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error
}
