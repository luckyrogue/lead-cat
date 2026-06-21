package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
	"github.com/luckyrogue/lead-cat/internal/platform/botreg"
	"github.com/luckyrogue/lead-cat/internal/platform/botsettings"
	"github.com/luckyrogue/lead-cat/internal/platform/checker"
	"github.com/luckyrogue/lead-cat/internal/platform/meetingedit"
	"github.com/luckyrogue/lead-cat/internal/platform/scheduler_agent"
	"github.com/luckyrogue/lead-cat/internal/platform/scheduleview"
)

type MultiHandler struct {
	store     *postgres.Store
	registrar *botreg.Service
	settings  *botsettings.Service
	editor    *meetingedit.Service
	schedule  *scheduleview.Service
	checker   *checker.Service
	agent     *scheduler_agent.Service
	log       *zap.Logger
	webappURL string
	admins    map[int64]bool
}

func NewMultiHandler(store *postgres.Store, b *bot.Bot, rdb *redis.Client, adminIDs []int64, webappURL string, services *application.Services, planner application.Planner, log *zap.Logger) *MultiHandler {
	registrar := botreg.New(store, botreg.NewRedisSessions(rdb), adminIDs)
	settings := botsettings.New(store)
	editor := meetingedit.New(services, meetingedit.NewRedisSessions(rdb))
	schedule := scheduleview.New(services, scheduleview.NewRedisSessions(rdb))
	chk := checker.New(services, checker.NewRedisSessions(rdb))
	booker := &agentBooker{store: store, services: services}
	agent := scheduler_agent.NewWithBooker(planner, services, booker, scheduler_agent.NewRedisSessions(rdb))
	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}
	return &MultiHandler{
		store:     store,
		registrar: registrar,
		settings:  settings,
		editor:    editor,
		schedule:  schedule,
		checker:   chk,
		agent:     agent,
		log:       log,
		webappURL: webappURL,
		admins:    admins,
	}
}

func (h *MultiHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, b, update.CallbackQuery)
		return
	}
	if update.Message == nil || update.Message.From == nil {
		return
	}
	h.trackChatMember(ctx, update.Message)
	from := update.Message.From
	text := strings.TrimSpace(update.Message.Text)
	isPrivate := update.Message.Chat.Type == models.ChatTypePrivate
	chatID := update.Message.Chat.ID

	cmd, ok := parseCommand(text)
	if !ok {
		if isPrivate {
			if reply, handled := h.registrar.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {
				h.reply(ctx, b, update.Message, reply)
				return
			}
			if reply, handled := h.editor.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {
				h.sendEditorReply(ctx, b, chatID, 0, reply)
				return
			}
			if reply, handled := h.schedule.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {
				h.sendSchedReply(ctx, b, chatID, 0, reply)
				return
			}
			if reply, handled := h.checker.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {
				h.sendCheckerReply(ctx, b, chatID, 0, reply)
				return
			}

			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
				reply, _ := h.agent.OnText(ctx, from.ID, text)
				h.sendAgentReply(ctx, b, chatID, 0, reply)
			}
		}
		return
	}
	switch cmd {
	case "/start":
		if isPrivate {
			h.reply(ctx, b, update.Message, h.registrar.Start(ctx, from.ID, h.resolveLang(ctx, from)))
		}
	case "/chatid":
		_ = h.store.UpsertPendingChat(ctx, from.ID, chatID, update.Message.Chat.Title)
		h.reply(ctx, b, update.Message, fmt.Sprintf("Логово id: %d — скопируй в админку 🐾", chatID))
	case "/settings":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			text, kb, serr := h.settings.Settings(ctx, from.ID, h.resolveLang(ctx, from))
			if serr == nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      chatID,
					Text:        text,
					ReplyMarkup: toInlineMarkup(kb),
				})
			}
		}
	case "/edit":
		if isPrivate {
			h.sendEditorReply(ctx, b, chatID, 0, h.editor.Start(ctx, from.ID, h.resolveLang(ctx, from)))
		}
	case "/schedule":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			h.sendSchedReply(ctx, b, chatID, 0, h.schedule.Start(ctx, from.ID, h.resolveLang(ctx, from)))
		}
	case "/checker":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			h.sendCheckerReply(ctx, b, chatID, 0, h.checker.Start(ctx, from.ID, h.resolveLang(ctx, from)))
		}
	case "/menu":
		if isPrivate {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        "Открой Lead Cat 🐾",
				ReplyMarkup: webAppMarkup("Открыть приложение", h.webappURL),
			})
		}
	case "/new":
		if isPrivate {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        "Запланируй встречу 🐾",
				ReplyMarkup: webAppMarkup("Новая встреча", joinURL(h.webappURL, "meetings/create")),
			})
		}
	case "/help":
		if isPrivate {
			h.reply(ctx, b, update.Message, helpText(h.resolveLang(ctx, from)))
		}
	case "/admin":
		if isPrivate {
			if h.admins[from.ID] {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      chatID,
					Text:        "Ты администратор 🐾 Настройки — в приложении.",
					ReplyMarkup: webAppMarkup("Открыть приложение", h.webappURL),
				})
			} else {
				h.reply(ctx, b, update.Message, "Ты не администратор 🐾")
			}
		}
	}
}

func (h *MultiHandler) handleCallback(ctx context.Context, b *bot.Bot, cq *models.CallbackQuery) {
	if strings.HasPrefix(cq.Data, "rem:") {
		if v, err := strconv.Atoi(strings.TrimPrefix(cq.Data, "rem:")); err == nil {
			text, kb, serr := h.settings.Toggle(ctx, cq.From.ID, v, h.resolveLang(ctx, &cq.From))
			if serr == nil && cq.Message.Message != nil {
				_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:      cq.Message.Message.Chat.ID,
					MessageID:   cq.Message.Message.ID,
					Text:        text,
					ReplyMarkup: toInlineMarkup(kb),
				})
			}
		}
	}
	if strings.HasPrefix(cq.Data, "medit:") {
		if reply, handled := h.editor.OnCallback(ctx, cq.From.ID, cq.Data, h.resolveLang(ctx, &cq.From)); handled && cq.Message.Message != nil {
			h.sendEditorReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
	if strings.HasPrefix(cq.Data, "sched:") {
		if reply, handled := h.schedule.OnCallback(ctx, cq.From.ID, cq.Data, h.resolveLang(ctx, &cq.From)); handled && cq.Message.Message != nil {
			h.sendSchedReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
	if strings.HasPrefix(cq.Data, "chk:") {
		if reply, handled := h.checker.OnCallback(ctx, cq.From.ID, cq.Data, h.resolveLang(ctx, &cq.From)); handled && cq.Message.Message != nil {
			h.sendCheckerReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
	if strings.HasPrefix(cq.Data, "agent:") {
		if reply, handled := h.agent.OnCallback(ctx, cq.From.ID, cq.Data); handled && cq.Message.Message != nil {
			h.sendAgentReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID})
}

func (h *MultiHandler) sendEditorReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply meetingedit.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toMeditMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toMeditMarkup(rows [][]meetingedit.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}

func (h *MultiHandler) sendSchedReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply scheduleview.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toSchedMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toSchedMarkup(rows [][]scheduleview.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}

func (h *MultiHandler) sendCheckerReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply checker.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toCheckerMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func (h *MultiHandler) sendAgentReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply scheduler_agent.Reply) {
	if reply.Text == "" {
		return
	}
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toAgentMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toAgentMarkup(rows [][]scheduler_agent.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}

func toCheckerMarkup(rows [][]checker.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}

func toInlineMarkup(rows [][]botsettings.Button) models.InlineKeyboardMarkup {
	out := make([][]models.InlineKeyboardButton, 0, len(rows))
	for _, r := range rows {
		row := make([]models.InlineKeyboardButton, 0, len(r))
		for _, btn := range r {
			row = append(row, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		out = append(out, row)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: out}
}

func (h *MultiHandler) reply(ctx context.Context, b *bot.Bot, msg *models.Message, text string) {
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: text})
}

// resolveLang returns the acting user's language: their stored bot_users.language
// if registered, else their Telegram client language_code, else ru.
func (h *MultiHandler) resolveLang(ctx context.Context, from *models.User) string {
	var stored string
	if u, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
		stored = u.Language
	}
	return boti18n.Resolve(stored, from.LanguageCode)
}

func (h *MultiHandler) trackChatMember(ctx context.Context, msg *models.Message) {
	if msg.Chat.Type == models.ChatTypePrivate || msg.From == nil {
		return
	}
	org, err := h.store.GetOrganizationByChatID(ctx, msg.Chat.ID)
	if err != nil || msg.From.Username == "" {
		return
	}
	_ = h.store.UpsertMemberFromChat(ctx, org.ID, msg.From.Username, "developer")
}

func parseCommand(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", false
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", false
	}
	return strings.SplitN(parts[0], "@", 2)[0], true
}
