package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func clientIP(c *fiber.Ctx, trustProxy bool) string {
	if trustProxy {
		if xri := strings.TrimSpace(c.Get("X-Real-IP")); xri != "" {
			return xri
		}
	}
	return c.IP()
}

func RateLimit(rdb *redis.Client, log *zap.Logger, max int, window time.Duration, prefix string, trustProxy bool) fiber.Handler {
	windowSecs := int64(window.Seconds())
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		bucket := time.Now().Unix() / windowSecs
		key := fmt.Sprintf("ratelimit:%s:%s:%d", prefix, clientIP(c, trustProxy), bucket)
		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			if log != nil {
				log.Warn("ratelimit_redis_error", zap.String("prefix", prefix), zap.Error(err))
			}
			return c.Next()
		}
		if n == 1 {
			_ = rdb.Expire(ctx, key, window).Err()
		}
		if n > int64(max) {
			c.Set("Retry-After", strconv.FormatInt(windowSecs, 10))
			return fiber.NewError(fiber.StatusTooManyRequests, "rate_limited")
		}
		return c.Next()
	}
}
