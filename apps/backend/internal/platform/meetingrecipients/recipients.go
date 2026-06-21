package meetingrecipients

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type Store interface {
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
}

type Recipient struct {
	TelegramID      int64
	ReminderMinutes string
	IsOrganizer     bool
	FullName        string
	Email           string
	Language        string
	Timezone        string
}

func Resolve(ctx context.Context, store Store, m postgres.Meeting) ([]Recipient, error) {
	var out []Recipient
	seen := map[int64]bool{}

	parts, err := store.ListParticipants(ctx, m.ID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	for _, p := range parts {
		if p.Email == "" {
			continue
		}
		u, err := store.GetBotUserByEmail(ctx, p.Email)
		if err != nil {
			continue
		}
		if seen[u.TelegramID] {
			continue
		}
		seen[u.TelegramID] = true
		out = append(out, Recipient{
			TelegramID:      u.TelegramID,
			ReminderMinutes: u.ReminderMinutes,
			FullName:        u.FullName,
			Email:           u.Email,
			Language:        u.Language,
			Timezone:        u.Timezone,
		})
	}

	if m.OrganizerUserID != nil {
		if tg, linked, err := store.GetUserTelegramID(ctx, *m.OrganizerUserID); err == nil && linked {
			if !seen[tg] {
				seen[tg] = true
				r := Recipient{TelegramID: tg, IsOrganizer: true}
				if bu, err := store.GetBotUserByTelegramID(ctx, tg); err == nil {
					r.FullName, r.Email, r.Language, r.Timezone = bu.FullName, bu.Email, bu.Language, bu.Timezone
				}
				out = append(out, r)
			}
		}
	}
	return out, nil
}
