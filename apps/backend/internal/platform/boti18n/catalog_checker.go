package boti18n

func init() {
	register(map[string]map[string]string{
		"checker.start":               {"ru": "Поиск общего свободного времени.\nВведи имя или email участника:", "en": "Find common free time.\nEnter a participant's name or email:", "kk": "Ортақ бос уақытты іздеу.\nҚатысушының атын немесе email-ін енгіз:"},
		"checker.session_expired":     {"ru": "Сессия истекла. Начни заново: /checker", "en": "Session expired. Start over: /checker", "kk": "Сессия аяқталды. Қайта баста: /checker"},
		"checker.search_failed":       {"ru": "Не удалось выполнить поиск, попробуй ещё раз:", "en": "Search failed, try again:", "kk": "Іздеу сәтсіз аяқталды, қайта көр:"},
		"checker.none_found":          {"ru": "Ничего не найдено. Введи другой запрос:", "en": "Nothing found. Try another query:", "kk": "Ештеңе табылмады. Басқа сұраныс енгіз:"},
		"checker.pick":                {"ru": "Выбери участника (можно несколько):", "en": "Pick participants (you can add several):", "kk": "Қатысушыларды таңда (бірнешеуін болады):"},
		"checker.btn_done":            {"ru": "Готово ✅", "en": "Done ✅", "kk": "Дайын ✅"},
		"checker.not_found_retry":     {"ru": "Не найдено, поищи ещё раз:", "en": "Not found, search again:", "kk": "Табылмады, қайта ізде:"},
		"checker.already_added":       {"ru": "Уже добавлен. Ищи ещё или нажми «Готово».", "en": "Already added. Search more or press “Done”.", "kk": "Қосылған. Тағы ізде немесе «Дайын» бас."},
		"checker.added":               {"ru": "Добавлен: %[1]s\nУчастников: %[2]d. Ищи ещё или нажми «Готово».", "en": "Added: %[1]s\nParticipants: %[2]d. Search more or press “Done”.", "kk": "Қосылды: %[1]s\nҚатысушылар: %[2]d. Тағы ізде немесе «Дайын» бас."},
		"checker.need_one":            {"ru": "Добавь хотя бы одного участника.", "en": "Add at least one participant.", "kk": "Кемінде бір қатысушы қос."},
		"checker.enter_range":         {"ru": "Введи диапазон дат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД", "en": "Enter a date range: YYYY-MM-DD..YYYY-MM-DD", "kk": "Күн аралығын енгіз: ЖЖЖЖ-АА-КК..ЖЖЖЖ-АА-КК"},
		"checker.bad_range":           {"ru": "Неверный формат диапазона. Введи ГГГГ-ММ-ДД..ГГГГ-ММ-ДД и попробуй ещё раз:", "en": "Invalid range format. Enter YYYY-MM-DD..YYYY-MM-DD and try again:", "kk": "Аралық форматы қате. ЖЖЖЖ-АА-КК..ЖЖЖЖ-АА-КК енгізіп қайта көр:"},
		"checker.pick_duration":       {"ru": "Выбери длительность встречи:", "en": "Pick the meeting duration:", "kk": "Кездесу ұзақтығын таңда:"},
		"checker.bad_duration":        {"ru": "Неверная длительность.", "en": "Invalid duration.", "kk": "Ұзақтық қате."},
		"checker.search_failed_later": {"ru": "Не удалось выполнить поиск, попробуй позже.", "en": "Search failed, try later.", "kk": "Іздеу сәтсіз, кейінірек көр."},
		"checker.no_slots":            {"ru": "Общих свободных слотов в выбранном диапазоне не найдено.\nПопробуй: расширить диапазон дат / уменьшить длительность / изменить состав участников.", "en": "No common free slots in the selected range.\nTry: widen the date range / shorten the duration / change the participants.", "kk": "Таңдалған аралықта ортақ бос уақыт табылмады.\nКөр: аралықты кеңейт / ұзақтықты қысқарт / қатысушыларды өзгерт."},
		"checker.dur.15m":             {"ru": "15 мин", "en": "15 min", "kk": "15 мин"},
		"checker.dur.30m":             {"ru": "30 мин", "en": "30 min", "kk": "30 мин"},
		"checker.dur.45m":             {"ru": "45 мин", "en": "45 min", "kk": "45 мин"},
		"checker.dur.1h":              {"ru": "1 час", "en": "1 hour", "kk": "1 сағат"},
		"checker.dur.1_5h":            {"ru": "1.5 часа", "en": "1.5 hours", "kk": "1.5 сағат"},
		"checker.dur.2h":              {"ru": "2 часа", "en": "2 hours", "kk": "2 сағат"},
		"checker.slots_header":        {"ru": "✅ Общее свободное время для %[1]d участников:\n\n", "en": "✅ Common free time for %[1]d participants:\n\n", "kk": "✅ %[1]d қатысушы үшін ортақ бос уақыт:\n\n"},
		"checker.slot_mins":           {"ru": "%[1]d мин свободно", "en": "%[1]d min free", "kk": "%[1]d мин бос"},
	})
}
