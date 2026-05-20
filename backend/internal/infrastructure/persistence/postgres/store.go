package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Store struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func New(pool *pgxpool.Pool, log *zap.Logger) *Store {
	return &Store{pool: pool, log: log}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
