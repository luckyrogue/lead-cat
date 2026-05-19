package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken           string
	NotifyChatID       int64
	MeetLink           string
	Timezone           string
	OwnerUsername      string
	DeveloperUsernames map[string]struct{}
}

func Load() (Config, error) {
	cfg := Config{
		BotToken:     os.Getenv("BOT_TOKEN"),
		MeetLink:     strings.TrimSpace(os.Getenv("MEET_LINK")),
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

	devs, err := parseDeveloperUsernames(os.Getenv("DEVELOPER_USERNAMES"))
	if err != nil {
		return cfg, err
	}
	if len(devs) == 0 {
		return cfg, fmt.Errorf("DEVELOPER_USERNAMES is required (username через запятую, без @)")
	}
	cfg.DeveloperUsernames = devs

	owner := normalizeUsername(os.Getenv("BOT_OWNER_USERNAME"))
	if owner == "" {
		return cfg, fmt.Errorf("BOT_OWNER_USERNAME is required (твой @username без @, для команд в личке)")
	}
	cfg.OwnerUsername = owner

	return cfg, nil
}

func parseDeveloperUsernames(raw string) (map[string]struct{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		name := normalizeUsername(part)
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out, nil
}

func normalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c Config) IsDeveloper(username string) bool {
	name := normalizeUsername(username)
	if name == "" {
		return false
	}
	_, ok := c.DeveloperUsernames[name]
	return ok
}

func (c Config) IsOwner(username string) bool {
	return normalizeUsername(username) == c.OwnerUsername
}
