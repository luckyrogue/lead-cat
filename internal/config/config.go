package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken     string
	NotifyChatID int64
	MeetLink     string
	Timezone     string
}

func Load() (Config, error) {
	cfg := Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		MeetLink: strings.TrimSpace(os.Getenv("MEET_LINK")),
		Timezone: envOr("TZ", "Asia/Almaty"),
	}

	if cfg.BotToken == "" {
		return cfg, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.MeetLink == "" {
		return cfg, fmt.Errorf("MEET_LINK is required (ссылка на созвон Пн/Ср/Пт)")
	}

	chatID, err := strconv.ParseInt(os.Getenv("NOTIFY_CHAT_ID"), 10, 64)
	if err != nil || chatID == 0 {
		return cfg, fmt.Errorf("NOTIFY_CHAT_ID must be a valid group chat id (negative number)")
	}
	cfg.NotifyChatID = chatID

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
