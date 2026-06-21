package boti18n

func init() {
	register(map[string]map[string]string{
		"agent.start":               {"ru": "Спроси меня про расписание — например: «когда у Миа и Алекса есть общий час на следующей неделе?» 🐾", "en": "Ask me about schedules — e.g. \"when do Mia and Alex share a free hour next week?\" 🐾", "kk": "Кестелер туралы сұра — мысалы: «келесі аптада Миа мен Алекстің ортақ бос сағаты қашан?» 🐾"},
		"agent.plan_failed":         {"ru": "Не получилось обработать запрос, попробуй ещё раз чуть позже 🐾", "en": "Couldn't process the request, try again a bit later 🐾", "kk": "Сұрауды өңдеу мүмкін болмады, сәл кейінірек қайта көр 🐾"},
		"agent.too_hard":            {"ru": "Это оказалось сложновато 🐾 Попробуй переформулировать или уточнить участников и даты.", "en": "That turned out tricky 🐾 Try rephrasing or clarifying participants and dates.", "kk": "Бұл күрделірек болды 🐾 Қайта тұжырымда немесе қатысушылар мен күндерді нақтыла."},
		"agent.proposal_stale":      {"ru": "Предложение устарело 🐾 Попроси заново.", "en": "The proposal expired 🐾 Ask again.", "kk": "Ұсыныс ескірді 🐾 Қайта сұра."},
		"agent.booking_unavailable": {"ru": "Бронирование сейчас недоступно.", "en": "Booking is unavailable right now.", "kk": "Брондау қазір қолжетімсіз."},
		"agent.cancelled":           {"ru": "Хорошо, не бронирую 🐾", "en": "Okay, not booking 🐾", "kk": "Жарайды, брондамаймын 🐾"},
		"agent.btn_confirm":         {"ru": "Подтвердить ✅", "en": "Confirm ✅", "kk": "Растау ✅"},
		"agent.btn_cancel":          {"ru": "Отмена", "en": "Cancel", "kk": "Болдырмау"},
		"agent.card_q":              {"ru": "Создать встречу?", "en": "Create the meeting?", "kk": "Кездесу құру керек пе?"},
	})
}
