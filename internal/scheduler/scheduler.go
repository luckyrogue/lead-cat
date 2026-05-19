package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/cats"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	commitsHour   = 18
	commitsMinute = 30

	meetHour   = 10
	meetMinute = 15
)

type Runner struct {
	bot     *bot.Bot
	chatID  int64
	loc     *time.Location
	meetURL string

	mu   sync.Mutex
	sent map[string]struct{}
}

func New(b *bot.Bot, chatID int64, loc *time.Location, meetURL string) *Runner {
	return &Runner{
		bot:     b,
		chatID:  chatID,
		loc:     loc,
		meetURL: meetURL,
		sent:    make(map[string]struct{}),
	}
}

func (r *Runner) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	now := time.Now().In(r.loc)
	wd := now.Weekday()

	if wd == time.Saturday || wd == time.Sunday {
		return
	}

	if now.Hour() == commitsHour && now.Minute() == commitsMinute {
		r.sendOnce(ctx, now, "commits", commitsMessage())
	}

	if (wd == time.Monday || wd == time.Wednesday || wd == time.Friday) &&
		now.Hour() == meetHour && now.Minute() == meetMinute {
		r.sendOnce(ctx, now, "meet", meetMessage(r.meetURL))
	}
}

func (r *Runner) sendOnce(ctx context.Context, now time.Time, kind, text string) {
	key := now.Format("2006-01-02") + ":" + kind

	r.mu.Lock()
	if _, ok := r.sent[key]; ok {
		r.mu.Unlock()
		return
	}
	r.sent[key] = struct{}{}
	cutoff := now.Add(-48 * time.Hour).Format("2006-01-02")
	for k := range r.sent {
		if k < cutoff {
			delete(r.sent, k)
		}
	}
	r.mu.Unlock()

	photoURL := cats.RandomImageURL(ctx)
	_, err := r.bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:    r.chatID,
		Photo:     &models.InputFileString{Data: photoURL},
		Caption:   text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		slog.Warn("photo send failed, fallback to text", "kind", kind, "err", err)
		_, err = r.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    r.chatID,
			Text:      text,
			ParseMode: models.ParseModeMarkdown,
		})
	}
	if err != nil {
		slog.Error("send notification", "kind", kind, "err", err)
		r.mu.Lock()
		delete(r.sent, key)
		r.mu.Unlock()
		return
	}
	slog.Info("notification sent", "kind", kind, "photo", photoURL)
}

func commitsMessage() string {
	return `📋 *18:30 — сдаём день*

Слежу. Закинь коммиты на бранчи и отчитайся техлиду. Не тяни хвост — вижу всё.`
}

func meetMessage(url string) string {
	return `📞 *10:15 — созвон*

Скоро мит. Лапы на клавиатуру — я смотрю:
` + url
}
