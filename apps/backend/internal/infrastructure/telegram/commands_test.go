package telegram

import (
	"strings"
	"testing"
)

func TestHelpText_Localized(t *testing.T) {
	ru := helpText("ru")
	en := helpText("en")
	kk := helpText("kk")
	if ru == "" || en == "" || kk == "" {
		t.Fatal("help text must be non-empty in all languages")
	}
	if en == ru || kk == ru {
		t.Fatal("help text must differ by language")
	}
	if !strings.Contains(en, "/menu") || !strings.Contains(en, "Lead Cat") {
		t.Errorf("en help text malformed: %q", en)
	}
}

func TestPublicCommands_Localized(t *testing.T) {
	en := PublicCommands("en")
	ru := PublicCommands("ru")
	if len(en) != len(ru) || len(en) == 0 {
		t.Fatalf("command lists must be same non-zero length: en=%d ru=%d", len(en), len(ru))
	}
	if en[0].Description == ru[0].Description {
		t.Errorf("command descriptions must differ by language")
	}
}
