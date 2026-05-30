import type { Employee, FreeSlot, Meeting, Scenario } from "./types"

export const EMPLOYEES: Employee[] = [
  {
    id: "u1",
    name: "Алия Жумабекова",
    email: "a.zhumabekova@company.kz",
    dept: "Разработка",
    tg: true,
    role: "admin",
  },
  {
    id: "u2",
    name: "Иван Петров",
    email: "i.petrov@company.kz",
    dept: "Разработка",
    tg: true,
  },
  {
    id: "u3",
    name: "Дамир Сейтказы",
    email: "d.seitkazy@company.kz",
    dept: "Дизайн",
    tg: true,
  },
  {
    id: "u4",
    name: "Мария Соколова",
    email: "m.sokolova@company.kz",
    dept: "HR",
    tg: false,
  },
  {
    id: "u5",
    name: "Тимур Абдрахманов",
    email: "t.abd@company.kz",
    dept: "Продукт",
    tg: true,
  },
  {
    id: "u6",
    name: "Айгерим Нурлан",
    email: "a.nurlan@company.kz",
    dept: "Маркетинг",
    tg: true,
  },
  {
    id: "u7",
    name: "Олег Васильев",
    email: "o.vasiliev@company.kz",
    dept: "Разработка",
    tg: false,
  },
  {
    id: "u8",
    name: "Сабина Ахмет",
    email: "s.akhmet@company.kz",
    dept: "Продукт",
    tg: true,
  },
  {
    id: "u9",
    name: "Ержан Касымов",
    email: "e.kasymov@company.kz",
    dept: "Разработка",
    tg: true,
  },
]

export const ME = EMPLOYEES[0]

export const DEPARTMENTS = [
  "Разработка",
  "Дизайн",
  "Продукт",
  "HR",
  "Маркетинг",
  "Аналитика",
]

export const MEETING_TYPES = [
  { key: "planning", label: "Планёрка" },
  { key: "weekly", label: "Еженедельная" },
  { key: "oneonone", label: "1:1" },
  { key: "retro", label: "Ретроспектива" },
  { key: "demo", label: "Демо / Презентация" },
  { key: "interview", label: "Интервью" },
  { key: "onboarding", label: "Онбординг" },
  { key: "brainstorm", label: "Брейншторм" },
  { key: "strategy", label: "Стратегическая сессия" },
  { key: "other", label: "Другое" },
]

export const RECURRENCE = [
  { key: "once", label: "Однократно", short: "" },
  { key: "daily", label: "Ежедневно", short: "Ежедневно" },
  { key: "weekly", label: "Еженедельно", short: "Еженедельно" },
  { key: "custom", label: "Выбранные дни", short: "По дням" },
  { key: "monthly", label: "Ежемесячно", short: "Ежемесячно" },
]

export function emailsToPeople(emails: string[]): Employee[] {
  return emails.map(
    (e) =>
      EMPLOYEES.find((x) => x.email === e) ?? {
        id: e,
        name: e,
        email: e,
        dept: "",
        tg: false,
      }
  )
}

export const INITIAL_MEETINGS: Meeting[] = [
  {
    id: "m1",
    type: "planning",
    dept: "Разработка",
    host: "Алия Жумабекова",
    date: "2026-06-01",
    start: "10:00",
    end: "10:30",
    rec: "weekly",
    recDays: [1],
    organizer: "a.zhumabekova@company.kz",
    participants: [
      "i.petrov@company.kz",
      "o.vasiliev@company.kz",
      "e.kasymov@company.kz",
    ],
    desc: "Синк по спринту, разбор блокеров.",
  },
  {
    id: "m2",
    type: "oneonone",
    dept: "Продукт",
    host: "Тимур Абдрахманов",
    date: "2026-06-01",
    start: "14:00",
    end: "14:45",
    rec: "once",
    organizer: "t.abd@company.kz",
    participants: ["s.akhmet@company.kz"],
    desc: "Карьерный разговор.",
  },
  {
    id: "m3",
    type: "demo",
    dept: "Продукт",
    host: "Сабина Ахмет",
    date: "2026-06-02",
    start: "16:00",
    end: "17:00",
    rec: "once",
    organizer: "s.akhmet@company.kz",
    participants: [
      "a.zhumabekova@company.kz",
      "t.abd@company.kz",
      "a.nurlan@company.kz",
      "d.seitkazy@company.kz",
    ],
    desc: "Демо новой фичи онбординга для стейкхолдеров.",
  },
  {
    id: "m4",
    type: "retro",
    dept: "Разработка",
    host: "Алия Жумабекова",
    date: "2026-06-03",
    start: "11:00",
    end: "12:00",
    rec: "weekly",
    recDays: [3],
    organizer: "a.zhumabekova@company.kz",
    participants: [
      "i.petrov@company.kz",
      "o.vasiliev@company.kz",
      "e.kasymov@company.kz",
      "d.seitkazy@company.kz",
    ],
    desc: "Ретро спринта №14.",
  },
  {
    id: "m5",
    type: "weekly",
    dept: "Маркетинг",
    host: "Айгерим Нурлан",
    date: "2026-05-28",
    start: "09:30",
    end: "10:00",
    rec: "weekly",
    recDays: [3],
    organizer: "a.nurlan@company.kz",
    participants: ["a.zhumabekova@company.kz"],
    desc: "Контент-план на неделю.",
  },
  {
    id: "m6",
    type: "interview",
    dept: "HR",
    host: "Мария Соколова",
    date: "2026-05-26",
    start: "15:00",
    end: "15:45",
    rec: "once",
    organizer: "m.sokolova@company.kz",
    participants: ["a.zhumabekova@company.kz"],
    desc: "Финальное интервью кандидата (Go).",
  },
]

export const INITIAL_SCENARIOS: Scenario[] = [
  {
    id: "s1",
    name: "Утренний созвон",
    enabled: true,
    trigger: { hour: 10, minute: 15, days: [1, 3, 5] },
    actions: ["message", "cat_photo"],
    note: "За 15 минут пингует команду в чат и кидает котика для настроения.",
  },
  {
    id: "s2",
    name: "Пора коммитить",
    enabled: true,
    trigger: { hour: 18, minute: 30, days: [1, 2, 3, 4, 5] },
    actions: ["message"],
    note: "Напоминает залить код перед концом дня. «Пора коммитить! 🐾»",
  },
  {
    id: "s3",
    name: "Отчёт по коммитам",
    enabled: false,
    trigger: { hour: 18, minute: 35, days: [1, 2, 3, 4, 5] },
    actions: ["commits_report"],
    note: "Собирает дайджест коммитов за день и шлёт в чат команды.",
  },
]

export const FREE_SLOTS: FreeSlot[] = [
  {
    day: "Пн, 01.06",
    iso: "2026-06-01",
    start: "11:00",
    end: "12:30",
    mins: 90,
  },
  {
    day: "Пн, 01.06",
    iso: "2026-06-01",
    start: "15:00",
    end: "17:00",
    mins: 120,
  },
  {
    day: "Вт, 02.06",
    iso: "2026-06-02",
    start: "09:00",
    end: "10:00",
    mins: 60,
  },
  {
    day: "Ср, 03.06",
    iso: "2026-06-03",
    start: "13:30",
    end: "15:30",
    mins: 120,
  },
]
