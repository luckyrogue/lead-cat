package application

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func ownerOrOrganizer(w model.Organization, organizerUserID *uuid.UUID, userID uuid.UUID) bool {
	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return true
	}
	return organizerUserID != nil && *organizerUserID == userID
}

func normalizeEmail(s string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}
	return strings.ToLower(addr.Address), nil
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

func (s *Services) SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]model.Employee, error) {
	all, err := s.Store.ListEmployees(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return filterEmployees(all, query), nil
}

func (s *Services) SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error) {
	return s.Store.SearchEmployeesGlobal(ctx, query)
}

func (s *Services) EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error) {
	return s.Store.ListScheduleForEmail(ctx, email, from, to)
}

func (s *Services) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error) {
	return s.Store.ListParticipants(ctx, meetingID)
}

func (s *Services) AddParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, organizationID, meetingID, userID)
	if err != nil {
		return err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return fmt.Errorf("%w: bad email: %v", ErrInvalidInput, err)
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if p.Email == email {
			return fmt.Errorf("%w: already a participant", ErrInvalidInput)
		}
	}
	if err := s.Store.AddParticipants(ctx, meetingID, []model.MeetingParticipant{{Email: email}}); err != nil {
		return err
	}
	if err := s.syncAttendees(ctx, organizationID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantAdded(ctx, organizationID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant added", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

func (s *Services) RemoveParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, organizationID, meetingID, userID)
	if err != nil {
		return err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return fmt.Errorf("%w: bad email: %v", ErrInvalidInput, err)
	}
	if err := s.Store.RemoveParticipant(ctx, meetingID, email); err != nil {
		return err
	}
	if err := s.syncAttendees(ctx, organizationID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantRemoved(ctx, organizationID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant removed", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

func (s *Services) loadForParticipantOp(ctx context.Context, organizationID, meetingID, userID uuid.UUID) (model.Meeting, model.Organization, error) {
	m, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return model.Meeting{}, model.Organization{}, err
	}
	org, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, model.Organization{}, err
	}
	if !ownerOrOrganizer(org, m.OrganizerUserID, userID) {
		return model.Meeting{}, model.Organization{}, ErrForbidden
	}
	if m.Status != "scheduled" {
		return model.Meeting{}, model.Organization{}, model.ErrMeetingNotEditable
	}
	return m, org, nil
}

func (s *Services) syncAttendees(ctx context.Context, organizationID uuid.UUID, googleEventID string, meetingID uuid.UUID) error {
	if googleEventID == "" {
		return nil
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	var emails []string
	for _, p := range parts {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := s.Calendar.For(ctx, organizationID)
	if err != nil {
		return err
	}
	if err := calSvc.UpdateAttendees(ctx, googleEventID, emails); err != nil {
		return fmt.Errorf("calendar: %w", err)
	}
	return nil
}
