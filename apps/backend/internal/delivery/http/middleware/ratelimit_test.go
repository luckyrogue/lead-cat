package middleware_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
)

func newRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func appWith(h fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Get("/x", h, func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func reqIP(app *fiber.App, ip string) int {
	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(r, 5000)
	if err != nil {
		return 0
	}
	return resp.StatusCode
}

func TestRateLimit_AllowsThenBlocks(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 3, time.Minute, "t"))
	for i := 0; i < 3; i++ {
		if code := reqIP(app, "1.1.1.1"); code != 200 {
			t.Fatalf("req %d: want 200 got %d", i, code)
		}
	}
	if code := reqIP(app, "1.1.1.1"); code != 429 {
		t.Fatalf("4th: want 429 got %d", code)
	}
}

func TestRateLimit_IndependentIPs(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t"))
	if reqIP(app, "1.1.1.1") != 200 || reqIP(app, "2.2.2.2") != 200 {
		t.Fatal("distinct IPs must each be allowed once")
	}
	if reqIP(app, "1.1.1.1") != 429 {
		t.Fatal("second hit from same IP must be 429")
	}
}

func TestRateLimit_FailOpenOnRedisError(t *testing.T) {
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{
		Addr:        mr.Addr(),
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
		MaxRetries:  0,
	})
	mr.Close()
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t"))
	if code := reqIP(app, "1.1.1.1"); code != 200 {
		t.Fatalf("fail-open: want 200 got %d", code)
	}
}
