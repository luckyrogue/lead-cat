// Package stub is a no-network CalendarService used locally and in tests until
// the real Google Calendar adapter lands. It fabricates event IDs and Meet links.
package stub

import (
	"context"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) CreateEvent(_ context.Context, _ application.CalendarEvent) (application.CalendarResult, error) {
	id := uuid.NewString()
	return application.CalendarResult{
		EventID:  id,
		MeetLink: "https://meet.google.com/stub-" + id[:8],
	}, nil
}

func (s *Service) DeleteEvent(_ context.Context, _ string) error { return nil }
