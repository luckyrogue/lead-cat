package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

// ErrForbidden is returned when a caller may not act on a meeting.
var ErrForbidden = errors.New("forbidden")

// CreateMeetingInput is the transport-level payload (strings as received over HTTP).
type CreateMeetingInput struct {
	Dept            string
	Type            string
	Host            string
	Date            string // YYYY-MM-DD
	Start           string // HH:MM
	End             string // HH:MM
	Recurrence      string
	RecurrenceUntil string // YYYY-MM-DD; required when Recurrence != once
	RecurrenceDays  []int  // 1..7 (Mon..Sun); required when Recurrence == "custom"
	Description     string
	Participants    []model.MeetingParticipant
}

func (s *Services) ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error) {
	return s.Store.ListEmployees(ctx, organizationID)
}

func (s *Services) ListMeetings(ctx context.Context, organizationID, userID uuid.UUID) ([]model.Meeting, error) {
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return s.Store.ListMeetings(ctx, organizationID)
	}
	return s.Store.ListMeetingsByOrganizer(ctx, organizationID, userID)
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
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, err
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return model.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	startsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Start, loc)
	if err != nil {
		return model.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
	}
	endsAt, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.End, loc)
	if err != nil {
		return model.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
	}

	rec := meeting.Recurrence(orDefault(in.Recurrence, string(meeting.Once)))
	dom := meeting.Input{
		Dept: in.Dept, Type: in.Type, Host: in.Host,
		StartsAt: startsAt, EndsAt: endsAt,
		Recurrence: rec, RecurrenceDays: in.RecurrenceDays,
		Description: in.Description,
	}
	if err := dom.Validate(); err != nil {
		return model.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	var until time.Time
	if in.RecurrenceUntil != "" {
		until, err = time.ParseInLocation("2006-01-02", in.RecurrenceUntil, loc)
		if err != nil {
			return model.Meeting{}, fmt.Errorf("%w: bad recurrence_until", ErrInvalidInput)
		}
	}
	spansList, err := meeting.Occurrences(startsAt, endsAt, rec, in.RecurrenceDays, until)
	if err != nil {
		return model.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	var emails []string
	for _, p := range in.Participants {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := s.Calendar.For(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, err
	}

	if rec == meeting.Once {
		name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)
		cal, err := calSvc.CreateEvent(ctx, CalendarEvent{
			Title: name, Description: in.Description, Start: startsAt, End: endsAt, AttendeeEmails: emails,
		})
		if err != nil {
			return model.Meeting{}, fmt.Errorf("calendar: %w", err)
		}
		m, err := s.Store.CreateMeeting(ctx, model.Meeting{
			OrganizationID: organizationID, OrganizerUserID: &organizerID,
			Dept: in.Dept, Type: in.Type, Host: in.Host,
			StartsAt: startsAt.UTC(), EndsAt: endsAt.UTC(),
			Recurrence: string(rec), Name: name, Description: in.Description,
			GoogleEventID: cal.EventID, MeetLink: cal.MeetLink,
		})
		if err != nil {
			return model.Meeting{}, err
		}
		if len(in.Participants) > 0 {
			if err := s.Store.AddParticipants(ctx, m.ID, in.Participants); err != nil {
				return m, err
			}
			m.Participants = in.Participants
		}
		s.enqueueCreated(ctx, organizationID, m.ID)
		return m, nil
	}

	names := make([]string, len(spansList))
	for i, sp := range spansList {
		names[i] = meeting.GenerateName(in.Dept, in.Type, in.Host, sp.Start, rec)
	}
	evs, err := s.createSeriesEvents(ctx, calSvc, names, in.Description, emails, spansList)
	if err != nil {
		return model.Meeting{}, err
	}
	seriesID := uuid.New()
	recUntil := until
	rows := make([]model.Meeting, len(evs))
	for i, e := range evs {
		rows[i] = model.Meeting{
			OrganizationID: organizationID, OrganizerUserID: &organizerID,
			Dept: in.Dept, Type: in.Type, Host: in.Host,
			StartsAt: e.Span.Start.UTC(), EndsAt: e.Span.End.UTC(),
			Recurrence: string(rec), Name: e.Name, Description: in.Description,
			GoogleEventID: e.EventID, MeetLink: e.MeetLink,
			SeriesID: &seriesID, RecurrenceUntil: &recUntil,
		}
	}
	created, err := s.Store.CreateMeetingSeries(ctx, rows, in.Participants)
	if err != nil {
		s.deleteEventsBestEffort(ctx, calSvc, eventIDs(evs))
		return model.Meeting{}, err
	}
	anchor := created[0]
	s.enqueueCreated(ctx, organizationID, anchor.ID)
	return anchor, nil
}

// enqueueCreated best-effort enqueues the meeting-created notification (once).
func (s *Services) enqueueCreated(ctx context.Context, organizationID, meetingID uuid.UUID) {
	if s.Queue == nil {
		return
	}
	if err := s.Queue.EnqueueMeetingCreated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
		s.Log.Warn("enqueue meeting created",
			zap.String("organization_id", organizationID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}

// enqueueCancelled best-effort enqueues the meeting-cancelled notification.
func (s *Services) enqueueCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) {
	if s.Queue == nil {
		return
	}
	if err := s.Queue.EnqueueMeetingCancelled(ctx, organizationID, meetingID); err != nil && s.Log != nil {
		s.Log.Warn("enqueue meeting cancelled",
			zap.String("organization_id", organizationID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}

// seriesEvent is a created Google event paired with its span and computed name.
type seriesEvent struct {
	Span     meeting.Span
	Name     string
	EventID  string
	MeetLink string
}

// createSeriesEvents creates one Google event per span (names[i] is the title for
// spans[i]). On any failure it best-effort deletes the events already created and
// returns the error, so no partial series leaks.
func (s *Services) createSeriesEvents(ctx context.Context, cal CalendarService, names []string, description string, emails []string, spans []meeting.Span) ([]seriesEvent, error) {
	var created []seriesEvent
	for i, sp := range spans {
		res, err := cal.CreateEvent(ctx, CalendarEvent{
			Title: names[i], Description: description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
		})
		if err != nil {
			s.deleteEventsBestEffort(ctx, cal, eventIDs(created))
			return nil, fmt.Errorf("calendar: %w", err)
		}
		created = append(created, seriesEvent{Span: sp, Name: names[i], EventID: res.EventID, MeetLink: res.MeetLink})
	}
	return created, nil
}

// deleteEventsBestEffort deletes Google events, logging (not failing) on error.
func (s *Services) deleteEventsBestEffort(ctx context.Context, cal CalendarService, ids []string) {
	for _, id := range ids {
		if err := cal.DeleteEvent(ctx, id); err != nil && s.Log != nil {
			s.Log.Warn("delete orphan meeting event", zap.String("event_id", id), zap.Error(err))
		}
	}
}

func eventIDs(evs []seriesEvent) []string {
	ids := make([]string, len(evs))
	for i, e := range evs {
		ids[i] = e.EventID
	}
	return ids
}

// UpdateMeeting applies field overrides to a meeting (organizer or organization
// owner only): validates, recomputes the name, patches the Google event, persists,
// and enqueues a change notification. Mirrors CreateMeeting.
func (s *Services) UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (model.Meeting, error) {
	cur, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return model.Meeting{}, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, err
	}
	if !ownerOrOrganizer(w, cur.OrganizerUserID, userID) {
		return model.Meeting{}, ErrForbidden
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return model.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	updated, err := applyMeetingUpdate(cur, in, loc)
	if err != nil {
		return model.Meeting{}, err
	}

	if updated.GoogleEventID != "" {
		calSvc, err := s.Calendar.For(ctx, organizationID)
		if err != nil {
			return model.Meeting{}, err
		}
		if err := calSvc.UpdateEvent(ctx, updated.GoogleEventID, CalendarEvent{
			Title: updated.Name, Description: updated.Description,
			Start: updated.StartsAt, End: updated.EndsAt,
		}); err != nil {
			return model.Meeting{}, fmt.Errorf("calendar: %w", err)
		}
	}
	if err := s.Store.UpdateMeeting(ctx, organizationID, meetingID, updated); err != nil {
		return model.Meeting{}, err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting updated",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return updated, nil
}

// ListEditableMeetings returns the upcoming meetings the Telegram user organizes,
// each with its organization timezone (for the bot edit FSM).
func (s *Services) ListEditableMeetings(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error) {
	return s.Store.ListMeetingsByOrganizerTelegram(ctx, telegramID)
}

func (s *Services) CancelMeeting(ctx context.Context, organizationID, userID, id uuid.UUID) error {
	m, err := s.Store.GetMeeting(ctx, organizationID, id)
	if err != nil {
		return err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return err
	}
	if !ownerOrOrganizer(w, m.OrganizerUserID, userID) {
		return ErrForbidden
	}
	if m.Status != "scheduled" {
		return nil
	}
	if m.GoogleEventID != "" {
		if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr == nil {
			_ = calSvc.DeleteEvent(ctx, m.GoogleEventID)
		}
	}
	if err := s.Store.CancelMeeting(ctx, organizationID, id); err != nil {
		return err
	}
	s.enqueueCancelled(ctx, organizationID, id)
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
