package telegram

import (
	"strings"

	"github.com/go-telegram/bot/models"
)

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

func webAppMarkup(text, url string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: text, WebApp: &models.WebAppInfo{URL: url}},
	}}}
}

func joinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	if path == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

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
