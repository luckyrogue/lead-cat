package scheduler_agent

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionTTL = 30 * time.Minute

type RedisSessions struct {
	rdb *redis.Client
}

func NewRedisSessions(rdb *redis.Client) *RedisSessions {
	return &RedisSessions{rdb: rdb}
}

func (r *RedisSessions) key(telegramID int64) string {
	return "agent:session:" + strconv.FormatInt(telegramID, 10)
}

func (r *RedisSessions) Get(ctx context.Context, telegramID int64) (*State, error) {
	raw, err := r.rdb.Get(ctx, r.key(telegramID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *RedisSessions) Set(ctx context.Context, telegramID int64, s State) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, r.key(telegramID), raw, sessionTTL).Err()
}

func (r *RedisSessions) Del(ctx context.Context, telegramID int64) error {
	return r.rdb.Del(ctx, r.key(telegramID)).Err()
}

func (r *RedisSessions) ClaimBooking(ctx context.Context, telegramID int64, signature string) (bool, error) {
	key := "agent:book:" + strconv.FormatInt(telegramID, 10) + ":" + signature
	return r.rdb.SetNX(ctx, key, "1", 10*time.Minute).Result()
}
