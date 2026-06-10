// Package meeting_notifier sends Telegram DMs when a meeting is created or updated.
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
	"github.com/luckyrogue/lead-cat/internal/platform/meetingrecipients"
)

type Notifier struct {
	store *postgres.Store
	bot   *bot.Bot
	log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, log *zap.Logger) *Notifier {
	return &Notifier{store: store, bot: b, log: log}
}

// HandleCreated DMs the meeting's recipients. Returns an error only when the
// meeting/workspace/recipients cannot be read (asynq should retry); a single
// failed send is logged and skipped so a retry does not re-DM everyone else.
func (n *Notifier) HandleCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	loc, err := time.LoadLocation(cmp.Or(w.TZ, "Asia/Almaty"))
	if err != nil {
		n.log.Warn("load location", zap.String("tz", w.TZ), zap.Error(err))
		loc = time.UTC
	}
	text := buildMessage(m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)

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

// HandleParticipantAdded DMs a newly-added participant (if they have a bot account).
func (n *Notifier) HandleParticipantAdded(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error {
	return n.notifyParticipant(ctx, organizationID, meetingID, email, true)
}

// HandleParticipantRemoved DMs a removed participant (if they have a bot account).
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
	loc, err := time.LoadLocation(cmp.Or(w.TZ, "Asia/Almaty"))
	if err != nil {
		n.log.Warn("load location", zap.String("tz", w.TZ), zap.Error(err))
		loc = time.UTC
	}
	u, err := n.store.GetBotUserByEmail(ctx, email)
	if postgres.IsNotFound(err) {
		return nil // not a bot user — the Google email invitation/cancellation covers them
	}
	if err != nil {
		return fmt.Errorf("get bot user: %w", err) // transient error — let asynq retry
	}
	var text string
	if added {
		text = buildEventMessage("➕ Вас добавили на встречу", m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
	} else {
		text = buildRemovedMessage(m.Name, m.StartsAt, loc)
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

// HandleCancelled DMs the meeting's recipients that it was cancelled. The meeting
// is already status='cancelled' but GetMeeting has no status filter. Best-effort
// sends; returns an error only on read failures (asynq retries before any send).
func (n *Notifier) HandleCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	loc, err := time.LoadLocation(cmp.Or(w.TZ, "Asia/Almaty"))
	if err != nil {
		n.log.Warn("load location", zap.String("tz", w.TZ), zap.Error(err))
		loc = time.UTC
	}
	text := buildCancelledMessage(m.Name, m.StartsAt, loc)

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
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

// HandleUpdated DMs the meeting's recipients that it changed. Like HandleCreated
// it returns an error only on read failures (asynq retries before any send);
// sends are best-effort. No dedup: each edit is its own notification.
func (n *Notifier) HandleUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetOrganization(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	loc, err := time.LoadLocation(cmp.Or(w.TZ, "Asia/Almaty"))
	if err != nil {
		n.log.Warn("load location", zap.String("tz", w.TZ), zap.Error(err))
		loc = time.UTC
	}
	text := buildUpdatedMessage(m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
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
