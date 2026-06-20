package telegram

import (
	"strings"

	"github.com/go-telegram/bot/models"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func PublicCommands(lang string) []models.BotCommand {
	return []models.BotCommand{
		{Command: "menu", Description: boti18n.T(lang, "cmd.menu")},
		{Command: "new", Description: boti18n.T(lang, "cmd.new")},
		{Command: "schedule", Description: boti18n.T(lang, "cmd.schedule")},
		{Command: "checker", Description: boti18n.T(lang, "cmd.checker")},
		{Command: "settings", Description: boti18n.T(lang, "cmd.settings")},
		{Command: "edit", Description: boti18n.T(lang, "cmd.edit")},
		{Command: "help", Description: boti18n.T(lang, "cmd.help")},
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

func helpText(lang string) string {
	return boti18n.T(lang, "cmd.help.text")
}
