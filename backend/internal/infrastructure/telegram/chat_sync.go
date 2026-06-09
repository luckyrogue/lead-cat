package telegram

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// SyncChatMembers imports chat administrators into workspace_members.
// Telegram Bot API does not expose full member lists; admins + anyone who posts are tracked.
func SyncChatMembers(ctx context.Context, b *bot.Bot, store *postgres.Store, workspaceID uuid.UUID) (int, error) {
	ws, err := store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	if ws.NotifyChatID == nil {
		return 0, fmt.Errorf("chat not linked")
	}

	admins, err := b.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{
		ChatID: *ws.NotifyChatID,
	})
	if err != nil {
		return 0, err
	}

	n := 0
	for _, m := range admins {
		user, role := chatMemberUser(m)
		if user == nil || user.IsBot || user.Username == "" {
			continue
		}
		if err := store.UpsertMemberFromChat(ctx, workspaceID, user.Username, role); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func chatMemberUser(m models.ChatMember) (*models.User, string) {
	switch m.Type {
	case models.ChatMemberTypeOwner:
		if m.Owner != nil && m.Owner.User != nil {
			return m.Owner.User, "admin"
		}
	case models.ChatMemberTypeAdministrator:
		if m.Administrator != nil {
			return &m.Administrator.User, "admin"
		}
	case models.ChatMemberTypeMember:
		if m.Member != nil && m.Member.User != nil {
			return m.Member.User, "developer"
		}
	case models.ChatMemberTypeRestricted:
		if m.Restricted != nil && m.Restricted.User != nil && m.Restricted.IsMember {
			return m.Restricted.User, "developer"
		}
	}
	return nil, ""
}
