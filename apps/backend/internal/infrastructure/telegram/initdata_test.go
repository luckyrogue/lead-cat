package telegram_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
)

const testBotToken = "123456:TEST-BotTokenForInitData"

func signInitData(botToken string, pairs map[string]string) string {
	vals := url.Values{}
	for k, v := range pairs {
		vals.Set(k, v)
	}
	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + vals.Get(k)
	}
	dataCheck := strings.Join(lines, "\n")
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	key := secret.Sum(nil)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(dataCheck))
	vals.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return vals.Encode()
}

func TestInitDataValidator_Valid(t *testing.T) {
	v := telegram.NewInitDataValidator(testBotToken)
	initData := signInitData(testBotToken, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":12345,"username":"dev"}`,
	})
	u, err := v.Validate(initData)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 12345 {
		t.Fatalf("id %d", u.ID)
	}
}

func TestInitDataValidator_TamperedHashRejected(t *testing.T) {
	v := telegram.NewInitDataValidator(testBotToken)
	initData := signInitData(testBotToken, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":12345}`,
	})
	tampered := strings.Replace(initData, "12345", "99999", 1)
	if _, err := v.Validate(tampered); err == nil {
		t.Fatal("expected invalid initData")
	}
}

func TestFreshAuthDate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if !telegram.FreshAuthDate(now.Unix(), now, 24*time.Hour) {
		t.Fatal("fresh")
	}
	if telegram.FreshAuthDate(now.Add(-25*time.Hour).Unix(), now, 24*time.Hour) {
		t.Fatal("stale should fail")
	}
}
