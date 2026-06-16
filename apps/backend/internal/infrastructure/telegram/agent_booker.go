package telegram

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/scheduler_agent"
)

// agentBooker creates a one-off meeting for the authenticated Telegram user.
// Org + organizer are resolved from the user's account here — never from the
// model. The returned error carries a user-facing message (the agent Service
// surfaces err.Error() to the user); the real cause is logged.
type agentBooker struct {
	store    *postgres.Store
	services *application.Services
}

var _ scheduler_agent.Booker = (*agentBooker)(nil)

func (b *agentBooker) Book(ctx context.Context, telegramID int64, pb scheduler_agent.PendingBooking) (string, error) {
	fail := func(cause error, userMsg string) (string, error) {
		if b.services.Log != nil {
			b.services.Log.Warn("agent_book_failed", zap.Int64("telegram_id", telegramID), zap.Error(cause))
		}
		return "", fmt.Errorf("%s", userMsg)
	}

	bu, err := b.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fail(err, "Сначала зарегистрируйся: /start")
	}
	organizationID, err := b.services.ResolveMiniAppOrganization(ctx)
	if err != nil {
		if errors.Is(err, application.ErrGoogleNotConfigured) {
			return fail(err, "Google-календарь не подключён — обратись к администратору.")
		}
		return fail(err, "Не удалось создать встречу, попробуй позже 🐾")
	}
	organizerID, err := b.services.EnsureMiniAppOrganizer(ctx, bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fail(err, "Этот Telegram привязан к другому аккаунту.")
		}
		return fail(err, "Не удалось создать встречу, попробуй позже 🐾")
	}
	parts := make([]model.MeetingParticipant, 0, len(pb.Emails))
	for _, e := range pb.Emails {
		parts = append(parts, model.MeetingParticipant{Email: e})
	}
	in := application.CreateMeetingInput{
		Dept: pb.Dept, Type: pb.Type, Host: bu.FullName,
		Date: pb.Date, Start: pb.Start, End: pb.End,
		Recurrence: "once", Description: pb.Desc,
		Participants: parts, Timezone: bu.Timezone,
	}
	m, err := b.services.CreateMeeting(ctx, organizationID, organizerID, in)
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) {
			return fail(err, "Проверь данные встречи — что-то не так с датой или временем.")
		}
		return fail(err, "Не удалось создать встречу, попробуй позже 🐾")
	}
	if m.MeetLink != "" {
		return "Встреча создана ✅\n" + m.MeetLink, nil
	}
	return "Встреча создана ✅", nil
}
