package boti18n

import "testing"

// fixture keys registered for tests (all three languages present).
func init() {
	register(map[string]map[string]string{
		"test.hi":     {"ru": "Привет", "en": "Hi", "kk": "Сәлем"},
		"test.greet":  {"ru": "Привет, %[1]s", "en": "Hi, %[1]s", "kk": "Сәлем, %[1]s"},
		"test.ruonly": {"ru": "ТолькоRU"}, // intentionally missing en/kk for fallback test
	})
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{"ru": "ru", "en": "en", "kk": "kk", "en-US": "en", "kk-KZ": "kk", "": "ru", "fr": "ru"}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve("en", "kk"); got != "en" {
		t.Errorf("stored should win: got %q", got)
	}
	if got := Resolve("", "kk"); got != "kk" {
		t.Errorf("empty stored should use telegram code: got %q", got)
	}
	if got := Resolve("", ""); got != "ru" {
		t.Errorf("both empty should be ru: got %q", got)
	}
	if got := Resolve("garbage", ""); got != "ru" {
		t.Errorf("garbage stored normalizes to ru: got %q", got)
	}
}

func TestT(t *testing.T) {
	if got := T("en", "test.hi"); got != "Hi" {
		t.Errorf("T en = %q", got)
	}
	if got := T("kk", "test.hi"); got != "Сәлем" {
		t.Errorf("T kk = %q", got)
	}
	if got := T("en", "test.greet", "Mia"); got != "Hi, Mia" {
		t.Errorf("T with arg = %q", got)
	}
	// missing language for a present key → ru fallback
	if got := T("en", "test.ruonly"); got != "ТолькоRU" {
		t.Errorf("missing-lang should fall back to ru: %q", got)
	}
	// missing key → key returned verbatim
	if got := T("en", "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key should return key: %q", got)
	}
}

func TestCatalog_AllKeysHaveAllLangs(t *testing.T) {
	for key, langs := range catalog {
		if key == "test.ruonly" {
			continue // intentional fixture gap for the fallback test
		}
		for _, l := range []string{"ru", "en", "kk"} {
			if langs[l] == "" {
				t.Errorf("catalog key %q missing %q translation", key, l)
			}
		}
	}
}
