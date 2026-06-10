package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// SeriesUpdateInput carries series-wide field overrides (nil = unchanged) plus an
// optional time-of-day change (Start/End HH:MM applied to each occurrence's own
// date). Date and recurrence pattern are not changed series-wide.
type SeriesUpdateInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Description *string
	Start       *string // HH:MM
	End         *string // HH:MM
}

// applySeriesUpdate applies field overrides + an optional time-of-day to one
// occurrence, keeping the occurrence's own date and recurrence. Pure; recomputes
// the name. Returns ErrInvalidInput on bad time.
func applySeriesUpdate(cur postgres.Meeting, in SeriesUpdateInput, loc *time.Location) (postgres.Meeting, error) {
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
			return postgres.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", day+" "+*in.End, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	rec := meeting.Recurrence(cur.Recurrence)
	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}

// Internal: not currently exposed via TMA HTTP (slice B uses UpdateWholeSeries/CancelWholeSeries). Slice E admin scope may revive for "from-forward" semantics.
// UpdateSeries applies a series-wide edit to the picked occurrence and all later
// ones (organizer or owner only): validates per occurrence, persists atomically,
// patches Google best-effort, and enqueues one change notification. Returns the
// number of occurrences updated.
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
	rows := make([]postgres.Meeting, 0, len(occs))
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
		for _, m := range rows {
			if m.GoogleEventID == "" {
				continue
			}
			if err := calSvc.UpdateEvent(ctx, m.GoogleEventID, CalendarEvent{
				Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
			}); err != nil && s.Log != nil {
				s.Log.Warn("series update event", zap.String("event_id", m.GoogleEventID), zap.Error(err))
			}
		}
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

// Internal: not currently exposed via TMA HTTP (slice B uses UpdateWholeSeries/CancelWholeSeries). Slice E admin scope may revive for "from-forward" semantics.
// CancelSeries cancels the picked occurrence and all later scheduled ones of its
// series (organizer or owner only): cancels in one atomic UPDATE, deletes the
// Google events best-effort, and enqueues one cancellation notification. Returns
// the number of occurrences cancelled.
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

// UpdateWholeSeries applies a series-wide edit to EVERY occurrence in the series
// (including past ones), keyed by series_id. Auth: organizer or organization owner.
// Returns the count of occurrences updated.
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
	rows := make([]postgres.Meeting, 0, len(occs))
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
		for _, m := range rows {
			if m.GoogleEventID == "" {
				continue
			}
			if err := calSvc.UpdateEvent(ctx, m.GoogleEventID, CalendarEvent{
				Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
			}); err != nil && s.Log != nil {
				s.Log.Warn("whole_series_update_event", zap.String("event_id", m.GoogleEventID), zap.Error(err))
			}
		}
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

// CancelWholeSeries cancels EVERY occurrence in the series (including past ones),
// keyed by series_id. Auth: organizer or organization owner. Returns the count.
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
