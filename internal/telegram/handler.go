package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/neuro"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
	cfg   config.Config
	neuro *neuro.Client
	seen  sync.Map
}

func NewHandler(cfg config.Config) *Handler {
	var nc *neuro.Client
	if cfg.GeminiAPIKey != "" {
		nc = neuro.NewClient(cfg.GeminiAPIKey)
	}
	return &Handler{cfg: cfg, neuro: nc}
}

func (h *Handler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	from := update.Message.From
	text := replyText(update)

	if cmd, ok := parseCommand(text); ok && h.cfg.IsDeveloper(from.Username) {
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

	// В группе оповещений молчим — только шлём по расписанию.
	if update.Message.Chat.Type != models.ChatTypePrivate {
		return
	}

	switch {
	case h.cfg.IsDeveloper(from.Username):
		h.replyDeveloper(ctx, b, update.Message.Chat.ID, from.ID, text)
	default:
		h.replyNonDeveloper(ctx, b, update.Message.Chat.ID, text)
	}
}

func (h *Handler) replyDeveloper(ctx context.Context, b *bot.Bot, chatID, userID int64, text string) {
	if _, seen := h.seen.LoadOrStore(userID, struct{}{}); !seen {
		h.send(ctx, b, chatID, devWelcomeText())
		return
	}
	_ = text
	h.send(ctx, b, chatID, devBusyText())
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

func devWelcomeText() string {
	return `🐈‍⬛ *Злой лид-кот следит за тобой.*

По расписанию:
• *Пн–Пт 18:30* — сдай день: коммиты на бранчи + отчёт техлиду
• *Пн / Ср / Пт 10:15* — зову на мит
• К каждому напоминанию — моё злое котофото

В личку болтать не буду — слежу и рычу в группу.

/test — я на посту (мяу)
/chatid — id чата для NOTIFY_CHAT_ID`
}

func devBusyText() string {
	return `Не отвлекай — *слежу и рычу в группу* по расписанию.

Глаз на тебе. 👁`
}

func nonDevStaticText() string {
	return `🚫 *Ты не из стаи* — за тобой не слежу. В личку не болтаю.

Мур.`
}
