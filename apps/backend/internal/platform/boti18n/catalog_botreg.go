package boti18n

func init() {
	register(map[string]map[string]string{
		"botreg.welcome_back": {
			"ru": "С возвращением! 🐾 Открой приложение из меню.",
			"en": "Welcome back! 🐾 Open the app from the menu.",
			"kk": "Қайта келдің! 🐾 Мәзірден қосымшаны аш.",
		},
		"botreg.start": {
			"ru": "Привет! Давай зарегистрируемся.\nВведи ФИО (Фамилия Имя Отчество):",
			"en": "Hi! Let's get you registered.\nEnter your full name:",
			"kk": "Сәлем! Тіркелейік.\nТолық атыңды енгіз:",
		},
		"botreg.ask_name": {
			"ru": "Введи ФИО:", "en": "Enter your full name:", "kk": "Толық атыңды енгіз:",
		},
		"botreg.ask_email": {
			"ru": "Теперь корпоративную почту:", "en": "Now your work email:", "kk": "Енді жұмыс поштаңды енгіз:",
		},
		"botreg.bad_email": {
			"ru": "Не похоже на email. Попробуй ещё раз:", "en": "That doesn't look like an email. Try again:", "kk": "Бұл email емес сияқты. Қайта көр:",
		},
		"botreg.email_taken": {
			"ru": "Эта почта уже привязана к другому аккаунту.", "en": "This email is already linked to another account.", "kk": "Бұл пошта басқа аккаунтқа тіркелген.",
		},
		"botreg.failed": {
			"ru": "Не удалось завершить регистрацию, попробуй позже.", "en": "Couldn't finish registration, please try later.", "kk": "Тіркеуді аяқтау мүмкін болмады, кейінірек көр.",
		},
		"botreg.done": {
			"ru": "Готово, %[1]s! 🐾", "en": "Done, %[1]s! 🐾", "kk": "Дайын, %[1]s! 🐾",
		},
	})
}
