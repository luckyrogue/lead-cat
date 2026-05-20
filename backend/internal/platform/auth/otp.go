package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const otpTTL = 10 * time.Minute

type OTP struct {
	rdb      *redis.Client
	log      *zap.Logger
	logCodes bool
}

func NewOTP(rdb *redis.Client, log *zap.Logger, logCodes bool) *OTP {
	return &OTP{rdb: rdb, log: log, logCodes: logCodes}
}

func (o *OTP) Send(ctx context.Context, channel, dest string) (string, error) {
	dest = normalizeDest(channel, dest)
	if dest == "" {
		return "", fmt.Errorf("invalid destination")
	}
	code, err := randomCode(6)
	if err != nil {
		return "", err
	}
	key := otpKey(channel, dest)
	if err := o.rdb.Set(ctx, key, code, otpTTL).Err(); err != nil {
		return "", err
	}
	if o.logCodes {
		if o.log != nil {
			o.log.Info("auth OTP (local dev — no email/SMS)",
				zap.String("channel", channel),
				zap.String("dest", dest),
				zap.String("code", code),
			)
		} else {
			fmt.Printf("[auth-otp] %s %s code=%s\n", channel, dest, code)
		}
	}
	return code, nil
}

func (o *OTP) Verify(ctx context.Context, channel, dest, code string) (bool, error) {
	dest = normalizeDest(channel, dest)
	key := otpKey(channel, dest)
	stored, err := o.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != strings.TrimSpace(code) {
		return false, nil
	}
	_ = o.rdb.Del(ctx, key).Err()
	return true, nil
}

func otpKey(channel, dest string) string {
	return "leadcat:otp:" + channel + ":" + dest
}

func normalizeDest(channel, dest string) string {
	dest = strings.TrimSpace(strings.ToLower(dest))
	if channel == "phone" {
		dest = strings.TrimPrefix(dest, "+")
		if dest == "" {
			return ""
		}
		return "+" + dest
	}
	return dest
}

func randomCode(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + v.Int64()))
	}
	return b.String(), nil
}

func SubEmail(email string) string  { return "email:" + strings.ToLower(strings.TrimSpace(email)) }
func SubPhone(phone string) string  { return "phone:" + normalizeDest("phone", phone) }
func SubGitHub(id int64) string     { return fmt.Sprintf("github:%d", id) }
func SubGitLab(id int64) string     { return fmt.Sprintf("gitlab:%d", id) }
