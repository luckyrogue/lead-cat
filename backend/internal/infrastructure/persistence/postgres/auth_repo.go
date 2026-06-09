package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) UpsertUserIdentity(ctx context.Context, authSub, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO platform_users (auth_sub, email)
		VALUES ($1, $2)
		ON CONFLICT (auth_sub) DO UPDATE SET
			email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE platform_users.email END
		RETURNING id, auth_sub, email, telegram_id`,
		authSub, email).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID)
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, auth_sub, email, telegram_id FROM platform_users WHERE id = $1`, id).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID)
	if err == pgx.ErrNoRows {
		return User{}, err
	}
	return u, err
}
