package emailtemplates

func NormalizeLang(lang string) string {
	switch lang {
	case "en", "kk":
		return lang
	default:
		return "ru"
	}
}
