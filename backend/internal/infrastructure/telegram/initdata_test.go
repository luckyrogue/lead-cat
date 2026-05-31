package telegram

import (
	"testing"
	"time"
)

func TestFreshAuthDate(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	if !FreshAuthDate(now.Unix()-60, now, time.Hour) {
		t.Fatal("recent auth_date should be fresh")
	}
	if FreshAuthDate(now.Unix()-2*3600, now, time.Hour) {
		t.Fatal("2h-old auth_date should be stale for a 1h window")
	}
	if FreshAuthDate(0, now, time.Hour) {
		t.Fatal("absent auth_date (0) should be stale")
	}
}
