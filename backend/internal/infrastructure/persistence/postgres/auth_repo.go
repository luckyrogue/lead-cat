package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Passkey struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CredentialID []byte
	PublicKey    []byte
	SignCount    uint32
	DeviceName   string
}

func (s *Store) UpsertUserIdentity(ctx context.Context, authSub, email, phone string) (User, error) {
	var u User
	var phoneVal *string
	if phone != "" {
		phoneVal = &phone
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO platform_users (auth_sub, email, phone)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth_sub) DO UPDATE SET
			email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE platform_users.email END,
			phone = COALESCE(EXCLUDED.phone, platform_users.phone)
		RETURNING id, auth_sub, email, telegram_id, phone`,
		authSub, email, phoneVal).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &phoneVal)
	if phoneVal != nil {
		u.Phone = *phoneVal
	}
	return u, err
}

func (s *Store) ListPasskeys(ctx context.Context, userID uuid.UUID) ([]Passkey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, credential_id, public_key, sign_count, device_name
		FROM auth_passkeys WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Passkey
	for rows.Next() {
		var p Passkey
		var sc int64
		if err := rows.Scan(&p.ID, &p.UserID, &p.CredentialID, &p.PublicKey, &sc, &p.DeviceName); err != nil {
			return nil, err
		}
		p.SignCount = uint32(sc)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPasskeyByCredentialID(ctx context.Context, credentialID []byte) (Passkey, User, error) {
	var p Passkey
	var u User
	var sc int64
	var phone *string
	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.user_id, p.credential_id, p.public_key, p.sign_count, p.device_name,
			u.id, u.auth_sub, u.email, u.telegram_id, u.phone
		FROM auth_passkeys p
		JOIN platform_users u ON u.id = p.user_id
		WHERE p.credential_id = $1`, credentialID).
		Scan(&p.ID, &p.UserID, &p.CredentialID, &p.PublicKey, &sc, &p.DeviceName,
			&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &phone)
	if err != nil {
		return Passkey{}, User{}, err
	}
	p.SignCount = uint32(sc)
	if phone != nil {
		u.Phone = *phone
	}
	return p, u, nil
}

func (s *Store) SavePasskey(ctx context.Context, userID uuid.UUID, credentialID, publicKey []byte, signCount uint32, deviceName string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO auth_passkeys (user_id, credential_id, public_key, sign_count, device_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (credential_id) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			sign_count = EXCLUDED.sign_count`,
		userID, credentialID, publicKey, signCount, deviceName)
	return err
}

func (s *Store) UpdatePasskeySignCount(ctx context.Context, credentialID []byte, signCount uint32) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_passkeys SET sign_count = $2 WHERE credential_id = $1`,
		credentialID, signCount)
	return err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	var phone *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, auth_sub, email, telegram_id, phone FROM platform_users WHERE id = $1`, id).
		Scan(&u.ID, &u.AuthSub, &u.Email, &u.TelegramID, &phone)
	if err == pgx.ErrNoRows {
		return User{}, err
	}
	if phone != nil {
		u.Phone = *phone
	}
	return u, err
}
