import type { Dict } from "~/shared/i18n/types"

export const ru: Dict = {
  common: {
    save: "Сохранить",
    cancel: "Отмена",
    delete: "Удалить",
    loading: "Загрузка…",
    retry: "Повторить",
    back: "Назад",
  },
  nav: {
    home: "Главная",
    meetings: "Встречи",
    checker: "Чекер",
    profile: "Профиль",
  },
  home: {
    appSubtitle: "Lead Cat",
    greeting: "Привет, {name}",
    bookMeeting: "Запланировать встречу",
    upcomingTitle: "Ближайшие",
    seeAll: "Все",
    errorLoad: "Не удалось загрузить встречи",
    emptyTitle: "Встреч пока нет",
    emptyHint: "Запланируй первую!",
  },
  states: {
    tryAgain: "Попробовать ещё раз",
  },
  auth: {
    signingIn: "Входим…",
    errorTitle: "Не удалось войти",
    errorMissingInitData: "Открой приложение из Telegram, чтобы продолжить.",
    errorGeneric: "Что-то пошло не так при авторизации.",
    notRegisteredTitle: "Ты ещё не зарегистрирован",
    notRegisteredHint:
      "Открой бота и нажми «Старт», чтобы получить доступ к встречам.",
    openBot: "Открыть бота",
    contactAdmin: "Свяжись с администратором для регистрации.",
  },
}
