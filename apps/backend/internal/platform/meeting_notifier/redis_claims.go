package meeting_notifier

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const claimTTL = 24 * time.Hour

type RedisClaims struct {
	rdb *redis.Client
}

func NewRedisClaims(rdb *redis.Client) *RedisClaims {
	return &RedisClaims{rdb: rdb}
}

func (c *RedisClaims) Claim(ctx context.Context, key string) (bool, error) {
	return c.rdb.SetNX(ctx, key, "1", claimTTL).Result()
}
