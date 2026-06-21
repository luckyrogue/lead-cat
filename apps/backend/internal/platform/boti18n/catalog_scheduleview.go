package boti18n

func init() {
	register(map[string]map[string]string{
		"sched.start":           {"ru": "Чьё расписание показать? Введи email сотрудника или часть имени:", "en": "Whose schedule? Enter an employee email or part of a name:", "kk": "Кімнің кестесі? Қызметкердің email-ін немесе атының бөлігін енгіз:"},
		"sched.session_expired": {"ru": "Сессия истекла. Начни заново: /schedule", "en": "Session expired. Start over: /schedule", "kk": "Сессия аяқталды. Қайта баста: /schedule"},
		"sched.search_failed":   {"ru": "Не удалось выполнить поиск, попробуй ещё раз:", "en": "Search failed, try again:", "kk": "Іздеу сәтсіз, қайта көр:"},
		"sched.schedule_btn":    {"ru": "Расписание %[1]s", "en": "Schedule of %[1]s", "kk": "%[1]s кестесі"},
		"sched.none_found":      {"ru": "Ничего не найдено. Введи корректный email или часть имени:", "en": "Nothing found. Enter a valid email or part of a name:", "kk": "Ештеңе табылмады. Дұрыс email немесе атының бөлігін енгіз:"},
		"sched.pick":            {"ru": "Выбери сотрудника:", "en": "Pick an employee:", "kk": "Қызметкерді таңда:"},
		"sched.not_found_retry": {"ru": "Не найдено, начни заново: /schedule", "en": "Not found, start over: /schedule", "kk": "Табылмады, қайта баста: /schedule"},
		"sched.enter_date":      {"ru": "Введи дату ГГГГ-ММ-ДД:", "en": "Enter a date YYYY-MM-DD:", "kk": "Күнді енгіз ЖЖЖЖ-АА-КК:"},
		"sched.enter_range":     {"ru": "Введи диапазон ГГГГ-ММ-ДД..ГГГГ-ММ-ДД:", "en": "Enter a range YYYY-MM-DD..YYYY-MM-DD:", "kk": "Аралықты енгіз ЖЖЖЖ-АА-КК..ЖЖЖЖ-АА-КК:"},
		"sched.bad_date":        {"ru": "Неверная дата (ГГГГ-ММ-ДД). Попробуй ещё раз:", "en": "Invalid date (YYYY-MM-DD). Try again:", "kk": "Қате күн (ЖЖЖЖ-АА-КК). Қайта көр:"},
		"sched.bad_range":       {"ru": "Неверный формат диапазона (ГГГГ-ММ-ДД..ГГГГ-ММ-ДД). Попробуй ещё раз:", "en": "Invalid range (YYYY-MM-DD..YYYY-MM-DD). Try again:", "kk": "Аралық форматы қате (ЖЖЖЖ-АА-КК..ЖЖЖЖ-АА-КК). Қайта көр:"},
		"sched.get_failed":      {"ru": "Не удалось получить расписание, попробуй позже.", "en": "Couldn't fetch the schedule, try later.", "kk": "Кестені алу мүмкін болмады, кейінірек көр."},
		"sched.pick_period":     {"ru": "Расписание %[1]s. Выбери период:", "en": "Schedule of %[1]s. Pick a period:", "kk": "%[1]s кестесі. Кезеңді таңда:"},
		"sched.btn.today":       {"ru": "Сегодня", "en": "Today", "kk": "Бүгін"},
		"sched.btn.tomorrow":    {"ru": "Завтра", "en": "Tomorrow", "kk": "Ертең"},
		"sched.btn.upcoming":    {"ru": "Все предстоящие", "en": "All upcoming", "kk": "Барлық алдағы"},
		"sched.btn.date":        {"ru": "Конкретная дата", "en": "Specific date", "kk": "Нақты күн"},
		"sched.btn.range":       {"ru": "Диапазон", "en": "Range", "kk": "Аралық"},
		"sched.btn.back":        {"ru": "⬅ Другой сотрудник", "en": "⬅ Another employee", "kk": "⬅ Басқа қызметкер"},
		"sched.btn.periods":     {"ru": "⬅ Периоды", "en": "⬅ Periods", "kk": "⬅ Кезеңдер"},
		"sched.period.today":    {"ru": "сегодня", "en": "today", "kk": "бүгін"},
		"sched.period.tomorrow": {"ru": "завтра", "en": "tomorrow", "kk": "ертең"},
		"sched.period.upcoming": {"ru": "все предстоящие", "en": "all upcoming", "kk": "барлық алдағы"},
		"sched.header":          {"ru": "Расписание %[1]s: %[2]s\n", "en": "Schedule of %[1]s: %[2]s\n", "kk": "%[1]s кестесі: %[2]s\n"},
		"sched.no_meetings":     {"ru": "Встреч нет.", "en": "No meetings.", "kk": "Кездесулер жоқ."},
	})
}
