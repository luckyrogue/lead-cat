package command

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/platform/fanio"
)

type CreateInput struct {
	Dept            string
	Type            string
	Host            string
	Date            string
	Start           string
	End             string
	Recurrence      string
	RecurrenceUntil string
	RecurrenceDays  []int
	Description     string
	Title           string
	Participants    []model.MeetingParticipant
	Timezone        string
}

type UpdateInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Date        *string
	Start       *string
	End         *string
	Recurrence  *string
	Description *string
	Timezone    string
}

type Meetings struct {
	Store    Store
	Calendar CalendarProvider
	Queue    JobQueue
	Log      *zap.Logger
}

func (c *Meetings) organizerEmail(ctx context.Context, organizerUserID *uuid.UUID) string {
	if organizerUserID == nil {
		return ""
	}
	u, err := c.Store.GetUserByID(ctx, *organizerUserID)
	if err != nil {
		return ""
	}
	return u.Email
}

func (c *Meetings) CreateMeeting(ctx context.Context, organizationID, organizerID uuid.UUID, in CreateInput) (model.Meeting, error) {
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, err
	}
	tz := w.TZ
	if in.Timezone != "" {
		tz = in.Timezone
	}
	loc, err := time.LoadLocation(orDefault(tz, "Asia/Almaty"))
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
	in.Participants, err = normalizeParticipants(in.Participants)
	if err != nil {
		return model.Meeting{}, fmt.Errorf("%w: invalid participant email", ErrInvalidInput)
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
	calSvc, err := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, &organizerID))
	if err != nil {
		return model.Meeting{}, err
	}

	if rec == meeting.Once {
		name := strings.TrimSpace(in.Title)
		if name == "" {
			name = meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)
		}
		cal, err := calSvc.CreateEvent(ctx, docalendar.CalendarEvent{
			Title: name, Description: in.Description, Start: startsAt, End: endsAt, AttendeeEmails: emails,
		})
		if err != nil {
			return model.Meeting{}, fmt.Errorf("calendar: %w", err)
		}
		m, err := c.Store.CreateMeetingWithParticipants(ctx, model.Meeting{
			OrganizationID: organizationID, OrganizerUserID: &organizerID,
			Dept: in.Dept, Type: in.Type, Host: in.Host,
			StartsAt: startsAt.UTC(), EndsAt: endsAt.UTC(),
			Recurrence: string(rec), Name: name, Description: in.Description,
			GoogleEventID: cal.EventID, MeetLink: cal.MeetLink,
		}, in.Participants)
		if err != nil {
			c.deleteEventsBestEffort(ctx, calSvc, []string{cal.EventID})
			return model.Meeting{}, err
		}
		c.enqueueCreated(ctx, organizationID, m.ID)
		return m, nil
	}

	names := make([]string, len(spansList))
	for i, sp := range spansList {
		names[i] = meeting.GenerateName(in.Dept, in.Type, in.Host, sp.Start, rec)
	}
	evs, err := c.createSeriesEvents(ctx, calSvc, names, in.Description, emails, spansList)
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
	created, err := c.Store.CreateMeetingSeries(ctx, rows, in.Participants)
	if err != nil {
		c.deleteEventsBestEffort(ctx, calSvc, eventIDs(evs))
		return model.Meeting{}, err
	}
	anchor := created[0]
	c.enqueueCreated(ctx, organizationID, anchor.ID)
	return anchor, nil
}

func (c *Meetings) UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in UpdateInput) (model.Meeting, error) {
	cur, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return model.Meeting{}, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, err
	}
	if !ownerOrOrganizer(w, cur.OrganizerUserID, userID) {
		return model.Meeting{}, ErrForbidden
	}
	tz := w.TZ
	if in.Timezone != "" {
		tz = in.Timezone
	}
	loc, err := time.LoadLocation(orDefault(tz, "Asia/Almaty"))
	if err != nil {
		return model.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	updated, err := ApplyMeetingUpdate(cur, in, loc)
	if err != nil {
		return model.Meeting{}, err
	}

	if err := c.Store.UpdateMeeting(ctx, organizationID, meetingID, updated); err != nil {
		return model.Meeting{}, err
	}
	if updated.GoogleEventID != "" {
		calSvc, err := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, cur.OrganizerUserID))
		if err != nil {
			return model.Meeting{}, err
		}
		if err := calSvc.UpdateEvent(ctx, updated.GoogleEventID, docalendar.CalendarEvent{
			Title: updated.Name, Description: updated.Description,
			Start: updated.StartsAt, End: updated.EndsAt,
		}); err != nil {
			return model.Meeting{}, fmt.Errorf("calendar: %w", err)
		}
	}
	if c.Queue != nil {
		if err := c.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && c.Log != nil {
			c.Log.Warn("enqueue meeting updated",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return updated, nil
}

func (c *Meetings) CancelMeeting(ctx context.Context, organizationID, userID, id uuid.UUID) error {
	m, err := c.Store.GetMeeting(ctx, organizationID, id)
	if err != nil {
		return err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
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
		if calSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, m.OrganizerUserID)); ferr == nil {
			_ = calSvc.DeleteEvent(ctx, m.GoogleEventID)
		}
	}
	if err := c.Store.CancelMeeting(ctx, organizationID, id); err != nil {
		return err
	}
	c.enqueueCancelled(ctx, organizationID, id)
	return nil
}

func ownerOrOrganizer(w model.Organization, organizerUserID *uuid.UUID, userID uuid.UUID) bool {
	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return true
	}
	return organizerUserID != nil && *organizerUserID == userID
}

func ApplyMeetingUpdate(cur model.Meeting, in UpdateInput, loc *time.Location) (model.Meeting, error) {
	dept := orStr(in.Dept, cur.Dept)
	typ := orStr(in.Type, cur.Type)
	host := orStr(in.Host, cur.Host)
	desc := orStr(in.Description, cur.Description)
	rec := meeting.Recurrence(orStr(in.Recurrence, cur.Recurrence))

	startLocal := cur.StartsAt.In(loc)
	startsAt := cur.StartsAt
	endsAt := cur.EndsAt

	hasDate := in.Date != nil || in.Start != nil || in.End != nil
	allDate := in.Date != nil && in.Start != nil && in.End != nil
	if hasDate && !allDate {
		return model.Meeting{}, fmt.Errorf("%w: date, start, and end must be supplied together", ErrInvalidInput)
	}

	if allDate {
		s, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.Start, loc)
		if err != nil {
			return model.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.End, loc)
		if err != nil {
			return model.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return model.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.Recurrence = string(rec)
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}

type seriesEvent struct {
	Span     meeting.Span
	Name     string
	EventID  string
	MeetLink string
}

func (c *Meetings) createSeriesEvents(ctx context.Context, cal docalendar.Service, names []string, description string, emails []string, spans []meeting.Span) ([]seriesEvent, error) {
	created := make([]seriesEvent, len(spans))
	err := fanio.All(ctx, fanio.CalendarLimit, len(spans), func(ctx context.Context, i int) error {
		sp := spans[i]
		res, err := cal.CreateEvent(ctx, docalendar.CalendarEvent{
			Title: names[i], Description: description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
		})
		if err != nil {
			return err
		}
		created[i] = seriesEvent{Span: sp, Name: names[i], EventID: res.EventID, MeetLink: res.MeetLink}
		return nil
	})
	if err != nil {
		c.deleteEventsBestEffort(ctx, cal, eventIDs(created))
		return nil, fmt.Errorf("calendar: %w", err)
	}
	return created, nil
}

func (c *Meetings) deleteEventsBestEffort(ctx context.Context, cal docalendar.Service, ids []string) {
	fanio.EachStringBestEffort(ctx, fanio.CalendarLimit, ids, func(ctx context.Context, id string) {
		if err := cal.DeleteEvent(ctx, id); err != nil && c.Log != nil {
			c.Log.Warn("delete orphan meeting event", zap.String("event_id", id), zap.Error(err))
		}
	})
}

func eventIDs(evs []seriesEvent) []string {
	var ids []string
	for _, e := range evs {
		if e.EventID != "" {
			ids = append(ids, e.EventID)
		}
	}
	return ids
}

func (c *Meetings) enqueueCreated(ctx context.Context, organizationID, meetingID uuid.UUID) {
	if c.Queue == nil {
		return
	}
	if err := c.Queue.EnqueueMeetingCreated(ctx, organizationID, meetingID); err != nil && c.Log != nil {
		c.Log.Warn("enqueue meeting created",
			zap.String("organization_id", organizationID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}

func (c *Meetings) enqueueCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) {
	if c.Queue == nil {
		return
	}
	if err := c.Queue.EnqueueMeetingCancelled(ctx, organizationID, meetingID); err != nil && c.Log != nil {
		c.Log.Warn("enqueue meeting cancelled",
			zap.String("organization_id", organizationID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

func normalizeParticipants(parts []model.MeetingParticipant) ([]model.MeetingParticipant, error) {
	if len(parts) == 0 {
		return parts, nil
	}
	out := make([]model.MeetingParticipant, len(parts))
	for i, p := range parts {
		addr, err := mail.ParseAddress(strings.TrimSpace(p.Email))
		if err != nil {
			return nil, err
		}
		out[i] = p
		out[i].Email = addr.Address
	}
	return out, nil
}
