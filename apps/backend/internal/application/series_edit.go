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

	if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr != nil {
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

	if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr != nil {
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
	if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr != nil {
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
	if calSvc, ferr := s.Calendar.For(ctx, organizationID); ferr != nil {
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
