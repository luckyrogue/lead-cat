import type { Scenario } from "@/entities/scenario/types"

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
