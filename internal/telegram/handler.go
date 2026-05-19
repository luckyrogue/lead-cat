package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/Jaryq-Lab/notify-bot/internal/config"
	"github.com/Jaryq-Lab/notify-bot/internal/neuro"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
	cfg    config.Config
	neuro  *neuro.Client
	seen   sync.Map // userID -> struct{} — уже получил приветствие
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

	if isTestCommand(text) {
		if h.cfg.IsDeveloper(from.Username) {
			h.sendPlain(ctx, b, update.Message.Chat.ID, "мяу")
		}
		return
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
		h.send(ctx, b, chatID, nonDevStaticText()+"\n\n_Нейрока сейчас недоступна._")
		return
	}

	h.send(ctx, b, chatID, nonDevNeuroText(answer))
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

func isTestCommand(text string) bool {
	cmd, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	cmd = strings.SplitN(cmd, "@", 2)[0]
	return cmd == "/test"
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
	return `👋 *Привет, разработчик!*

Я notify-бот. Работаю по расписанию:
• *Пн–Пт 18:30* — сдать день техлиду (коммиты на бранчи)
• *Пн / Ср / Пт 10:15* — ссылка на мит
• К каждому оповещению — случайный злой кот 🐈‍⬛

Дальше болтать не буду — просто шлю в группу.

Проверка: /test → мяу`
}

func devBusyText() string {
	return `Я не для долгих разговоров — *делаю оповещения* по расписанию.

Всё ок, работаю 🤖`
}

func nonDevStaticText() string {
	return `🚫 *Ты не разработчик* — notify-бот с тобой не общается.

Вопросы и болтовня → *Нейрока* (Gemini). Попроси админа включить ` + "`GEMINI_API_KEY`" + `.`
}

func nonDevNeuroText(answer string) string {
	return `🚫 *Notify-бот* с не-разработчиками не разговаривает.

🤖 *Нейрока:*
` + answer
}
