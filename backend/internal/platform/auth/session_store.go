package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	rdb *redis.Client
}

func NewSessionStore(rdb *redis.Client) *SessionStore {
	return &SessionStore{rdb: rdb}
}

func (s *SessionStore) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, "leadcat:auth:"+key, b, ttl).Err()
}

func (s *SessionStore) Get(ctx context.Context, key string, dest any) (bool, error) {
	b, err := s.rdb.Get(ctx, "leadcat:auth:"+key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SessionStore) Delete(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, "leadcat:auth:"+key).Err()
}
