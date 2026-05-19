package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/cats"
	"github.com/Jaryq-Lab/notify-bot/internal/commitsreport"
	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/github"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	commitsHour   = 18
	commitsMinute = 30

	commitsReportHour   = 18
	commitsReportMinute = 35

	meetHour   = 10
	meetMinute = 15
)

type Runner struct {
	bot     *bot.Bot
	chatID  int64
	loc     *time.Location
	meetURL string

	report *commitsreport.Builder
	ownerID int64

	mu   sync.Mutex
	sent map[string]struct{}
}

func New(b *bot.Bot, cfg config.Config, loc *time.Location) *Runner {
	r := &Runner{
		bot:     b,
		chatID:  cfg.NotifyChatID,
		loc:     loc,
		meetURL: cfg.MeetLink,
		ownerID: cfg.OwnerTelegramID,
		sent:    make(map[string]struct{}),
	}

	if cfg.CommitsReportEnabled() {
		mappings := make([]commitsreport.DevMapping, 0, len(cfg.DeveloperList()))
		for _, tg := range cfg.DeveloperList() {
			mappings = append(mappings, commitsreport.DevMapping{
				Telegram: tg,
				GitHub:   cfg.GitHubLogin(tg),
			})
		}
		r.report = &commitsreport.Builder{
			GH:       github.NewClient(cfg.GitHubToken, cfg.GitHubOrg),
			Mappings: commitsreport.SortMappings(mappings),
		}
	}

	return r
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

	if now.Hour() == commitsReportHour && now.Minute() == commitsReportMinute {
		r.sendCommitsReport(ctx, now)
	}

	if (wd == time.Monday || wd == time.Wednesday || wd == time.Friday) &&
		now.Hour() == meetHour && now.Minute() == meetMinute {
		r.sendOnce(ctx, now, "meet", meetMessage(r.meetURL))
	}
}

func (r *Runner) sendOnce(ctx context.Context, now time.Time, kind, text string) {
	key := now.Format("2006-01-02") + ":" + kind
	if !r.markSent(key, now) {
		return
	}

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
		r.unmarkSent(key)
		return
	}
	slog.Info("notification sent", "kind", kind, "photo", photoURL)
}

func (r *Runner) sendCommitsReport(ctx context.Context, now time.Time) {
	if r.report == nil || r.ownerID == 0 {
		return
	}

	key := now.Format("2006-01-02") + ":commits-report"
	if !r.markSent(key, now) {
		return
	}

	text, err := r.report.Daily(ctx, now, r.loc)
	if err != nil {
		slog.Error("commits report", "err", err)
		r.unmarkSent(key)
		return
	}

	_, err = r.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: r.ownerID,
		Text:   text,
	})
	if err != nil {
		slog.Error("send commits report", "owner_id", r.ownerID, "err", err)
		r.unmarkSent(key)
		return
	}
	slog.Info("commits report sent", "owner_id", r.ownerID)
}

func (r *Runner) markSent(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sent[key]; ok {
		return false
	}
	r.sent[key] = struct{}{}
	cutoff := now.Add(-48 * time.Hour).Format("2006-01-02")
	for k := range r.sent {
		if k < cutoff {
			delete(r.sent, k)
		}
	}
	return true
}

func (r *Runner) unmarkSent(key string) {
	r.mu.Lock()
	delete(r.sent, key)
	r.mu.Unlock()
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
