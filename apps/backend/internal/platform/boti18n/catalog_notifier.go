package boti18n

func init() {
	register(map[string]map[string]string{
		"notif.created":   {"ru": "📅 Новая встреча", "en": "📅 New meeting", "kk": "📅 Жаңа кездесу"},
		"notif.updated":   {"ru": "✏️ Встреча изменена", "en": "✏️ Meeting updated", "kk": "✏️ Кездесу өзгертілді"},
		"notif.removed":   {"ru": "➖ Вас удалили из встречи", "en": "➖ You were removed from a meeting", "kk": "➖ Сіз кездесуден шығарылдыңыз"},
		"notif.cancelled": {"ru": "❌ Встреча отменена", "en": "❌ Meeting cancelled", "kk": "❌ Кездесу болдырылмады"},
		"notif.added":     {"ru": "➕ Вас добавили на встречу", "en": "➕ You were added to a meeting", "kk": "➕ Сіз кездесуге қосылдыңыз"},
	})
}
