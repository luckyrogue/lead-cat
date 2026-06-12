package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) InsertMagicLink(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) error {
	const q = `INSERT INTO magic_link_tokens (email, token_hash, purpose, expires_at) VALUES ($1, $2, 'login', $3)`
	_, err := s.pool.Exec(ctx, q, email, tokenHash, expiresAt)
	return err
}

func (s *Store) ConsumeMagicLink(ctx context.Context, tokenHash []byte, now time.Time) (string, bool, error) {
	const q = `UPDATE magic_link_tokens SET consumed_at = $2
	  WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $2
	  RETURNING email`
	var email string
	err := s.pool.QueryRow(ctx, q, tokenHash, now).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return email, true, nil
}
