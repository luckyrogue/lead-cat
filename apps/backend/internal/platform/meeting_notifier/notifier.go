package meeting_notifier

import (
	"cmp"
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
	"github.com/luckyrogue/lead-cat/internal/platform/meetingrecipients"
)

type Notifier struct {
	store store
	bot   sender
	log   *zap.Logger
}

func New(st store, b sender, log *zap.Logger) *Notifier {
	return &Notifier{store: st, bot: b, log: log}
}

// recipientLoc resolves a recipient's display timezone, falling back to the
// organization TZ then Asia/Almaty; on a load error it warns and uses UTC.
func (n *Notifier) recipientLoc(recipientTZ, orgTZ string) *time.Location {
	tz := cmp.Or(recipientTZ, orgTZ, "Asia/Almaty")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		n.log.Warn("load location", zap.String("tz", tz), zap.Error(err))
		return time.UTC
	}
	return loc
}

func (n *Notifier) HandleCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		claimed, err := n.store.TryClaimReminder(ctx, m.ID, r.TelegramID, postgres.ReminderOffsetCreated)
		if err != nil {
			return fmt.Errorf("claim reminder: %w", err)
		}
		if !claimed {
			continue
		}
		loc := n.recipientLoc(r.Timezone, w.TZ)
		text := buildMessage(r.Language, m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting created",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

func (n *Notifier) HandleParticipantAdded(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error {
	return n.notifyParticipant(ctx, organizationID, meetingID, email, true)
}

func (n *Notifier) HandleParticipantRemoved(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error {
	return n.notifyParticipant(ctx, organizationID, meetingID, email, false)
}

func (n *Notifier) notifyParticipant(ctx context.Context, organizationID, meetingID uuid.UUID, email string, added bool) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	u, err := n.store.GetBotUserByEmail(ctx, email)
	if postgres.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get bot user: %w", err)
	}
	loc := n.recipientLoc(u.Timezone, w.TZ)
	var text string
	if added {
		text = buildEventMessage(boti18n.T(u.Language, "notif.added"), m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
	} else {
		text = buildRemovedMessage(u.Language, m.Name, m.StartsAt, loc)
	}
	if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: u.TelegramID, Text: text}); err != nil {
		n.log.Warn("send participant notice",
			zap.Int64("telegram_id", u.TelegramID),
			zap.String("meeting_id", m.ID.String()),
			zap.Bool("added", added),
			zap.String("email", email),
			zap.Error(err))
	}
	return nil
}

func (n *Notifier) HandleCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		loc := n.recipientLoc(r.Timezone, w.TZ)
		text := buildCancelledMessage(r.Language, m.Name, m.StartsAt, loc)
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting cancelled",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

func (n *Notifier) HandleUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		loc := n.recipientLoc(r.Timezone, w.TZ)
		text := buildUpdatedMessage(r.Language, m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting updated",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}
