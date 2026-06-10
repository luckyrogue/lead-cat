package botsettings

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type store interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
}

type Service struct{ store store }

func New(s store) *Service { return &Service{store: s} }

// Settings renders the current reminder keyboard for a user.
func (s *Service) Settings(ctx context.Context, telegramID int64) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	text, kb := render(parse(u.ReminderMinutes))
	return text, kb, nil
}

// Toggle flips one interval, persists, and returns the refreshed keyboard.
func (s *Service) Toggle(ctx context.Context, telegramID int64, minutes int) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	next := toggle(parse(u.ReminderMinutes), minutes)
	if err := s.store.SetReminderMinutes(ctx, telegramID, format(next)); err != nil {
		return "", nil, err
	}
	text, kb := render(next)
	return text, kb, nil
}
