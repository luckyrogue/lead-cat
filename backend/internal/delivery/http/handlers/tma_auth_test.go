package handlers

import "testing"

func TestLooksLikeTelegramInitData(t *testing.T) {
	t.Parallel()
	if !looksLikeTelegramInitData("query_id=AA&user=%7B%7D&auth_date=1&hash=abc") {
		t.Fatal("expected telegram initData shape")
	}
	if looksLikeTelegramInitData("123456789") {
		t.Fatal("numeric dev id must not look like initData")
	}
	if looksLikeTelegramInitData("") {
		t.Fatal("empty must not look like initData")
	}
}
