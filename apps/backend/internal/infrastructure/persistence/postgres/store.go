package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
)

type Store struct {
	pool   *pgxpool.Pool
	log    *zap.Logger
	cipher *crypto.TokenCipher
}

func New(pool *pgxpool.Pool, log *zap.Logger, cipher *crypto.TokenCipher) *Store {
	return &Store{pool: pool, log: log, cipher: cipher}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
