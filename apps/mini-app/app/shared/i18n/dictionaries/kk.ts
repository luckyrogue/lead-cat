import type { Dict } from "~/shared/i18n/types"

export const kk: Dict = {
  common: {
    save: "Сақтау",
    cancel: "Болдырмау",
    delete: "Жою",
    loading: "Жүктелуде…",
    retry: "Қайталау",
    back: "Артқа",
  },
  nav: {
    home: "Басты",
    meetings: "Кездесулер",
    checker: "Тексергіш",
    profile: "Профиль",
  },
  home: {
    appSubtitle: "Lead Cat",
    greeting: "Сәлем, {name}",
    bookMeeting: "Кездесу жоспарлау",
    upcomingTitle: "Жақындағы",
    seeAll: "Барлығы",
    errorLoad: "Кездесулерді жүктеу мүмкін болмады",
    emptyTitle: "Кездесулер жоқ",
    emptyHint: "Бірінші кездесуді жоспарла!",
  },
  states: {
    tryAgain: "Қайта көру",
  },
  auth: {
    signingIn: "Кіріп жатырмыз…",
    errorTitle: "Кіру мүмкін болмады",
    errorMissingInitData: "Жалғастыру үшін қолданбаны Telegram ішінен ашыңыз.",
    errorGeneric: "Авторизация кезінде қате орын алды.",
    notRegisteredTitle: "Әлі тіркелмегенсің",
    notRegisteredHint:
      "Ботты ашып, «Старт» батырмасын бас, сонда кездесулерге қол жетімді болады.",
    openBot: "Ботты ашу",
    contactAdmin: "Тіркелу үшін әкімшімен хабарлас.",
  },
}
