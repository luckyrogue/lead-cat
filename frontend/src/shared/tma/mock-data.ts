import type { Meeting } from "@/entities/meeting/types"
import type { FreeSlot } from "@/entities/meeting/types"
import type { Scenario } from "@/entities/scenario/types"
import {
  DEPARTMENTS,
  EMPLOYEES,
  emailsToPeople,
} from "@/entities/employee/fixtures"
import {
  MEETING_TYPES,
  RECURRENCE,
} from "@/entities/meeting/constants"
import { INITIAL_SCENARIOS } from "@/features/auto/fixtures"

export {
  DEPARTMENTS,
  EMPLOYEES,
  emailsToPeople,
  MEETING_TYPES,
  RECURRENCE,
  INITIAL_SCENARIOS,
}

export const ME = EMPLOYEES[0]

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

export type { Meeting, FreeSlot, Scenario }
