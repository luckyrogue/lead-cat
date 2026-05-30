export const TMA_NOW = "2026-06-01"

export const WEEKDAYS = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"] as const

export const TYPE_COLORS: Record<string, { h: number; emoji: string }> = {
  planning: { h: 45, emoji: "🗒️" },
  weekly: { h: 180, emoji: "📆" },
  oneonone: { h: 300, emoji: "🤝" },
  retro: { h: 350, emoji: "🔁" },
  demo: { h: 150, emoji: "🎬" },
  interview: { h: 255, emoji: "💼" },
  onboarding: { h: 95, emoji: "🌱" },
  brainstorm: { h: 25, emoji: "💡" },
  strategy: { h: 280, emoji: "🧭" },
  other: { h: 0, emoji: "📌" },
}

export function typeAccent(key: string, dark: boolean) {
  const c = TYPE_COLORS[key] ?? TYPE_COLORS.other
  return {
    solid: `oklch(0.64 0.15 ${c.h})`,
    text: dark ? `oklch(0.80 0.13 ${c.h})` : `oklch(0.50 0.16 ${c.h})`,
    soft: dark ? `oklch(0.34 0.07 ${c.h})` : `oklch(0.95 0.045 ${c.h})`,
    emoji: c.emoji,
  }
}
