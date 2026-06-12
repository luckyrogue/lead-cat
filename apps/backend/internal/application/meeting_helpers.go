package application

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/fanio"
)

func orStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

func (s *Services) deleteEventsBestEffort(ctx context.Context, cal CalendarService, ids []string) {
	fanio.EachStringBestEffort(ctx, fanio.CalendarLimit, ids, func(ctx context.Context, id string) {
		if err := cal.DeleteEvent(ctx, id); err != nil && s.Log != nil {
			s.Log.Warn("delete orphan meeting event", zap.String("event_id", id), zap.Error(err))
		}
	})
}

func (s *Services) updateCalendarEventsBestEffort(ctx context.Context, cal CalendarService, rows []model.Meeting, logMsg string) {
	fanio.AllBestEffort(ctx, fanio.CalendarLimit, len(rows), func(ctx context.Context, i int) {
		m := rows[i]
		if m.GoogleEventID == "" {
			return
		}
		if err := cal.UpdateEvent(ctx, m.GoogleEventID, CalendarEvent{
			Title: m.Name, Description: m.Description, Start: m.StartsAt, End: m.EndsAt,
		}); err != nil && s.Log != nil {
			s.Log.Warn(logMsg, zap.String("event_id", m.GoogleEventID), zap.Error(err))
		}
	})
}

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
