package reminder_scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
	"github.com/luckyrogue/lead-cat/internal/platform/emailtemplates"
	"github.com/luckyrogue/lead-cat/internal/platform/meetingrecipients"
)

const lockKey = "leadcat:reminders:leader"

var defaultOrganizerOffsets = []int{15}

// Scheduler sends meeting reminders via Telegram and optionally email.
type Scheduler struct {
	store          *postgres.Store
	bot            *bot.Bot
	rdb            *redis.Client
	email          application.EmailSender
	emailEnabled   bool
	unsubscribeURL string
	log            *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, rdb *redis.Client, email application.EmailSender, emailEnabled bool, unsubscribeURL string, log *zap.Logger) *Scheduler {
	return &Scheduler{
		store:          store,
		bot:            b,
		rdb:            rdb,
		email:          email,
		emailEnabled:   emailEnabled,
		unsubscribeURL: unsubscribeURL,
		log:            log,
	}
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
		for _, t := range s.recipients(ctx, m) {
			for _, off := range dueOffsets(now, m.StartsAt, t.Offsets) {
				claimed, err := s.store.TryClaimReminder(ctx, m.ID, t.TelegramID, off)
				if err != nil || !claimed {
					continue
				}

				if _, err := s.bot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: t.TelegramID,
					Text:   message(m.Name, m.MeetLink, off),
				}); err != nil {
					s.log.Warn("send reminder",
						zap.Int64("telegram_id", t.TelegramID),
						zap.String("meeting_id", m.ID.String()),
						zap.Error(err))
				}

				if s.emailEnabled && t.Email != "" {
					s.sendReminderEmail(ctx, m, t)
				}
			}
		}
	}
}

type reminderTarget struct {
	TelegramID int64
	Offsets    []int
	FullName   string
	Email      string
	Language   string
	Timezone   string
}

func (s *Scheduler) recipients(ctx context.Context, m postgres.Meeting) []reminderTarget {
	recs, err := meetingrecipients.Resolve(ctx, s.store, m)
	if err != nil {
		s.log.Warn("resolve recipients", zap.String("meeting_id", m.ID.String()), zap.Error(err))
		return nil
	}
	out := make([]reminderTarget, 0, len(recs))
	for _, r := range recs {
		offsets := botsettings.Parse(r.ReminderMinutes)
		if r.IsOrganizer {
			offsets = defaultOrganizerOffsets
		}
		out = append(out, reminderTarget{
			TelegramID: r.TelegramID,
			Offsets:    offsets,
			FullName:   r.FullName,
			Email:      r.Email,
			Language:   r.Language,
			Timezone:   r.Timezone,
		})
	}
	return out
}

// sendReminderEmail renders and sends the HTML reminder in the recipient's
// timezone and language. Best-effort: failures are logged, not retried (the
// reminder claim already covers this recipient+offset).
func (s *Scheduler) sendReminderEmail(ctx context.Context, m postgres.Meeting, t reminderTarget) {
	loc, err := time.LoadLocation(t.Timezone)
	if t.Timezone == "" || err != nil {
		if loc, err = time.LoadLocation("Asia/Almaty"); err != nil {
			loc = time.FixedZone("Almaty", 5*3600)
		}
	}
	start, end := m.StartsAt.In(loc), m.EndsAt.In(loc)
	timeStr := start.Format("15:04") + "–" + end.Format("15:04")

	data := emailtemplates.ReminderData{
		Language:         t.Language,
		Name:             t.FullName,
		Title:            m.Name,
		Date:             start.Format("02.01.2006"),
		Time:             timeStr,
		Tz:               tzLabel(start),
		MeetLink:         m.MeetLink,
		UnsubscribeURL:   s.unsubscribeURL,
	}
	subject, textBody, htmlBody, rerr := emailtemplates.RenderReminder(data)
	if rerr != nil {
		s.log.Warn("render reminder email", zap.String("meeting_id", m.ID.String()), zap.Error(rerr))
		return
	}
	if err := s.email.SendMultipart(ctx, t.Email, subject, textBody, htmlBody, s.unsubscribeURL); err != nil {
		s.log.Warn("send reminder email",
			zap.String("email", t.Email),
			zap.String("meeting_id", m.ID.String()),
			zap.Error(err))
	}
}

func tzLabel(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	h, mn := off/3600, (off%3600)/60
	if mn == 0 {
		return fmt.Sprintf("UTC%s%d", sign, h)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, h, mn)
}
