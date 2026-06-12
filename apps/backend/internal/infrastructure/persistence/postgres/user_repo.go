package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/platform/auth"
)

func (s *Store) GetUserBySub(ctx context.Context, sub string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, auth_sub, email, telegram_id FROM platform_users WHERE auth_sub = $1`, sub).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID)
	return u, err
}

func (s *Store) GetPlatformUserIDByTelegramID(ctx context.Context, telegramID int64) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM platform_users WHERE telegram_id = $1`, telegramID).Scan(&id)
	if IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func (s *Store) GetPlatformUserByID(ctx context.Context, id uuid.UUID) (PlatformUser, bool, error) {
	var u PlatformUser
	err := s.pool.QueryRow(ctx, `
		SELECT id, auth_sub, email, telegram_id, avatar_url, auth_method, created_at
		FROM platform_users WHERE id = $1`, id).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &u.AvatarURL, &u.AuthMethod, &u.CreatedAt)
	if IsNotFound(err) {
		return PlatformUser{}, false, nil
	}
	if err != nil {
		return PlatformUser{}, false, err
	}
	return u, true, nil
}

func (s *Store) LinkTelegram(ctx context.Context, userID uuid.UUID, telegramID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE platform_users SET telegram_id = $2 WHERE id = $1`, userID, telegramID)
	return err
}

func (s *Store) OwnerTelegramID(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var tid *int64
	err := s.pool.QueryRow(ctx, `
		SELECT u.telegram_id FROM organizations w
		JOIN platform_users u ON u.id = w.owner_user_id
		WHERE w.id = $1`, organizationID).Scan(&tid)
	if err != nil || tid == nil {
		return 0, fmt.Errorf("owner telegram not linked")
	}
	return *tid, nil
}

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

func (s *Store) UpsertWebIdentity(ctx context.Context, email, _ string, avatarURL, authMethod string) (PlatformUser, error) {
	sub := auth.SubEmail(email)
	const q = `
	INSERT INTO platform_users (auth_sub, email, avatar_url, auth_method)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (auth_sub) DO UPDATE SET
	  email = EXCLUDED.email,
	  avatar_url = CASE WHEN EXCLUDED.avatar_url <> '' THEN EXCLUDED.avatar_url ELSE platform_users.avatar_url END,
	  auth_method = EXCLUDED.auth_method
	RETURNING id, auth_sub, email, telegram_id, avatar_url, auth_method, created_at`
	var u PlatformUser
	err := s.pool.QueryRow(ctx, q, sub, email, avatarURL, authMethod).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &u.AvatarURL, &u.AuthMethod, &u.CreatedAt)
	return u, err
}
