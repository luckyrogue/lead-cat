package application

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// ownerOrOrganizer reports whether userID is the workspace owner or the meeting's organizer.
func ownerOrOrganizer(w postgres.Workspace, organizerUserID *uuid.UUID, userID uuid.UUID) bool {
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

func filterEmployees(all []postgres.Employee, query string) []postgres.Employee {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []postgres.Employee
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.FullName), q) || strings.Contains(strings.ToLower(e.Email), q) {
			out = append(out, e)
		}
	}
	return out
}

// SearchEmployees returns directory entries whose name or email contains query.
func (s *Services) SearchEmployees(ctx context.Context, workspaceID uuid.UUID, query string) ([]postgres.Employee, error) {
	all, err := s.Store.ListEmployees(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return filterEmployees(all, query), nil
}

// ListParticipants returns a meeting's participants (for the bot FSM).
func (s *Services) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return s.Store.ListParticipants(ctx, meetingID)
}

// AddParticipant adds a guest by email (organizer or owner only): persists, syncs
// the Google attendee list, and enqueues a notification.
func (s *Services) AddParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, workspaceID, meetingID, userID)
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
	if err := s.Store.AddParticipants(ctx, meetingID, []postgres.MeetingParticipant{{Email: email}}); err != nil {
		return err
	}
	if err := s.syncAttendees(ctx, workspaceID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantAdded(ctx, workspaceID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant added", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

// RemoveParticipant removes a guest by email (organizer or owner only): persists,
// syncs the Google attendee list, and enqueues a notification.
func (s *Services) RemoveParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, workspaceID, meetingID, userID)
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
	if err := s.syncAttendees(ctx, workspaceID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantRemoved(ctx, workspaceID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant removed", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

// loadForParticipantOp loads the meeting + workspace and enforces the ACL.
func (s *Services) loadForParticipantOp(ctx context.Context, workspaceID, meetingID, userID uuid.UUID) (postgres.Meeting, postgres.Workspace, error) {
	m, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return postgres.Meeting{}, postgres.Workspace{}, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, postgres.Workspace{}, err
	}
	if !ownerOrOrganizer(w, m.OrganizerUserID, userID) {
		return postgres.Meeting{}, postgres.Workspace{}, ErrForbidden
	}
	return m, w, nil
}

// syncAttendees patches the Google event's guest list to the meeting's current
// participants (no-op when the meeting has no Google event).
func (s *Services) syncAttendees(ctx context.Context, workspaceID uuid.UUID, googleEventID string, meetingID uuid.UUID) error {
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
	calSvc, err := s.Calendar.For(ctx, workspaceID)
	if err != nil {
		return err
	}
	if err := calSvc.UpdateAttendees(ctx, googleEventID, emails); err != nil {
		return fmt.Errorf("calendar: %w", err)
	}
	return nil
}
