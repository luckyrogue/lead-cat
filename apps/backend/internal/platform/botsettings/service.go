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

func (s *Service) Settings(ctx context.Context, telegramID int64, lang string) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	text, kb := render(parse(u.ReminderMinutes), lang)
	return text, kb, nil
}

func (s *Service) Toggle(ctx context.Context, telegramID int64, minutes int, lang string) (string, [][]Button, error) {
	u, err := s.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	next := toggle(parse(u.ReminderMinutes), minutes)
	if err := s.store.SetReminderMinutes(ctx, telegramID, format(next)); err != nil {
		return "", nil, err
	}
	text, kb := render(next, lang)
	return text, kb, nil
}
