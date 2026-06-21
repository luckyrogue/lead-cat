package boti18n

import (
	"fmt"
	"strings"
)

var catalog = map[string]map[string]string{}

func register(entries map[string]map[string]string) {
	for key, langs := range entries {
		catalog[key] = langs
	}
}

func Normalize(lang string) string {
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		lang = lang[:i]
	}
	switch lang {
	case "en", "kk":
		return lang
	default:
		return "ru"
	}
}

func Resolve(stored, telegramCode string) string {
	if strings.TrimSpace(stored) != "" {
		return Normalize(stored)
	}
	return Normalize(telegramCode)
}

func T(lang, key string, args ...any) string {
	entry, ok := catalog[key]
	if !ok {
		return key
	}
	s, ok := entry[Normalize(lang)]
	if !ok {
		s = entry["ru"]
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}
