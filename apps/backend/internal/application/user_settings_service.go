package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/query"
)

type UserSettings = query.UserSettings

type WebUserSettings = query.WebUserSettings

var ErrInvalidReminderMinute = command.ErrInvalidReminderMinute

func (s *Services) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	return s.SettingsQueries.GetUserSettings(ctx, telegramID)
}

func (s *Services) GetWebUserSettings(ctx context.Context, userID uuid.UUID) (WebUserSettings, error) {
	return s.SettingsQueries.GetWebUserSettings(ctx, userID)
}

func (s *Services) SetUserReminderMinutes(ctx context.Context, telegramID int64, minutes []int) error {
	return s.SettingsCommands.SetUserReminderMinutes(ctx, telegramID, minutes)
}

func (s *Services) SetUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	return s.SettingsCommands.SetUserPrefs(ctx, telegramID, timezone, language)
}

func (s *Services) SetWebUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error {
	return s.SettingsCommands.SetWebUserPrefs(ctx, userID, timezone, language)
}
