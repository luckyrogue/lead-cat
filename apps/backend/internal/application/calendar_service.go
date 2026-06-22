package application

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/query"
)

var ErrUnknownConnector = command.ErrUnknownConnector

type CalendarConnectionView = query.CalendarConnectionView

func (s *Services) StartCalendarConnect(ctx context.Context, email, provider, redirectURL string) (string, error) {
	return s.CalendarConnectCommands.StartCalendarConnect(ctx, email, provider, redirectURL)
}

func (s *Services) FinishCalendarConnect(ctx context.Context, state, code, redirectURL string) error {
	return s.CalendarConnectCommands.FinishCalendarConnect(ctx, state, code, redirectURL)
}

func (s *Services) DisconnectCalendar(ctx context.Context, email, provider string) error {
	return s.CalendarConnectCommands.DisconnectCalendar(ctx, email, provider)
}

func (s *Services) ListCalendarConnections(ctx context.Context, email string) ([]CalendarConnectionView, error) {
	return s.CalendarConnectQueries.ListCalendarConnections(ctx, email)
}
