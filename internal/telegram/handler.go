package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/Jaryq-Lab/notify-bot/internal/commitsreport"
	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/github"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
	cfg         config.Config
	loc         *time.Location
	report      *commitsreport.Builder
	botID       int64
	botUsername string
}

func NewHandler(cfg config.Config, loc *time.Location, botID int64, botUsername string) *Handler {
	if loc == nil {
		loc = time.UTC
	}
	h := &Handler{
		cfg:         cfg,
		loc:         loc,
		botID:       botID,
		botUsername: normalizeUsername(botUsername),
	}
	if cfg.CommitsReportEnabled() {
		mappings := make([]commitsreport.DevMapping, 0, len(cfg.DeveloperList()))
		for _, tg := range cfg.DeveloperList() {
			mappings = append(mappings, commitsreport.DevMapping{
				Telegram: tg,
				GitHub:   cfg.GitHubLogin(tg),
			})
		}
		h.report = &commitsreport.Builder{
			GH:       github.NewClient(cfg.GitHubToken, cfg.GitHubOrg),
			Mappings: commitsreport.SortMappings(mappings),
		}
	}
	return h
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
					h.cmdChatID(ctx, b, update, isPrivate)
					return
				case "/report":
					if isPrivate && h.cfg.IsOwner(from.Username) {
						h.cmdReport(ctx, b, update.Message)
					}
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
	h.send(ctx, b, msg, devBusyText())
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

func normalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}

func (h *Handler) cmdChatID(ctx context.Context, b *bot.Bot, update *models.Update, isPrivate bool) {
	chat := update.Message.Chat
	text := fmt.Sprintf(
		"chat_id: %d\ntype: %s\ntitle: %s\nthread_id: %d",
		chat.ID, chat.Type, chatTitle(&chat), update.Message.MessageThreadID,
	)
	if isPrivate && update.Message.From != nil {
		text += fmt.Sprintf("\nuser_id: %d\n\nДля BOT_OWNER_USER_ID — этот user_id.", update.Message.From.ID)
	}
	text += "\n\nДля NOTIFY_CHAT_ID в .env — chat_id группы (отрицательный).\n\n— злой лид-кот 👁"
	h.sendPlain(ctx, b, update.Message, text)
}

func (h *Handler) cmdReport(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if h.report == nil {
		h.sendPlain(ctx, b, msg, "Отчёт выключен: нужны GITHUB_TOKEN, GITHUB_ORG и BOT_OWNER_USER_ID.")
		return
	}
	text, err := h.report.Daily(ctx, time.Now(), h.loc)
	if err != nil {
		slog.Error("report command", "err", err)
		h.sendPlain(ctx, b, msg, "Не собрал отчёт: "+err.Error())
		return
	}
	h.sendPlain(ctx, b, msg, text)
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
	return `Не отвлекай — *слежу и рычу в группу* по расписанию.

Глаз на тебе. 👁`
}

func nonDevStaticText() string {
	return `🚫 *Ты не из стаи* — за тобой не слежу. В личку не болтаю.

Мур.`
}
