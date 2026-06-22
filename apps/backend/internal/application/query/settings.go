package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
)

type UserSettings struct {
	ReminderMinutes []int  `json:"reminder_minutes"`
	Timezone        string `json:"timezone"`
	Language        string `json:"language"`
}

type WebUserSettings struct {
	Timezone string `json:"timezone"`
	Language string `json:"language"`
}

type settingsStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error)
	GetPlatformUserByID(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error)
}

type Settings struct {
	Store settingsStore
}

func NewSettings(store settingsStore) *Settings {
	return &Settings{Store: store}
}

func (q *Settings) GetUserSettings(ctx context.Context, telegramID int64) (UserSettings, error) {
	u, err := q.Store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return UserSettings{}, err
	}
	return UserSettings{
		ReminderMinutes: botsettings.Parse(u.ReminderMinutes),
		Timezone:        u.Timezone,
		Language:        u.Language,
	}, nil
}

func (q *Settings) GetWebUserSettings(ctx context.Context, userID uuid.UUID) (WebUserSettings, error) {
	u, _, err := q.Store.GetPlatformUserByID(ctx, userID)
	if err != nil {
		return WebUserSettings{}, err
	}
	return WebUserSettings{Timezone: u.Timezone, Language: u.Language}, nil
}
