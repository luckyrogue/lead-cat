package postgres

import (
	"context"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error {
	atEnc, err := s.cipher.Encrypt(conn.AccessToken)
	if err != nil {
		return err
	}
	rtEnc, err := s.cipher.Encrypt(conn.RefreshToken)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO calendar_connections (email, provider, access_token_enc, refresh_token_enc, expiry, scopes, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (email, provider) DO UPDATE SET
			access_token_enc = EXCLUDED.access_token_enc,
			refresh_token_enc = EXCLUDED.refresh_token_enc,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			updated_at = now()`,
		conn.Email, conn.Provider, atEnc, rtEnc, conn.Expiry, conn.Scopes)
	return err
}

func (s *Store) GetCalendarConnection(ctx context.Context, email, provider string) (model.CalendarConnection, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT email, provider, access_token_enc, refresh_token_enc, expiry, scopes, connected_at, updated_at
		FROM calendar_connections WHERE email = $1 AND provider = $2`, email, provider)
	return scanCalendarConnection(s, row)
}

func (s *Store) ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT email, provider, access_token_enc, refresh_token_enc, expiry, scopes, connected_at, updated_at
		FROM calendar_connections WHERE email = $1 ORDER BY provider`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.CalendarConnection
	for rows.Next() {
		conn, err := scanCalendarConnection(s, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, conn)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCalendarConnection(ctx context.Context, email, provider string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM calendar_connections WHERE email = $1 AND provider = $2`, email, provider)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanCalendarConnection(s *Store, row rowScanner) (model.CalendarConnection, error) {
	var (
		conn         model.CalendarConnection
		atEnc, rtEnc []byte
		connectedAt  time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(&conn.Email, &conn.Provider, &atEnc, &rtEnc, &conn.Expiry, &conn.Scopes, &connectedAt, &updatedAt); err != nil {
		return model.CalendarConnection{}, err
	}
	at, err := s.cipher.Decrypt(atEnc)
	if err != nil {
		return model.CalendarConnection{}, err
	}
	rt, err := s.cipher.Decrypt(rtEnc)
	if err != nil {
		return model.CalendarConnection{}, err
	}
	conn.AccessToken, conn.RefreshToken = at, rt
	conn.ConnectedAt, conn.UpdatedAt = connectedAt, updatedAt
	return conn, nil
}
