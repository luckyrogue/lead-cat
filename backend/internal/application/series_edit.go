package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
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

// UpdateSeries applies a series-wide edit to the picked occurrence and all later
// ones (organizer or owner only): validates per occurrence, persists atomically,
// patches Google best-effort, and enqueues one change notification. Returns the
// number of occurrences updated.
func (s *Services) UpdateSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
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
	occs, err := s.Store.ListSeriesOccurrences(ctx, workspaceID, *picked.SeriesID, picked.StartsAt)
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
	if err := s.Store.UpdateMeetingsTx(ctx, workspaceID, rows); err != nil {
		return 0, err
	}
	// Google best-effort: DB is the source of truth; a failed patch is reconciled
	// by re-editing (series patches are not reversible, so we don't compensate).
	if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("series update calendar provider", zap.String("workspace_id", workspaceID.String()), zap.Error(ferr))
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
		if err := s.Queue.EnqueueMeetingUpdated(ctx, workspaceID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting updated",
				zap.String("workspace_id", workspaceID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return len(rows), nil
}
