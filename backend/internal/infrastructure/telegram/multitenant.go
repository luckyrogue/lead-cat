package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/cats"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/scenario_executor"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

type MultiHandler struct {
	store    *postgres.Store
	executor *scenario_executor.Executor
	log      *zap.Logger
}

func NewMultiHandler(store *postgres.Store, cipher *crypto.TokenCipher, b *bot.Bot, log *zap.Logger) *MultiHandler {
	return &MultiHandler{
		store:    store,
		executor: scenario_executor.New(store, cipher, b, log),
		log:      log,
	}
}

func (h *MultiHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
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
		return
	}
	switch cmd {
	case "/chatid":
		_ = h.store.UpsertPendingChat(ctx, from.ID, chatID, update.Message.Chat.Title)
		h.reply(ctx, b, update.Message, fmt.Sprintf("Логово id: %d — скопируй в админку 🐾", chatID))
	case "/test":
		ws, err := h.store.GetWorkspaceByChatID(ctx, chatID)
		if err != nil {
			h.reply(ctx, b, update.Message, "Кот не привязан к логову.")
			return
		}
		okDev, _ := h.store.IsMemberDeveloper(ctx, ws.ID, from.Username)
		if !okDev {
			h.reply(ctx, b, update.Message, "Сюда только для своих котиков")
			return
		}
		h.reply(ctx, b, update.Message, "Мяу! Проверка связи 🐾")
		if url := cats.RandomImageURL(ctx); url != "" {
			_, _ = b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID: chatID,
				Photo:  &models.InputFileString{Data: url},
			})
		}
	case "/report":
		ws, err := h.store.GetWorkspaceByChatID(ctx, chatID)
		if err != nil {
			return
		}
		okDev, _ := h.store.IsMemberDeveloper(ctx, ws.ID, from.Username)
		if !okDev && !isPrivate {
			return
		}
		if err := h.executor.SendCommitsReport(ctx, ws); err != nil {
			h.reply(ctx, b, update.Message, "Кот не смог собрать отчёт: "+err.Error())
		}
	}
}

func (h *MultiHandler) reply(ctx context.Context, b *bot.Bot, msg *models.Message, text string) {
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: text})
}

func (h *MultiHandler) trackChatMember(ctx context.Context, msg *models.Message) {
	if msg.Chat.Type == models.ChatTypePrivate || msg.From == nil {
		return
	}
	ws, err := h.store.GetWorkspaceByChatID(ctx, msg.Chat.ID)
	if err != nil || msg.From.Username == "" {
		return
	}
	_ = h.store.UpsertMemberFromChat(ctx, ws.ID, msg.From.Username, "developer")
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
