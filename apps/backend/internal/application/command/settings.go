package command

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
)

var ErrInvalidReminderMinute = errors.New("invalid_reminder_minute")

var supportedLanguages = map[string]bool{"": true, "ru": true, "en": true, "kk": true}

type settingsStore interface {
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
	SetBotUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error
	SetPlatformUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error
}

type Settings struct {
	Store settingsStore
}

func (c *Settings) SetUserReminderMinutes(ctx context.Context, telegramID int64, minutes []int) error {
	allowed := allowedReminderMinutes()
	for _, m := range minutes {
		if !slices.Contains(allowed, m) {
			return ErrInvalidReminderMinute
		}
	}
	cp := append([]int(nil), minutes...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	return c.Store.SetReminderMinutes(ctx, telegramID, botsettings.Format(cp))
}

func (c *Settings) SetUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return c.Store.SetBotUserPrefs(ctx, telegramID, timezone, language)
}

func (c *Settings) SetWebUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error {
	if err := validatePrefs(timezone, language); err != nil {
		return err
	}
	return c.Store.SetPlatformUserPrefs(ctx, userID, timezone, language)
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

func allowedReminderMinutes() []int {
	out := make([]int, 0, len(botsettings.Intervals))
	for _, iv := range botsettings.Intervals {
		out = append(out, iv.Minutes)
	}
	return out
}
