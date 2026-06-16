package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type CreateMeetingInput = command.CreateInput

type UpdateMeetingInput = command.UpdateInput

func (s *Services) ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error) {
	return s.Store.ListEmployees(ctx, organizationID)
}

func (s *Services) ListMeetings(ctx context.Context, organizationID, userID uuid.UUID) ([]model.Meeting, error) {
	return s.ListMeetingsFiltered(ctx, organizationID, userID, model.MeetingFilter{})
}

func (s *Services) ListMeetingsFiltered(ctx context.Context, organizationID, userID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if w.OwnerUserID == nil || *w.OwnerUserID != userID {
		f.Organizer = &userID
	}
	return s.Store.ListMeetingsFiltered(ctx, organizationID, f)
}

func (s *Services) GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error) {
	m, err := s.Store.GetMeeting(ctx, organizationID, id)
	if err != nil {
		return m, err
	}
	m.Participants, err = s.Store.ListParticipants(ctx, id)
	return m, err
}

func (s *Services) CreateMeeting(ctx context.Context, organizationID, organizerID uuid.UUID, in CreateMeetingInput) (model.Meeting, error) {
	return s.Commands.CreateMeeting(ctx, organizationID, organizerID, in)
}

func (s *Services) UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (model.Meeting, error) {
	return s.Commands.UpdateMeeting(ctx, organizationID, userID, meetingID, in)
}

func (s *Services) ListEditableMeetings(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error) {
	return s.Store.ListMeetingsByOrganizerTelegram(ctx, telegramID)
}

func (s *Services) CancelMeeting(ctx context.Context, organizationID, userID, id uuid.UUID) error {
	return s.Commands.CancelMeeting(ctx, organizationID, userID, id)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
