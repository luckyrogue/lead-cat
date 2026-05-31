package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// ErrForbidden is returned when a caller may not act on a meeting.
var ErrForbidden = errors.New("forbidden")

// CreateMeetingInput is the transport-level payload (strings as received over HTTP).
type CreateMeetingInput struct {
	Dept         string
	Type         string
	Host         string
	Date         string // YYYY-MM-DD
	Start        string // HH:MM
	End          string // HH:MM
	Recurrence   string
	Description  string
	Participants []postgres.MeetingParticipant
}

func (s *Services) ListEmployees(ctx context.Context, workspaceID uuid.UUID) ([]postgres.Employee, error) {
	return s.Store.ListEmployees(ctx, workspaceID)
}

func (s *Services) ListMeetings(ctx context.Context, workspaceID, userID uuid.UUID) ([]postgres.Meeting, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	// Workspace owner sees all meetings; everyone else sees their own (ТЗ §2).
	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return s.Store.ListMeetings(ctx, workspaceID)
	}
	return s.Store.ListMeetingsByOrganizer(ctx, workspaceID, userID)
}

func (s *Services) GetMeeting(ctx context.Context, workspaceID, id uuid.UUID) (postgres.Meeting, error) {
	m, err := s.Store.GetMeeting(ctx, workspaceID, id)
	if err != nil {
		return m, err
	}
	m.Participants, err = s.Store.ListParticipants(ctx, id)
	return m, err
}

func (s *Services) CreateMeeting(ctx context.Context, workspaceID, organizerID uuid.UUID, in CreateMeetingInput) (postgres.Meeting, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	startsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Start, loc)
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
	}
	endsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.End, loc)
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
	}

	rec := meeting.Recurrence(orDefault(in.Recurrence, string(meeting.Once)))
	dom := meeting.Input{
		Dept: in.Dept, Type: in.Type, Host: in.Host,
		StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: in.Description,
	}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)

	var emails []string
	for _, p := range in.Participants {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := s.Calendar.For(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	cal, err := calSvc.CreateEvent(ctx, CalendarEvent{
		Title: name, Description: in.Description,
		Start: startsAt, End: endsAt, AttendeeEmails: emails,
	})
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("calendar: %w", err)
	}

	m, err := s.Store.CreateMeeting(ctx, postgres.Meeting{
		WorkspaceID: workspaceID, OrganizerUserID: &organizerID,
		Dept: in.Dept, Type: in.Type, Host: in.Host,
		StartsAt: startsAt.UTC(), EndsAt: endsAt.UTC(),
		Recurrence: string(rec), Name: name, Description: in.Description,
		GoogleEventID: cal.EventID, MeetLink: cal.MeetLink,
	})
	if err != nil {
		return postgres.Meeting{}, err
	}
	if len(in.Participants) > 0 {
		if err := s.Store.AddParticipants(ctx, m.ID, in.Participants); err != nil {
			return m, err
		}
		m.Participants = in.Participants
	}
	// Best-effort: the meeting is already created; a failed enqueue only loses the
	// creation notification, so log and still return the meeting.
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingCreated(ctx, workspaceID, m.ID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting created",
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return m, nil
}

func (s *Services) CancelMeeting(ctx context.Context, workspaceID, userID, id uuid.UUID) error {
	m, err := s.Store.GetMeeting(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	isOwner := w.OwnerUserID != nil && *w.OwnerUserID == userID
	isOrganizer := m.OrganizerUserID != nil && *m.OrganizerUserID == userID
	if !isOwner && !isOrganizer {
		return ErrForbidden
	}
	if m.GoogleEventID != "" {
		if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr == nil {
			_ = calSvc.DeleteEvent(ctx, m.GoogleEventID) // best-effort
		}
	}
	return s.Store.CancelMeeting(ctx, workspaceID, id)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
