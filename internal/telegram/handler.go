package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/neuro"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const devGroupGeminiDailyLimit = 2

type Handler struct {
	cfg         config.Config
	neuro       *neuro.Client
	loc         *time.Location
	botID       int64
	botUsername string

	devGeminiMu     sync.Mutex
	devGeminiPerDay map[string]int // "userID:YYYY-MM-DD" → count
}

func NewHandler(cfg config.Config, loc *time.Location, botID int64, botUsername string) *Handler {
	var nc *neuro.Client
	if cfg.GeminiAPIKey != "" {
		nc = neuro.NewClient(cfg.GeminiAPIKey)
	}
	if loc == nil {
		loc = time.UTC
	}
	return &Handler{
		cfg:             cfg,
		neuro:           nc,
		loc:             loc,
		botID:           botID,
		botUsername:     normalizeUsername(botUsername),
		devGeminiPerDay: make(map[string]int),
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
					h.sendPlain(ctx, b, update.Message, "мяу. злой лид-кот на посту — слежу 🐈‍⬛")
					return
				case "/chatid":
					chat := update.Message.Chat
					h.sendPlain(ctx, b, update.Message, fmt.Sprintf(
						"chat_id: %d\ntype: %s\ntitle: %s\nthread_id: %d\n\nДля NOTIFY_CHAT_ID в .env — этот id, если type supergroup/group.\n\n— злой лид-кот 👁",
						chat.ID, chat.Type, chatTitle(&chat), update.Message.MessageThreadID,
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

	msg := update.Message
	if !h.cfg.IsDeveloper(from.Username) {
		h.send(ctx, b, msg, nonDevStaticText())
		return
	}

	if isPrivate {
		if h.cfg.IsOwner(from.Username) {
			h.replyWithGemini(ctx, b, msg, text)
		} else {
			h.send(ctx, b, msg, devBusyText())
		}
		return
	}

	// Группа, обращение к боту
	if h.cfg.IsOwner(from.Username) {
		h.replyWithGemini(ctx, b, msg, text)
		return
	}

	if h.devGroupGeminiCount(from.ID) >= devGroupGeminiDailyLimit {
		h.send(ctx, b, msg, devGeminiLimitText())
		return
	}
	if h.replyWithGemini(ctx, b, msg, text) {
		h.recordDevGroupGemini(from.ID)
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
			if h.botUsername != "" && strings.EqualFold(strings.TrimPrefix(entityTextUTF16(body, e), "@"), h.botUsername) {
				return true
			}
		}
	}

	if h.botUsername != "" {
		return strings.Contains(strings.ToLower(body), "@"+h.botUsername)
	}
	return false
}

func entityTextUTF16(text string, e models.MessageEntity) string {
	u16 := utf16.Encode([]rune(text))
	end := e.Offset + e.Length
	if e.Offset < 0 || end > len(u16) {
		return ""
	}
	return string(utf16.Decode(u16[e.Offset:end]))
}

func stripBotMention(text, botUsername string) string {
	if botUsername == "" {
		return strings.TrimSpace(text)
	}
	lower := strings.ToLower(text)
	needle := "@" + botUsername
	for {
		i := strings.Index(lower, needle)
		if i < 0 {
			break
		}
		text = text[:i] + text[i+len(needle):]
		lower = strings.ToLower(text)
	}
	return strings.TrimSpace(text)
}

func normalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}

func (h *Handler) cmdLeave(ctx context.Context, b *bot.Bot, update *models.Update, isPrivate bool, text string) {
	msg := update.Message
	targetChatID, err := h.leaveTargetChatID(update, isPrivate, text)
	if err != nil {
		h.sendPlain(ctx, b, msg, err.Error())
		return
	}

	wasNotify := targetChatID == h.cfg.NotifyChatID
	if !isPrivate {
		h.sendPlain(ctx, b, msg, "Ухожу. Больше не слежу за этим чатом. 🐈‍⬛")
	}

	ok, err := b.LeaveChat(ctx, &bot.LeaveChatParams{ChatID: targetChatID})
	if err != nil {
		slog.Error("leave chat", "chat_id", targetChatID, "err", err)
		if isPrivate {
			h.sendPlain(ctx, b, msg, leaveChatErrorText(err))
		}
		return
	}
	if !ok {
		slog.Warn("leave chat returned false", "chat_id", targetChatID)
	}

	if !isPrivate {
		return
	}

	reply := fmt.Sprintf("Вышел из чата %d. Мур.", targetChatID)
	if wasNotify {
		reply += "\n\nЭто был NOTIFY_CHAT_ID — оповещения сюда больше не придут."
	}
	h.sendPlain(ctx, b, msg, reply)
}

func (h *Handler) leaveTargetChatID(update *models.Update, isPrivate bool, text string) (int64, error) {
	if !isPrivate {
		chat := update.Message.Chat
		switch chat.Type {
		case models.ChatTypeGroup, models.ChatTypeSupergroup:
			return chat.ID, nil
		default:
			return 0, fmt.Errorf("выйти можно только из группы — вызови /leave в группе")
		}
	}

	args := commandArgs(text)
	targetID := h.cfg.NotifyChatID
	if args != "" {
		id, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("использование: /leave -1001234567890\nid группы — через /chatid в группе")
		}
		targetID = id
	}

	if !isGroupChatID(targetID) {
		return 0, fmt.Errorf(
			"это не id группы (нужно отрицательное число).\n\n" +
				"в группе: /chatid\n" +
				"в личке: /leave -100…",
		)
	}
	return targetID, nil
}

func isGroupChatID(id int64) bool {
	return id < 0
}

func leaveChatErrorText(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "private chats") {
		return "Не вышел: указан личный чат, а не группа.\n\nВ группе: /leave\nВ личке: /leave -100… (id из /chatid)"
	}
	return "Не вышел: " + msg
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

func (h *Handler) replyWithGemini(ctx context.Context, b *bot.Bot, msg *models.Message, text string) bool {
	question := stripBotMention(text, h.botUsername)
	if question == "" {
		question = text
	}

	if h.neuro == nil || question == "" {
		h.sendPlain(ctx, b, msg, "Мур. Сейчас рыкнуть не могу.")
		return false
	}

	answer, err := h.neuro.Ask(ctx, question)
	if err != nil {
		slog.Error("gemini", "err", err)
		h.sendPlain(ctx, b, msg, "Сейчас рыкнуть не могу — мур позже.")
		return false
	}

	h.sendPlain(ctx, b, msg, answer)
	return true
}

func (h *Handler) devGeminiDayKey(userID int64) string {
	return fmt.Sprintf("%d:%s", userID, time.Now().In(h.loc).Format("2006-01-02"))
}

func (h *Handler) devGroupGeminiCount(userID int64) int {
	h.devGeminiMu.Lock()
	defer h.devGeminiMu.Unlock()
	return h.devGeminiPerDay[h.devGeminiDayKey(userID)]
}

func (h *Handler) recordDevGroupGemini(userID int64) {
	h.devGeminiMu.Lock()
	defer h.devGeminiMu.Unlock()
	key := h.devGeminiDayKey(userID)
	h.devGeminiPerDay[key]++
	h.pruneDevGeminiCountsLocked()
}

func (h *Handler) pruneDevGeminiCountsLocked() {
	today := time.Now().In(h.loc).Format("2006-01-02")
	for k := range h.devGeminiPerDay {
		if !strings.HasSuffix(k, ":"+today) {
			delete(h.devGeminiPerDay, k)
		}
	}
}

func devGeminiLimitText() string {
	return fmt.Sprintf(
		"Лимит на сегодня — *%d ответа*. Завтра снова.\n\nМур.",
		devGroupGeminiDailyLimit,
	)
}

func (h *Handler) send(ctx context.Context, b *bot.Bot, msg *models.Message, text string) {
	h.sendMessage(ctx, b, msg, text, true)
}

func (h *Handler) sendPlain(ctx context.Context, b *bot.Bot, msg *models.Message, text string) {
	h.sendMessage(ctx, b, msg, text, false)
}

func (h *Handler) sendMessage(ctx context.Context, b *bot.Bot, msg *models.Message, text string, markdown bool) {
	params := &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   text,
	}
	if msg.MessageThreadID != 0 {
		params.MessageThreadID = msg.MessageThreadID
	}
	if markdown {
		params.ParseMode = models.ParseModeMarkdown
	}

	_, err := b.SendMessage(ctx, params)
	if err != nil && markdown {
		params.ParseMode = ""
		_, err = b.SendMessage(ctx, params)
	}
	if err != nil {
		slog.Error("send reply",
			"chat_id", msg.Chat.ID,
			"thread_id", msg.MessageThreadID,
			"err", err,
		)
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
	return `Не отвлекай — **слежу и рычу в группу** по расписанию.

Глаз на тебе. 👁`
}

func nonDevStaticText() string {
	return `🚫 **Ты не из стаи** — за тобой не слежу. В личку не болтаю.

Мур.`
}
