package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	BotToken           string
	NotifyChatID       int64
	MeetLink           string
	Timezone           string
	OwnerUsername      string
	OwnerTelegramID    int64
	DeveloperUsernames map[string]struct{}
	TelegramToGitHub   map[string]string
	GitHubToken        string
	GitHubOrg          string
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

	if v := strings.TrimSpace(os.Getenv("BOT_OWNER_USER_ID")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return cfg, fmt.Errorf("BOT_OWNER_USER_ID must be a positive telegram user id")
		}
		cfg.OwnerTelegramID = id
	}

	cfg.GitHubToken = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	cfg.GitHubOrg = envOr("GITHUB_ORG", "Jaryq-Lab")
	cfg.TelegramToGitHub = parseTelegramGitHubMap(os.Getenv("GITHUB_LOGINS"))

	return cfg, nil
}

func parseTelegramGitHubMap(raw string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tg, gh, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		tg = normalizeUsername(tg)
		gh = strings.TrimSpace(gh)
		if tg != "" && gh != "" {
			out[tg] = gh
		}
	}
	return out
}

func (c Config) DeveloperList() []string {
	out := make([]string, 0, len(c.DeveloperUsernames))
	for u := range c.DeveloperUsernames {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}

func (c Config) GitHubLogin(telegramUsername string) string {
	tg := normalizeUsername(telegramUsername)
	if gh, ok := c.TelegramToGitHub[tg]; ok {
		return gh
	}
	return tg
}

func (c Config) CommitsReportEnabled() bool {
	return c.GitHubToken != "" && c.GitHubOrg != "" && c.OwnerTelegramID > 0
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
