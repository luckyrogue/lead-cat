package meeting_notifier

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type sender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

type store interface {
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error)
	GetBotUserByEmail(ctx context.Context, email string) (model.BotUser, error)
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)
}

// claimer provides per-delivery idempotency: Claim returns true only the
// first time a key is seen, so a redelivered asynq task does not re-notify.
type claimer interface {
	Claim(ctx context.Context, key string) (bool, error)
}

var _ sender = (*bot.Bot)(nil)
