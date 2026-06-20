// Package boti18n is a dependency-free translation catalog for Telegram bot text.
// Catalog entries are registered by per-domain files (catalog_*.go) via init().
package boti18n

import (
	"fmt"
	"strings"
)

// catalog maps a message key to a map of language → format string.
var catalog = map[string]map[string]string{}

// register merges domain entries into the catalog. Called from init() in
// per-domain catalog_*.go files.
func register(entries map[string]map[string]string) {
	for key, langs := range entries {
		catalog[key] = langs
	}
}

// Normalize returns a supported language code (ru|en|kk), defaulting to ru.
// A region subtag is stripped first ("en-US" → "en").
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

// Resolve picks the effective language: a non-empty stored preference wins
// (normalized); otherwise the Telegram language_code (normalized); otherwise ru.
func Resolve(stored, telegramCode string) string {
	if strings.TrimSpace(stored) != "" {
		return Normalize(stored)
	}
	return Normalize(telegramCode)
}

// T returns the catalog string for key in the given language, applying args via
// fmt.Sprintf. Missing key → key returned verbatim; key present but missing the
// language → ru fallback.
func T(lang, key string, args ...any) string {
	entry, ok := catalog[key]
	if !ok {
		return key
	}
	s, ok := entry[Normalize(lang)]
	if !ok {
		s = entry["ru"]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
