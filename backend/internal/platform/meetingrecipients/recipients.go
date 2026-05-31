// Package meetingrecipients resolves the Telegram recipients of a meeting:
// registered participants (by email) plus the organizer (if their Telegram is
// linked). Shared by the reminder engine and the meeting-created notifier.
package meetingrecipients

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Store is the subset of *postgres.Store this package needs.
type Store interface {
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
}

// Recipient is one notification target.
type Recipient struct {
	TelegramID      int64
	ReminderMinutes string // populated only when resolved via bot_users (participant path)
	IsOrganizer     bool
}

// Resolve returns the meeting's recipients: registered participants (skipping
// those without a bot_users record) plus the organizer when linked and not
// already a participant. Order: participants first, organizer last.
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
		out = append(out, Recipient{TelegramID: u.TelegramID, ReminderMinutes: u.ReminderMinutes})
	}

	if m.OrganizerUserID != nil {
		if tg, linked, err := store.GetUserTelegramID(ctx, *m.OrganizerUserID); err == nil && linked {
			if !seen[tg] {
				seen[tg] = true
				out = append(out, Recipient{TelegramID: tg, IsOrganizer: true})
			}
		}
	}
	return out, nil
}
