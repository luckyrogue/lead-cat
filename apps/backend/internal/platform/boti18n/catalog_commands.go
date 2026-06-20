package boti18n

func init() {
	register(map[string]map[string]string{
		"cmd.menu":     {"ru": "Открыть приложение", "en": "Open the app", "kk": "Қосымшаны ашу"},
		"cmd.new":      {"ru": "Запланировать встречу", "en": "Schedule a meeting", "kk": "Кездесу жоспарлау"},
		"cmd.schedule": {"ru": "Расписание коллеги", "en": "Colleague's schedule", "kk": "Әріптестің кестесі"},
		"cmd.checker":  {"ru": "Найти свободный слот", "en": "Find a free slot", "kk": "Бос уақыт табу"},
		"cmd.settings": {"ru": "Напоминания", "en": "Reminders", "kk": "Еске салулар"},
		"cmd.edit":     {"ru": "Редактировать встречи", "en": "Edit meetings", "kk": "Кездесулерді өңдеу"},
		"cmd.help":     {"ru": "Помощь", "en": "Help", "kk": "Көмек"},
		"cmd.help.text": {
			"ru": "Lead Cat — помощник по встречам 🐾\n\n/menu — открыть приложение\n/new — запланировать встречу\n/schedule — расписание коллеги\n/checker — найти свободный слот\n/settings — настроить напоминания\n/edit — редактировать свои встречи\n/help — это сообщение",
			"en": "Lead Cat — your meetings assistant 🐾\n\n/menu — open the app\n/new — schedule a meeting\n/schedule — a colleague's schedule\n/checker — find a free slot\n/settings — configure reminders\n/edit — edit your meetings\n/help — this message",
			"kk": "Lead Cat — кездесу көмекшісі 🐾\n\n/menu — қосымшаны ашу\n/new — кездесу жоспарлау\n/schedule — әріптестің кестесі\n/checker — бос уақыт табу\n/settings — еске салуларды баптау\n/edit — кездесулеріңді өңдеу\n/help — осы хабарлама",
		},
	})
}
