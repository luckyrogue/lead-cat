package boti18n

func init() {
	register(map[string]map[string]string{
		"botset.title":  {"ru": "⏰ Напоминания о встречах. Выбери, за сколько предупреждать (можно несколько):", "en": "⏰ Meeting reminders. Choose how far ahead to notify (you can pick several):", "kk": "⏰ Кездесу еске салулары. Қанша уақыт бұрын ескертуді таңда (бірнешеуін болады):"},
		"botset.off":    {"ru": "Сейчас напоминания выключены.", "en": "Reminders are currently off.", "kk": "Қазір еске салулар өшірулі."},
		"botset.iv.10m": {"ru": "10м", "en": "10m", "kk": "10м"},
		"botset.iv.15m": {"ru": "15м", "en": "15m", "kk": "15м"},
		"botset.iv.30m": {"ru": "30м", "en": "30m", "kk": "30м"},
		"botset.iv.1h":  {"ru": "1ч", "en": "1h", "kk": "1с"},
		"botset.iv.2h":  {"ru": "2ч", "en": "2h", "kk": "2с"},
		"botset.iv.1d":  {"ru": "1день", "en": "1d", "kk": "1күн"},
	})
}
