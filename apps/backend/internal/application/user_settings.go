package application

import (
	"context"
	"errors"
	"slices"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
)

var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

type UserSettings struct {
	ReminderMinutes []int `json:"reminder_minutes"`
}

type userSettingsStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
}

func (s *Services) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	return getUserSettings(ctx, s.Store, telegramID)
}

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
