package boti18n

func init() {
	register(map[string]map[string]string{
		"reminder.telegram":     {"ru": "⏰ Напоминание: встреча через %[1]s!", "en": "⏰ Reminder: meeting in %[1]s!", "kk": "⏰ Еске салу: кездесу %[1]s кейін!"},
		"reminder.offset.10m":   {"ru": "10 минут", "en": "10 minutes", "kk": "10 минут"},
		"reminder.offset.15m":   {"ru": "15 минут", "en": "15 minutes", "kk": "15 минут"},
		"reminder.offset.30m":   {"ru": "30 минут", "en": "30 minutes", "kk": "30 минут"},
		"reminder.offset.1h":    {"ru": "1 час", "en": "1 hour", "kk": "1 сағат"},
		"reminder.offset.2h":    {"ru": "2 часа", "en": "2 hours", "kk": "2 сағат"},
		"reminder.offset.1d":    {"ru": "1 день", "en": "1 day", "kk": "1 күн"},
		"reminder.offset.n_min": {"ru": "%[1]d мин", "en": "%[1]d min", "kk": "%[1]d мин"},
	})
}
