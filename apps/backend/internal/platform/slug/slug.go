package slug

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Make turns an arbitrary name into a lowercase ascii slug: diacritics are
// folded, runs of non-alphanumeric characters become single hyphens, and
// leading/trailing hyphens are trimmed.
func Make(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, name)
	if err != nil {
		folded = name
	}
	lower := strings.ToLower(folded)

	var sb strings.Builder
	inSep := true
	for _, r := range lower {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
			inSep = false
		} else {
			if !inSep {
				sb.WriteRune('-')
				inSep = true
			}
		}
	}
	return strings.TrimRight(sb.String(), "-")
}
