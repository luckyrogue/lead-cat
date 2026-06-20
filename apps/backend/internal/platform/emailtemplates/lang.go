package emailtemplates

// NormalizeLang returns ru, en, or kk; default ru.
func NormalizeLang(lang string) string {
	switch lang {
	case "en", "kk":
		return lang
	default:
		return "ru"
	}
}
