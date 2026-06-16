package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
)

var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

var supportedLanguages = map[string]bool{"": true, "ru": true, "en": true, "kk": true}

type UserSettings struct {
	ReminderMinutes []int  `json:"reminder_minutes"`
	Timezone        string `json:"timezone"`
	Language        string `json:"language"`
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

func (s *Services) SetUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return s.Store.SetBotUserPrefs(ctx, telegramID, timezone, language)
}

type WebUserSettings struct {
	Timezone string `json:"timezone"`
	Language string `json:"language"`
}

func (s *Services) GetWebUserSettings(ctx context.Context, userID uuid.UUID) (WebUserSettings, error) {
	u, _, err := s.Store.GetPlatformUserByID(ctx, userID)
	if err != nil {
		return WebUserSettings{}, err
	}
	return WebUserSettings{Timezone: u.Timezone, Language: u.Language}, nil
}

func (s *Services) SetWebUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return s.Store.SetPlatformUserPrefs(ctx, userID, timezone, language)
}

func validatePrefs(timezone, language string) error {
	if !supportedLanguages[language] {
		return fmt.Errorf("%w: language", ErrInvalidInput)
	}
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err != nil {
			return fmt.Errorf("%w: timezone", ErrInvalidInput)
		}
	}
	return nil
}

func getUserSettings(ctx context.Context, store userSettingsStore, telegramID int64) (UserSettings, error) {
	u, err := store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return UserSettings{}, err
	}
	return UserSettings{
		ReminderMinutes: botsettings.Parse(u.ReminderMinutes),
		Timezone:        u.Timezone,
		Language:        u.Language,
	}, nil
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
