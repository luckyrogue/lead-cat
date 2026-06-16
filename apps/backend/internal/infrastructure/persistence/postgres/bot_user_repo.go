package postgres

import (
	"context"
)

const botUserCols = `id, telegram_id, full_name, email, role, reminder_minutes, timezone, language`

func (s *Store) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE telegram_id = $1`, telegramID).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes, &u.Timezone, &u.Language)
	return u, err
}

func (s *Store) GetBotUserByEmail(ctx context.Context, email string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE email = $1`, email).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes, &u.Timezone, &u.Language)
	return u, err
}

func (s *Store) CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bot_users (telegram_id, full_name, email, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+botUserCols,
		telegramID, fullName, email, role).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role, &u.ReminderMinutes, &u.Timezone, &u.Language)
	return u, err
}

func (s *Store) SetBotUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bot_users SET timezone = $2, language = $3 WHERE telegram_id = $1`, telegramID, timezone, language)
	return err
}

func (s *Store) SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bot_users SET reminder_minutes = $2 WHERE telegram_id = $1`, telegramID, csv)
	return err
}
