package reminder_scheduler

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/botsettings"
)

const lockKey = "leadcat:reminders:leader"

// defaultOrganizerOffsets is used for the organizer (who has no bot_users
// reminder settings).
var defaultOrganizerOffsets = []int{15}

type Scheduler struct {
	store *postgres.Store
	bot   *bot.Bot
	rdb   *redis.Client
	log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, rdb *redis.Client, log *zap.Logger) *Scheduler {
	return &Scheduler{store: store, bot: b, rdb: rdb, log: log}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	ok, err := s.rdb.SetNX(ctx, lockKey, "1", 90*time.Second).Result()
	if err != nil || !ok {
		return
	}
	now := time.Now()
	meetings, err := s.store.ListUpcomingMeetings(ctx, now.Add(24*time.Hour))
	if err != nil {
		s.log.Warn("list upcoming meetings", zap.Error(err))
		return
	}
	for _, m := range meetings {
		recipients := s.recipients(ctx, m)
		for tg, offsets := range recipients {
			for _, off := range dueOffsets(now, m.StartsAt, offsets) {
				claimed, err := s.store.TryClaimReminder(ctx, m.ID, tg, off)
				if err != nil || !claimed {
					continue
				}
				// Claim is durable; if the send fails the reminder is lost
				// (best-effort) — log so a systematically broken bot is visible.
				if _, err := s.bot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: tg,
					Text:   message(m.Name, m.MeetLink, off),
				}); err != nil {
					s.log.Warn("send reminder",
						zap.Int64("telegram_id", tg),
						zap.String("meeting_id", m.ID.String()),
						zap.Error(err))
				}
			}
		}
	}
}

// recipients maps telegram_id -> reminder offsets: registered participants use
// their own settings; the organizer (if linked and not already a participant)
// uses the default.
func (s *Scheduler) recipients(ctx context.Context, m postgres.Meeting) map[int64][]int {
	out := map[int64][]int{}
	parts, err := s.store.ListParticipants(ctx, m.ID)
	if err != nil {
		return out
	}
	for _, p := range parts {
		if p.Email == "" {
			continue
		}
		u, err := s.store.GetBotUserByEmail(ctx, p.Email)
		if err != nil {
			continue
		}
		out[u.TelegramID] = botsettings.Parse(u.ReminderMinutes)
	}
	if m.OrganizerUserID != nil {
		if tg, linked, err := s.store.GetUserTelegramID(ctx, *m.OrganizerUserID); err == nil && linked {
			if _, exists := out[tg]; !exists {
				out[tg] = defaultOrganizerOffsets
			}
		}
	}
	return out
}
