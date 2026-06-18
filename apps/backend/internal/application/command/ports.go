package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type Store interface {
	GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error)
	CreateMeeting(ctx context.Context, m model.Meeting) (model.Meeting, error)
	CreateMeetingSeries(ctx context.Context, ms []model.Meeting, ps []model.MeetingParticipant) ([]model.Meeting, error)
	UpdateMeeting(ctx context.Context, organizationID, id uuid.UUID, m model.Meeting) error
	CancelMeeting(ctx context.Context, organizationID, id uuid.UUID) error
	AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []model.MeetingParticipant) error
}

type CalendarProvider interface {
	For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error)
}

type JobQueue interface {
	EnqueueMeetingCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error
}
