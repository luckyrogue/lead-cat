package telegram

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
	"github.com/luckyrogue/lead-cat/internal/platform/scheduler_agent"
)

type agentBooker struct {
	store    *postgres.Store
	services *application.Services
}

var _ scheduler_agent.Booker = (*agentBooker)(nil)

func (b *agentBooker) Book(ctx context.Context, telegramID int64, pb scheduler_agent.PendingBooking, lang string) (string, error) {
	fail := func(cause error, userMsg string) (string, error) {
		if b.services.Log != nil {
			b.services.Log.Warn("agent_book_failed", zap.Int64("telegram_id", telegramID), zap.Error(cause))
		}
		return "", fmt.Errorf("%s", userMsg)
	}

	bu, err := b.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fail(err, boti18n.T(lang, "agentbook.register_first"))
	}
	organizationID, err := b.services.ResolveMiniAppOrganization(ctx)
	if err != nil {
		if errors.Is(err, application.ErrGoogleNotConfigured) {
			return fail(err, boti18n.T(lang, "agentbook.google_not_configured"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
	}
	organizerID, err := b.services.EnsureMiniAppOrganizer(ctx, bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fail(err, boti18n.T(lang, "agentbook.telegram_linked_elsewhere"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
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
			return fail(err, boti18n.T(lang, "agentbook.bad_input"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
	}
	if m.MeetLink != "" {
		return boti18n.T(lang, "agentbook.created") + "\n" + m.MeetLink, nil
	}
	return boti18n.T(lang, "agentbook.created"), nil
}
