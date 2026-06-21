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

func newRedisWithMR(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return mr, redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func newRedis(t *testing.T) *redis.Client {
	_, rdb := newRedisWithMR(t)
	return rdb
}

func appWith(h fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Get("/x", h, func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func reqXRI(app *fiber.App, ip string) int {
	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("X-Real-IP", ip)
	resp, err := app.Test(r, 5000)
	if err != nil {
		return 0
	}
	return resp.StatusCode
}

func TestRateLimit_AllowsThenBlocks(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 3, time.Minute, "t", true, false))
	for i := 0; i < 3; i++ {
		if code := reqXRI(app, "1.1.1.1"); code != 200 {
			t.Fatalf("req %d: want 200 got %d", i, code)
		}
	}
	if code := reqXRI(app, "1.1.1.1"); code != 429 {
		t.Fatalf("4th: want 429 got %d", code)
	}
}

func TestRateLimit_IndependentIPs(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t", true, false))
	if reqXRI(app, "1.1.1.1") != 200 || reqXRI(app, "2.2.2.2") != 200 {
		t.Fatal("distinct IPs must each be allowed once")
	}
	if reqXRI(app, "1.1.1.1") != 429 {
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
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t", true, true))
	if code := reqXRI(app, "1.1.1.1"); code != 200 {
		t.Fatalf("fail-open: want 200 got %d", code)
	}
}

func TestRateLimit_FailClosedOnRedisError(t *testing.T) {
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{
		Addr:        mr.Addr(),
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
		MaxRetries:  0,
	})
	mr.Close()
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t", true, false))
	if code := reqXRI(app, "1.1.1.1"); code != 503 {
		t.Fatalf("fail-closed: want 503 got %d", code)
	}
}

func TestRateLimit_WindowReset(t *testing.T) {
	mr, rdb := newRedisWithMR(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 2, time.Minute, "t", true, false))

	if code := reqXRI(app, "3.3.3.3"); code != 200 {
		t.Fatalf("req 1: want 200 got %d", code)
	}
	if code := reqXRI(app, "3.3.3.3"); code != 200 {
		t.Fatalf("req 2: want 200 got %d", code)
	}
	if code := reqXRI(app, "3.3.3.3"); code != 429 {
		t.Fatalf("req 3 (over limit): want 429 got %d", code)
	}

	mr.FastForward(61 * time.Second)

	if code := reqXRI(app, "3.3.3.3"); code != 200 {
		t.Fatalf("after window reset: want 200 got %d", code)
	}
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	_, rdb := newRedisWithMR(t)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Get("/x", middleware.RateLimit(rdb, zap.NewNop(), 1, time.Minute, "t", true, false), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	if code := reqXRI(app, "4.4.4.4"); code != 200 {
		t.Fatalf("first req: want 200 got %d", code)
	}

	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("X-Real-IP", "4.4.4.4")
	resp, err := app.Test(r, 5000)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Fatalf("second req: want 429 got %d", resp.StatusCode)
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header missing on 429 response")
	}
	if retryAfter != "60" {
		t.Errorf("Retry-After=%q want 60", retryAfter)
	}
}

func TestRateLimit_IgnoresSpoofedHeaderWhenUntrusted(t *testing.T) {
	rdb := newRedis(t)
	app := appWith(middleware.RateLimit(rdb, zap.NewNop(), 2, time.Minute, "t", false, false))

	sendWith := func(xri, xff string) int {
		r := httptest.NewRequest("GET", "/x", nil)
		if xri != "" {
			r.Header.Set("X-Real-IP", xri)
		}
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		resp, err := app.Test(r, 5000)
		if err != nil {
			return 0
		}
		return resp.StatusCode
	}

	if code := sendWith("10.0.0.1", "10.0.0.1"); code != 200 {
		t.Fatalf("req 1 spoofed IP A: want 200 got %d", code)
	}
	if code := sendWith("10.0.0.2", "10.0.0.2"); code != 200 {
		t.Fatalf("req 2 spoofed IP B: want 200 got %d", code)
	}
	if code := sendWith("10.0.0.3", "10.0.0.3"); code != 429 {
		t.Fatalf("req 3 different spoofed IP: want 429 got %d (all share c.IP())", code)
	}
}
