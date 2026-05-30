package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) UpsertUser(ctx context.Context, authSub, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO platform_users (auth_sub, email)
		VALUES ($1, $2)
		ON CONFLICT (auth_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, auth_sub, email, telegram_id`,
		authSub, email).Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID)
	return u, err
}

func (s *Store) GetUserBySub(ctx context.Context, sub string) (User, error) {
	var u User
	var phone *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, auth_sub, email, telegram_id, phone FROM platform_users WHERE auth_sub = $1`, sub).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &phone)
	if phone != nil {
		u.Phone = *phone
	}
	return u, err
}

func (s *Store) LinkTelegram(ctx context.Context, userID uuid.UUID, telegramID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE platform_users SET telegram_id = $2 WHERE id = $1`, userID, telegramID)
	return err
}

func (s *Store) OwnerTelegramID(ctx context.Context, workspaceID uuid.UUID) (int64, error) {
	var tid *int64
	err := s.pool.QueryRow(ctx, `
		SELECT u.telegram_id FROM workspaces w
		JOIN platform_users u ON u.id = w.owner_user_id
		WHERE w.id = $1`, workspaceID).Scan(&tid)
	if err != nil || tid == nil {
		return 0, fmt.Errorf("owner telegram not linked")
	}
	return *tid, nil
}

// GetUserTelegramID returns the platform user's linked Telegram id. ok is false
// when the user exists but has not linked Telegram.
func (s *Store) GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error) {
	var tg *int64
	err := s.pool.QueryRow(ctx, `SELECT telegram_id FROM platform_users WHERE id = $1`, userID).Scan(&tg)
	if err != nil {
		return 0, false, err
	}
	if tg == nil {
		return 0, false, nil
	}
	return *tg, true, nil
}
