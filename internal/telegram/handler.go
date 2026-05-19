package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/neuro"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
	cfg         config.Config
	neuro       *neuro.Client
	botID       int64
	botUsername string
}

func NewHandler(cfg config.Config, botID int64, botUsername string) *Handler {
	var nc *neuro.Client
	if cfg.GeminiAPIKey != "" {
		nc = neuro.NewClient(cfg.GeminiAPIKey)
	}
	return &Handler{
		cfg:         cfg,
		neuro:       nc,
		botID:       botID,
		botUsername: normalizeUsername(botUsername),
	}
}

func (h *Handler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	from := update.Message.From
	text := replyText(update)
	isPrivate := update.Message.Chat.Type == models.ChatTypePrivate

	if cmd, ok := parseCommand(text); ok {
		switch cmd {
		case "/leave":
			if h.cfg.IsOwner(from.Username) {
				h.cmdLeave(ctx, b, update, isPrivate, text)
			}
			return
		default:
			if h.mayRunCommand(from.Username, isPrivate) {
				switch cmd {
				case "/test":
					h.sendPlain(ctx, b, update.Message.Chat.ID, "мяу. злой лид-кот на посту — слежу 🐈‍⬛")
					return
				case "/chatid":
					chat := update.Message.Chat
					h.sendPlain(ctx, b, chat.ID, fmt.Sprintf(
						"chat_id: %d\ntype: %s\ntitle: %s\n\nДля NOTIFY_CHAT_ID в .env — этот id, если type supergroup/group.\n\n— злой лид-кот 👁",
						chat.ID, chat.Type, chatTitle(&chat),
					))
					return
				}
			}
		}
		if isPrivate {
			return
		}
	}

	if !isPrivate && !h.isAddressedToBot(update.Message) {
		return
	}

	chatID := update.Message.Chat.ID
	switch {
	case h.cfg.IsDeveloper(from.Username):
		_ = text
		h.send(ctx, b, chatID, devBusyText())
	default:
		h.replyNonDeveloper(ctx, b, chatID, text)
	}
}

func (h *Handler) isAddressedToBot(msg *models.Message) bool {
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.ID == h.botID {
		return true
	}

	body := msg.Text
	entities := msg.Entities
	if body == "" {
		body = msg.Caption
		entities = msg.CaptionEntities
	}
	if body == "" {
		return false
	}

	for _, e := range entities {
		switch e.Type {
		case models.MessageEntityTypeTextMention:
			if e.User != nil && e.User.ID == h.botID {
				return true
			}
		case models.MessageEntityTypeMention:
			if h.botUsername != "" && strings.EqualFold(entityText(body, e), "@"+h.botUsername) {
				return true
			}
		}
	}

	if h.botUsername != "" {
		return strings.Contains(strings.ToLower(body), "@"+h.botUsername)
	}
	return false
}

func entityText(text string, e models.MessageEntity) string {
	runes := []rune(text)
	end := e.Offset + e.Length
	if e.Offset < 0 || end > len(runes) {
		return ""
	}
	return string(runes[e.Offset:end])
}

func normalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}

func (h *Handler) cmdLeave(ctx context.Context, b *bot.Bot, update *models.Update, isPrivate bool, text string) {
	replyChatID := update.Message.Chat.ID
	targetChatID, err := h.leaveTargetChatID(update, isPrivate, text)
	if err != nil {
		h.sendPlain(ctx, b, replyChatID, err.Error())
		return
	}

	wasNotify := targetChatID == h.cfg.NotifyChatID
	if !isPrivate {
		h.sendPlain(ctx, b, replyChatID, "Ухожу. Больше не слежу за этим чатом. 🐈‍⬛")
	}

	ok, err := b.LeaveChat(ctx, &bot.LeaveChatParams{ChatID: targetChatID})
	if err != nil {
		slog.Error("leave chat", "chat_id", targetChatID, "err", err)
		if isPrivate {
			h.sendPlain(ctx, b, replyChatID, "Не вышел: "+err.Error())
		}
		return
	}
	if !ok {
		slog.Warn("leave chat returned false", "chat_id", targetChatID)
	}

	if !isPrivate {
		return
	}

	msg := fmt.Sprintf("Вышел из чата %d. Мур.", targetChatID)
	if wasNotify {
		msg += "\n\nЭто был NOTIFY_CHAT_ID — оповещения сюда больше не придут."
	}
	h.sendPlain(ctx, b, replyChatID, msg)
}

func (h *Handler) leaveTargetChatID(update *models.Update, isPrivate bool, text string) (int64, error) {
	if !isPrivate {
		return update.Message.Chat.ID, nil
	}

	args := commandArgs(text)
	if args == "" {
		return h.cfg.NotifyChatID, nil
	}

	id, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("использование: /leave [chat_id]\nбез id — выйду из NOTIFY_CHAT_ID")
	}
	return id, nil
}

func commandArgs(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.IndexByte(text, ' '); i >= 0 {
		return strings.TrimSpace(text[i+1:])
	}
	return ""
}

func (h *Handler) mayRunCommand(username string, isPrivate bool) bool {
	if isPrivate {
		return h.cfg.IsOwner(username)
	}
	return h.cfg.IsDeveloper(username)
}

func (h *Handler) replyNonDeveloper(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if h.neuro == nil || text == "" {
		h.send(ctx, b, chatID, nonDevStaticText())
		return
	}

	answer, err := h.neuro.Ask(ctx, text)
	if err != nil {
		slog.Error("gemini", "err", err)
		h.send(ctx, b, chatID, nonDevStaticText()+"\n\n_Сейчас рыкнуть не могу — мур позже._")
		return
	}

	h.send(ctx, b, chatID, answer)
}

func (h *Handler) send(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	}); err != nil {
		slog.Error("send reply", "chat_id", chatID, "err", err)
	}
}

func (h *Handler) sendPlain(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}); err != nil {
		slog.Error("send reply", "chat_id", chatID, "err", err)
	}
}

func parseCommand(text string) (cmd string, ok bool) {
	part, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	if part == "" || !strings.HasPrefix(part, "/") {
		return "", false
	}
	part = strings.SplitN(part, "@", 2)[0]
	return part, true
}

func chatTitle(chat *models.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.Username != "" {
		return "@" + chat.Username
	}
	return "—"
}

func replyText(update *models.Update) string {
	if update.Message.Text != "" {
		return update.Message.Text
	}
	if update.Message.Caption != "" {
		return update.Message.Caption
	}
	return ""
}

func devBusyText() string {
	return `Не отвлекай — *слежу и рычу в группу* по расписанию.

Глаз на тебе. 👁`
}

func nonDevStaticText() string {
	return `🚫 *Ты не из стаи* — за тобой не слежу. В личку не болтаю.

Мур.`
}
