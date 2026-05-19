package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/scheduler"
	"github.com/go-telegram/bot"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("timezone", "tz", cfg.Timezone, "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tg, err := bot.New(cfg.BotToken)
	if err != nil {
		slog.Error("telegram bot", "err", err)
		os.Exit(1)
	}

	runner := scheduler.New(tg, cfg.NotifyChatID, loc, cfg.MeetLink)
	slog.Info("notify-bot started",
		"chat_id", cfg.NotifyChatID,
		"tz", cfg.Timezone,
		"commits", "Пн–Пт 18:30",
		"meet", "Пн/Ср/Пт 10:15",
	)

	runner.Start(ctx)
}
