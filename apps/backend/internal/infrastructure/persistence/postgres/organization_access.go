package postgres

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) LinkMemberUserIDsByTelegram(ctx context.Context, userID uuid.UUID, telegramUsername string) error {
	telegramUsername = normalizeUsername(telegramUsername)
	if telegramUsername == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE organization_members SET user_id = $1
		WHERE lower(telegram_username) = $2 AND (user_id IS NULL OR user_id = $1)`,
		userID, telegramUsername)
	return err
}
