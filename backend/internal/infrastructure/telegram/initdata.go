package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type InitDataUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type InitDataValidator struct {
	botToken string
}

func NewInitDataValidator(botToken string) *InitDataValidator {
	return &InitDataValidator{botToken: botToken}
}

func (v *InitDataValidator) Validate(initData string) (InitDataUser, error) {
	if err := v.verify(initData); err != nil {
		return InitDataUser{}, err
	}
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return InitDataUser{}, err
	}
	var u InitDataUser
	if err := json.Unmarshal([]byte(vals.Get("user")), &u); err != nil {
		return InitDataUser{}, fmt.Errorf("parse user: %w", err)
	}
	if u.ID == 0 {
		return InitDataUser{}, fmt.Errorf("missing user id")
	}
	return u, nil
}

func (v *InitDataValidator) verify(initData string) error {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return err
	}
	hash := vals.Get("hash")
	if hash == "" {
		return fmt.Errorf("missing hash")
	}
	vals.Del("hash")
	var pairs []string
	for k := range vals {
		pairs = append(pairs, k+"="+vals.Get(k))
	}
	sort.Strings(pairs)
	dataCheck := strings.Join(pairs, "\n")
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(v.botToken))
	key := secret.Sum(nil)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(dataCheck))
	if hex.EncodeToString(mac.Sum(nil)) != hash {
		return fmt.Errorf("invalid initData")
	}
	return nil
}
