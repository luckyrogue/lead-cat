package postgres

import (
	"context"
)

const botUserCols = `id, telegram_id, full_name, email, role`

func (s *Store) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE telegram_id = $1`, telegramID).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}

func (s *Store) GetBotUserByEmail(ctx context.Context, email string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE email = $1`, email).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}

func (s *Store) CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bot_users (telegram_id, full_name, email, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+botUserCols,
		telegramID, fullName, email, role).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}
