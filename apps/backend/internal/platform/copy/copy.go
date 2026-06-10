package copy

const (
	AppName = "Lead Cat"
)

func APIError(code string) string {
	switch code {
	case "unauthorized":
		return "Кот не узнал тебя. Войди снова 🐾"
	case "forbidden":
		return "Сюда только для своих котиков"
	case "not_found":
		return "Логово не найдено"
	default:
		return "Кот уронил сервер. Попробуй ещё раз"
	}
}
