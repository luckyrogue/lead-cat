package reminder_scheduler

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
	"github.com/luckyrogue/lead-cat/internal/platform/meetingrecipients"
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
// their own settings; the organizer uses the default. Resolution is delegated to
// the shared meetingrecipients helper.
func (s *Scheduler) recipients(ctx context.Context, m postgres.Meeting) map[int64][]int {
	out := map[int64][]int{}
	recs, err := meetingrecipients.Resolve(ctx, s.store, m)
	if err != nil {
		s.log.Warn("resolve recipients", zap.String("meeting_id", m.ID.String()), zap.Error(err))
		return out
	}
	for _, r := range recs {
		if r.IsOrganizer {
			out[r.TelegramID] = defaultOrganizerOffsets
		} else {
			out[r.TelegramID] = botsettings.Parse(r.ReminderMinutes)
		}
	}
	return out
}
