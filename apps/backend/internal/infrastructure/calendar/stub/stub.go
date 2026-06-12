package stub

import (
	"context"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) CreateEvent(_ context.Context, _ docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	id := uuid.NewString()
	return docalendar.CalendarResult{
		EventID:  id,
		MeetLink: "https://meet.google.com/stub-" + id[:8],
	}, nil
}

func (s *Service) UpdateEvent(_ context.Context, _ string, _ docalendar.CalendarEvent) error {
	return nil
}

func (s *Service) UpdateAttendees(_ context.Context, _ string, _ []string) error { return nil }

func (s *Service) DeleteEvent(_ context.Context, _ string) error { return nil }
