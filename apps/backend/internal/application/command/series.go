package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/platform/fanio"
)

type SeriesUpdateInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Description *string
	Start       *string
	End         *string
}

func applySeriesUpdate(cur model.Meeting, in SeriesUpdateInput, loc *time.Location) (model.Meeting, error) {
	dept := orStr(in.Dept, cur.Dept)
	typ := orStr(in.Type, cur.Type)
	host := orStr(in.Host, cur.Host)
	desc := orStr(in.Description, cur.Description)

	startLocal := cur.StartsAt.In(loc)
	startsAt := cur.StartsAt
	endsAt := cur.EndsAt
	if in.Start != nil && in.End != nil {
		day := cur.StartsAt.In(loc).Format("2006-01-02")
		s, err := time.ParseInLocation("2006-01-02 15:04", day+" "+*in.Start, loc)
		if err != nil {
			return model.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", day+" "+*in.End, loc)
		if err != nil {
			return model.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	rec := meeting.Recurrence(cur.Recurrence)
	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return model.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}

func (c *Meetings) updateCalendarEventsBestEffort(ctx context.Context, cal docalendar.Service, rows []model.Meeting, logMsg string) {
	fanio.AllBestEffort(ctx, fanio.CalendarLimit, len(rows), func(ctx context.Context, i int) {
		m := rows[i]
		if m.GoogleEventID == "" {
			return
		}
		if err := cal.UpdateEvent(ctx, m.GoogleEventID, docalendar.CalendarEvent{
			Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
		}); err != nil && c.Log != nil {
			c.Log.Warn(logMsg, zap.String("event_id", m.GoogleEventID), zap.Error(err))
		}
	})
}

func (c *Meetings) UpdateSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return 0, fmt.Errorf("bad timezone: %w", err)
	}
	occs, err := c.Store.ListSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	rows := make([]model.Meeting, 0, len(occs))
	for _, oc := range occs {
		upd, err := applySeriesUpdate(oc, in, loc)
		if err != nil {
			return 0, err
		}
		rows = append(rows, upd)
	}
	if err := c.Store.UpdateMeetingsTx(ctx, organizationID, rows); err != nil {
		return 0, err
	}

	if calSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if c.Log != nil {
			c.Log.Warn("series update calendar provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		c.updateCalendarEventsBestEffort(ctx, calSvc, rows, "series update event")
	}
	if c.Queue != nil {
		if err := c.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && c.Log != nil {
			c.Log.Warn("enqueue meeting updated",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}

func (c *Meetings) CancelSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := c.Store.ListSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	n, err := c.Store.CancelSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}

	if calSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if c.Log != nil {
			c.Log.Warn("cancel series calendar provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		c.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	c.enqueueCancelled(ctx, organizationID, meetingID)
	return n, nil
}

func (c *Meetings) UpdateWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return 0, fmt.Errorf("bad timezone: %w", err)
	}
	occs, err := c.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	rows := make([]model.Meeting, 0, len(occs))
	for _, oc := range occs {
		upd, err := applySeriesUpdate(oc, in, loc)
		if err != nil {
			return 0, err
		}
		rows = append(rows, upd)
	}
	if err := c.Store.UpdateMeetingsTx(ctx, organizationID, rows); err != nil {
		return 0, err
	}
	if calSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if c.Log != nil {
			c.Log.Warn("whole_series_update_calendar_provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		c.updateCalendarEventsBestEffort(ctx, calSvc, rows, "whole_series_update_event")
	}
	if c.Queue != nil {
		if err := c.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && c.Log != nil {
			c.Log.Warn("enqueue_meeting_updated_whole_series",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}

func (c *Meetings) CancelWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := c.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	n, err := c.Store.CancelAllSeriesOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	if calSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if c.Log != nil {
			c.Log.Warn("whole_series_cancel_calendar_provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		c.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	c.enqueueCancelled(ctx, organizationID, meetingID)
	return n, nil
}

func (c *Meetings) ChangeSeriesEnd(ctx context.Context, organizationID, userID, meetingID uuid.UUID, untilStr string) (int, int, error) {
	picked, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, 0, err
	}
	w, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return 0, 0, fmt.Errorf("bad timezone: %w", err)
	}
	newUntil, err := time.ParseInLocation("2006-01-02", untilStr, loc)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: bad until date", ErrInvalidInput)
	}
	occs, err := c.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, 0, err
	}
	if len(occs) == 0 {
		return 0, 0, nil
	}
	anchor := occs[0]
	rec := meeting.Recurrence(anchor.Recurrence)
	if rec == meeting.Once {
		return 0, 0, fmt.Errorf("%w: not a recurring series", ErrInvalidInput)
	}
	anchorStart := anchor.StartsAt.In(loc)
	anchorEnd := anchor.EndsAt.In(loc)
	candidate, err := meeting.Occurrences(anchorStart, anchorEnd, rec, anchor.RecurrenceDays, newUntil)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	existingStarts, err := c.Store.ListSeriesOccurrenceStarts(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, 0, err
	}
	plan := planSeriesReshape(occs, existingStarts, candidate, newUntil, loc)
	cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	var (
		calSvc     docalendar.Service
		createdIDs []string
		rows       []model.Meeting
		parts      []model.MeetingParticipant
	)
	if len(plan.Create) > 0 {
		var ferr error
		calSvc, ferr = c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, anchor.OrganizerUserID))
		if ferr != nil {
			return 0, 0, ferr
		}
		parts, err = c.Store.ListParticipants(ctx, anchor.ID)
		if err != nil {
			return 0, 0, err
		}
		var emails []string
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
		rows = make([]model.Meeting, 0, len(plan.Create))
		for _, sp := range plan.Create {
			name := meeting.GenerateName(anchor.Dept, anchor.Type, anchor.Host, sp.Start, rec)
			res, cerr := calSvc.CreateEvent(ctx, docalendar.CalendarEvent{
				Title: name, Description: anchor.Description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
			})
			if cerr != nil {
				c.deleteEventsBestEffort(ctx, calSvc, createdIDs)
				return 0, 0, fmt.Errorf("calendar: %w", cerr)
			}
			createdIDs = append(createdIDs, res.EventID)
			until := newUntil
			rows = append(rows, model.Meeting{
				OrganizationID: organizationID, OrganizerUserID: anchor.OrganizerUserID,
				Dept: anchor.Dept, Type: anchor.Type, Host: anchor.Host,
				StartsAt: sp.Start.UTC(), EndsAt: sp.End.UTC(),
				Recurrence: string(rec), RecurrenceDays: anchor.RecurrenceDays,
				Name: name, Description: anchor.Description,
				GoogleEventID: res.EventID, MeetLink: res.MeetLink,
				SeriesID: picked.SeriesID, RecurrenceUntil: &until,
			})
		}
	}

	added, removed, err := c.Store.ReshapeSeriesTx(ctx, organizationID, *picked.SeriesID, rows, parts, cutoff, newUntil)
	if err != nil {
		if len(createdIDs) > 0 && calSvc != nil {
			c.deleteEventsBestEffort(ctx, calSvc, createdIDs)
		}
		return 0, 0, err
	}

	if len(plan.CancelIDs) > 0 {
		cancelSet := make(map[uuid.UUID]bool, len(plan.CancelIDs))
		for _, id := range plan.CancelIDs {
			cancelSet[id] = true
		}
		if delSvc, ferr := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, anchor.OrganizerUserID)); ferr == nil {
			var ids []string
			for _, o := range occs {
				if cancelSet[o.ID] && o.GoogleEventID != "" {
					ids = append(ids, o.GoogleEventID)
				}
			}
			c.deleteEventsBestEffort(ctx, delSvc, ids)
		}
	}

	if c.Queue != nil {
		if err := c.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && c.Log != nil {
			c.Log.Warn("enqueue_series_end_changed",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return added, removed, nil
}
