package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
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

func (s *Services) UpdateSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
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
	occs, err := s.Store.ListSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
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
	if err := s.Store.UpdateMeetingsTx(ctx, organizationID, rows); err != nil {
		return 0, err
	}

	if calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("series update calendar provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		s.updateCalendarEventsBestEffort(ctx, calSvc, rows, "series update event")
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting updated",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}

func (s *Services) CancelSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := s.Store.ListSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	n, err := s.Store.CancelSeriesOccurrences(ctx, organizationID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}

	if calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("cancel series calendar provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		s.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	s.enqueueCancelled(ctx, organizationID, meetingID)
	return n, nil
}

func (s *Services) UpdateWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
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
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
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
	if err := s.Store.UpdateMeetingsTx(ctx, organizationID, rows); err != nil {
		return 0, err
	}
	if calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("whole_series_update_calendar_provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		s.updateCalendarEventsBestEffort(ctx, calSvc, rows, "whole_series_update_event")
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue_meeting_updated_whole_series",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}

func (s *Services) CancelWholeSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	n, err := s.Store.CancelAllSeriesOccurrences(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, err
	}
	if calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, picked.OrganizerUserID)); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("whole_series_cancel_calendar_provider", zap.String("organization_id", organizationID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		s.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	s.enqueueCancelled(ctx, organizationID, meetingID)
	return n, nil
}

func (s *Services) ChangeSeriesEnd(ctx context.Context, organizationID, userID, meetingID uuid.UUID, untilStr string) (int, int, error) {
	picked, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return 0, 0, err
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
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
	occs, err := s.Store.ListSeriesAllOccurrences(ctx, organizationID, *picked.SeriesID)
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
	existingStarts, err := s.Store.ListSeriesOccurrenceStarts(ctx, organizationID, *picked.SeriesID)
	if err != nil {
		return 0, 0, err
	}
	plan := planSeriesReshape(occs, existingStarts, candidate, newUntil, loc)

	added := 0
	if len(plan.Create) > 0 {
		calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, anchor.OrganizerUserID))
		if ferr != nil {
			return 0, 0, ferr
		}
		parts, perr := s.Store.ListParticipants(ctx, anchor.ID)
		if perr != nil {
			return 0, 0, perr
		}
		var emails []string
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
		var createdIDs []string
		rows := make([]model.Meeting, 0, len(plan.Create))
		for _, sp := range plan.Create {
			name := meeting.GenerateName(anchor.Dept, anchor.Type, anchor.Host, sp.Start, rec)
			res, cerr := calSvc.CreateEvent(ctx, CalendarEvent{
				Title: name, Description: anchor.Description, Start: sp.Start, End: sp.End, AttendeeEmails: emails,
			})
			if cerr != nil {
				s.deleteEventsBestEffort(ctx, calSvc, createdIDs)
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
		if _, serr := s.Store.CreateMeetingSeries(ctx, rows, parts); serr != nil {
			s.deleteEventsBestEffort(ctx, calSvc, createdIDs)
			return 0, 0, serr
		}
		added = len(rows)
	}

	removed := 0
	if len(plan.CancelIDs) > 0 {
		cancelSet := make(map[uuid.UUID]bool, len(plan.CancelIDs))
		for _, id := range plan.CancelIDs {
			cancelSet[id] = true
		}
		cutoff := time.Date(newUntil.Year(), newUntil.Month(), newUntil.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
		if calSvc, ferr := s.Calendar.For(ctx, organizationID, s.organizerEmail(ctx, anchor.OrganizerUserID)); ferr == nil {
			var ids []string
			for _, o := range occs {
				if cancelSet[o.ID] && o.GoogleEventID != "" {
					ids = append(ids, o.GoogleEventID)
				}
			}
			s.deleteEventsBestEffort(ctx, calSvc, ids)
		}
		n, cerr := s.Store.CancelSeriesOccurrences(ctx, organizationID, *picked.SeriesID, cutoff)
		if cerr != nil {
			return added, 0, cerr
		}
		removed = n
	}

	if err := s.Store.SetSeriesRecurrenceUntil(ctx, organizationID, *picked.SeriesID, newUntil); err != nil {
		return added, removed, err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, organizationID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue_series_end_changed",
				zap.String("organization_id", organizationID.String()),
				zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return added, removed, nil
}
