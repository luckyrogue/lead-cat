package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateWebSession(ctx context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time, ua, ip string) (WebSession, error) {
	const q = `INSERT INTO web_sessions (token_hash, user_id, expires_at, user_agent, ip)
	  VALUES ($1,$2,$3,$4,$5)
	  RETURNING id, token_hash, user_id, created_at, last_seen_at, expires_at, revoked_at, user_agent, ip`
	var w WebSession
	err := s.pool.QueryRow(ctx, q, tokenHash, userID, expiresAt, ua, ip).
		Scan(&w.ID, &w.TokenHash, &w.UserID, &w.CreatedAt, &w.LastSeenAt, &w.ExpiresAt, &w.RevokedAt, &w.UserAgent, &w.IP)
	return w, err
}

func (s *Store) ResolveWebSession(ctx context.Context, tokenHash []byte, now time.Time) (WebSession, bool, error) {
	const q = `SELECT id, token_hash, user_id, created_at, last_seen_at, expires_at, revoked_at, user_agent, ip
	  FROM web_sessions WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at > $2`
	var w WebSession
	err := s.pool.QueryRow(ctx, q, tokenHash, now).
		Scan(&w.ID, &w.TokenHash, &w.UserID, &w.CreatedAt, &w.LastSeenAt, &w.ExpiresAt, &w.RevokedAt, &w.UserAgent, &w.IP)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebSession{}, false, nil
	}
	if err != nil {
		return WebSession{}, false, err
	}
	return w, true, nil
}

func (s *Store) TouchWebSession(ctx context.Context, id uuid.UUID, lastSeen, expiresAt time.Time) error {
	const q = `UPDATE web_sessions SET last_seen_at=$2, expires_at=$3 WHERE id=$1`
	_, err := s.pool.Exec(ctx, q, id, lastSeen, expiresAt)
	return err
}

func (s *Store) RevokeWebSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	const q = `UPDATE web_sessions SET revoked_at=$2 WHERE token_hash=$1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, q, tokenHash, now)
	return err
}
