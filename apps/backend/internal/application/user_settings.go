package application

import (
	"context"
	"errors"
	"slices"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
)

// ErrInvalidReminderMinute is returned when a minute value is not in the
// botsettings.Intervals whitelist.
var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

// UserSettings is the per-user settings projection exposed to the Mini App.
type UserSettings struct {
	ReminderMinutes []int `json:"reminder_minutes"`
}

// userSettingsStore is the narrow store interface used by GetUserSettings /
// SetUserReminderMinutes — defined here so unit tests can mock it.
type userSettingsStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
}

// GetUserSettings returns the authed bot user's settings.
func (s *Services) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	return getUserSettings(ctx, s.Store, telegramID)
}

// SetUserReminderMinutes validates input against the whitelist, dedupes/sorts,
// and writes canonical CSV.
func (s *Services) SetUserReminderMinutes(ctx context.Context, telegramID int64, minutes []int) error {
	return setUserReminderMinutes(ctx, s.Store, telegramID, minutes)
}

func getUserSettings(ctx context.Context, store userSettingsStore, telegramID int64) (UserSettings, error) {
	u, err := store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return UserSettings{}, err
	}
	return UserSettings{ReminderMinutes: botsettings.Parse(u.ReminderMinutes)}, nil
}

func setUserReminderMinutes(ctx context.Context, store userSettingsStore, telegramID int64, minutes []int) error {
	allowed := allowedReminderMinutes()
	for _, m := range minutes {
		if !slices.Contains(allowed, m) {
			return ErrInvalidReminderMinute
		}
	}
	cp := append([]int(nil), minutes...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	return store.SetReminderMinutes(ctx, telegramID, botsettings.Format(cp))
}

func allowedReminderMinutes() []int {
	out := make([]int, 0, len(botsettings.Intervals))
	for _, iv := range botsettings.Intervals {
		out = append(out, iv.Minutes)
	}
	return out
}
