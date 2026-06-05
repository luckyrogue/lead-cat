import type { Employee } from "@/entities/employee/types"

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

export const DEPARTMENTS = [
  "Разработка",
  "Дизайн",
  "Продукт",
  "HR",
  "Маркетинг",
  "Аналитика",
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
