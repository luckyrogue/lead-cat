package telegram

import (
	"strings"

	"github.com/go-telegram/bot/models"
)

// PublicCommands is the bot command list published to Telegram via SetMyCommands.
// Command values must be lowercase and without a leading slash.
func PublicCommands() []models.BotCommand {
	return []models.BotCommand{
		{Command: "menu", Description: "Открыть приложение"},
		{Command: "new", Description: "Запланировать встречу"},
		{Command: "schedule", Description: "Расписание коллеги"},
		{Command: "checker", Description: "Найти свободный слот"},
		{Command: "settings", Description: "Напоминания"},
		{Command: "edit", Description: "Редактировать встречи"},
		{Command: "help", Description: "Помощь"},
	}
}

// webAppMarkup builds a one-button inline keyboard that opens the Mini App at url.
func webAppMarkup(text, url string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: text, WebApp: &models.WebAppInfo{URL: url}},
	}}}
}

// joinURL joins a base URL and a path, tolerating slashes on either side.
func joinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	if path == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

// helpText is the /help reply describing the available commands.
func helpText() string {
	return strings.Join([]string{
		"Lead Cat — помощник по встречам 🐾",
		"",
		"/menu — открыть приложение",
		"/new — запланировать встречу",
		"/schedule — расписание коллеги",
		"/checker — найти свободный слот",
		"/settings — настроить напоминания",
		"/edit — редактировать свои встречи",
		"/help — это сообщение",
	}, "\n")
}
