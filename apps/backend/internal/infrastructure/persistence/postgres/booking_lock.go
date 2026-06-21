package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) WithHostBookingLock(ctx context.Context, hostUserID uuid.UUID, start time.Time, fn func(ctx context.Context) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockKey := fmt.Sprintf("%s:%d", hostUserID.String(), start.Unix()/60)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return err
	}
	if err := fn(ctx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
