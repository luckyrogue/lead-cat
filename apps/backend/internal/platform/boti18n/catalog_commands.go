package boti18n

func init() {
	register(map[string]map[string]string{
		"cmd.menu":            {"ru": "Открыть приложение", "en": "Open the app", "kk": "Қосымшаны ашу"},
		"cmd.new":             {"ru": "Запланировать встречу", "en": "Schedule a meeting", "kk": "Кездесу жоспарлау"},
		"cmd.schedule":        {"ru": "Расписание коллеги", "en": "Colleague's schedule", "kk": "Әріптестің кестесі"},
		"cmd.checker":         {"ru": "Найти свободный слот", "en": "Find a free slot", "kk": "Бос уақыт табу"},
		"cmd.settings":        {"ru": "Напоминания", "en": "Reminders", "kk": "Еске салулар"},
		"cmd.edit":            {"ru": "Редактировать встречи", "en": "Edit meetings", "kk": "Кездесулерді өңдеу"},
		"cmd.help":            {"ru": "Помощь", "en": "Help", "kk": "Көмек"},
		"cmd.chatid":          {"ru": "Логово id: %[1]d — скопируй в админку 🐾", "en": "Den id: %[1]d — copy to admin panel 🐾", "kk": "Үй id: %[1]d — әкімші панеліне көшір 🐾"},
		"cmd.register_first":  {"ru": "Сначала зарегистрируйся: /start", "en": "Register first: /start", "kk": "Алдымен тіркел: /start"},
		"cmd.menu_open":       {"ru": "Открой Lead Cat 🐾", "en": "Open Lead Cat 🐾", "kk": "Lead Cat-ты аш 🐾"},
		"cmd.new_prompt":      {"ru": "Запланируй встречу 🐾", "en": "Schedule a meeting 🐾", "kk": "Кездесу жоспарла 🐾"},
		"cmd.btn_new_meeting": {"ru": "Новая встреча", "en": "New meeting", "kk": "Жаңа кездесу"},
		"cmd.admin_yes":       {"ru": "Ты администратор 🐾 Настройки — в приложении.", "en": "You are an administrator 🐾 Settings — in the app.", "kk": "Сен әкімшісің 🐾 Баптаулар — қосымшада."},
		"cmd.admin_no":        {"ru": "Ты не администратор 🐾", "en": "You are not an administrator 🐾", "kk": "Сен әкімші емессің 🐾"},
		"cmd.help.text": {
			"ru": "Lead Cat — помощник по встречам 🐾\n\n/menu — открыть приложение\n/new — запланировать встречу\n/schedule — расписание коллеги\n/checker — найти свободный слот\n/settings — настроить напоминания\n/edit — редактировать свои встречи\n/help — это сообщение",
			"en": "Lead Cat — your meetings assistant 🐾\n\n/menu — open the app\n/new — schedule a meeting\n/schedule — a colleague's schedule\n/checker — find a free slot\n/settings — configure reminders\n/edit — edit your meetings\n/help — this message",
			"kk": "Lead Cat — кездесу көмекшісі 🐾\n\n/menu — қосымшаны ашу\n/new — кездесу жоспарлау\n/schedule — әріптестің кестесі\n/checker — бос уақыт табу\n/settings — еске салуларды баптау\n/edit — кездесулеріңді өңдеу\n/help — осы хабарлама",
		},
	})
}
