package boti18n

func init() {
	register(map[string]map[string]string{
		"agentbook.created":                   {"ru": "Встреча создана ✅", "en": "Meeting created ✅", "kk": "Кездесу құрылды ✅"},
		"agentbook.register_first":            {"ru": "Сначала зарегистрируйся: /start", "en": "Register first: /start", "kk": "Алдымен тіркел: /start"},
		"agentbook.google_not_configured":     {"ru": "Google-календарь не подключён — обратись к администратору.", "en": "Google Calendar isn't connected — contact your administrator.", "kk": "Google күнтізбесі қосылмаған — әкімшіге хабарлас."},
		"agentbook.telegram_linked_elsewhere": {"ru": "Этот Telegram привязан к другому аккаунту.", "en": "This Telegram is linked to another account.", "kk": "Бұл Telegram басқа аккаунтқа байланған."},
		"agentbook.bad_input":                 {"ru": "Проверь данные встречи — что-то не так с датой или временем.", "en": "Check the meeting details — something's off with the date or time.", "kk": "Кездесу деректерін тексер — күн не уақытта бірдеңе дұрыс емес."},
		"agentbook.create_failed":             {"ru": "Не удалось создать встречу, попробуй позже 🐾", "en": "Couldn't create the meeting, try later 🐾", "kk": "Кездесу құру мүмкін болмады, кейінірек көр 🐾"},
	})
}
