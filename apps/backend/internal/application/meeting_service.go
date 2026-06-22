package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type CreateMeetingInput = command.CreateInput

type UpdateMeetingInput = command.UpdateInput

type SeriesUpdateInput = command.SeriesUpdateInput

func (s *Services) ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error) {
	return s.Queries.ListEmployees(ctx, organizationID)
}

func (s *Services) ListMeetings(ctx context.Context, organizationID, userID uuid.UUID) ([]model.Meeting, error) {
	return s.Queries.ListMeetings(ctx, organizationID, userID)
}

func (s *Services) ListMeetingsFiltered(ctx context.Context, organizationID, userID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	return s.Queries.ListMeetingsFiltered(ctx, organizationID, userID, f)
}

func (s *Services) GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error) {
	return s.Queries.GetMeeting(ctx, organizationID, id)
}

func (s *Services) SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]model.Employee, error) {
	return s.Queries.SearchEmployees(ctx, organizationID, query)
}

func (s *Services) SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error) {
	return s.Queries.SearchEmployeesGlobal(ctx, query)
}

func (s *Services) EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error) {
	return s.Queries.EmployeeSchedule(ctx, email, from, to)
}

func (s *Services) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error) {
	return s.Queries.ListParticipants(ctx, meetingID)
}

func (s *Services) CreateMeeting(ctx context.Context, organizationID, organizerID uuid.UUID, in CreateMeetingInput) (model.Meeting, error) {
	return s.Commands.CreateMeeting(ctx, organizationID, organizerID, in)
}

func (s *Services) UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (model.Meeting, error) {
	return s.Commands.UpdateMeeting(ctx, organizationID, userID, meetingID, in)
}

func (s *Services) ListEditableMeetings(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error) {
	return s.Queries.ListEditableMeetings(ctx, telegramID)
}

func (s *Services) CancelMeeting(ctx context.Context, organizationID, userID, id uuid.UUID) error {
	return s.Commands.CancelMeeting(ctx, organizationID, userID, id)
}

func (s *Services) UpdateSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	return s.Commands.UpdateSeries(ctx, organizationID, userID, meetingID, in)
}

func (s *Services) CancelSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	return s.Commands.CancelSeries(ctx, organizationID, userID, meetingID)
}

func (s *Services) UpdateWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	return s.Commands.UpdateWholeSeries(ctx, organizationID, userID, meetingID, in)
}

func (s *Services) CancelWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	return s.Commands.CancelWholeSeries(ctx, organizationID, userID, meetingID)
}

func (s *Services) ChangeSeriesEnd(ctx context.Context, organizationID, userID, meetingID uuid.UUID, untilStr string) (int, int, error) {
	return s.Commands.ChangeSeriesEnd(ctx, organizationID, userID, meetingID, untilStr)
}

func (s *Services) AddParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	return s.Commands.AddParticipant(ctx, organizationID, userID, meetingID, email)
}

func (s *Services) RemoveParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	return s.Commands.RemoveParticipant(ctx, organizationID, userID, meetingID, email)
}
