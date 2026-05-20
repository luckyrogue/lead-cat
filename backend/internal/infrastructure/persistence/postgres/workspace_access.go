package postgres

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) UserCanAccessWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspaces w
			WHERE w.id = $2 AND (
				w.owner_user_id = $1
				OR EXISTS (
					SELECT 1 FROM workspace_members m
					WHERE m.workspace_id = w.id AND m.user_id = $1
				)
			)
		)`, userID, workspaceID).Scan(&ok)
	return ok, err
}

func (s *Store) LinkMemberUserIDsByTelegram(ctx context.Context, userID uuid.UUID, telegramUsername string) error {
	telegramUsername = normalizeUsername(telegramUsername)
	if telegramUsername == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE workspace_members SET user_id = $1
		WHERE lower(telegram_username) = $2 AND (user_id IS NULL OR user_id = $1)`,
		userID, telegramUsername)
	return err
}
