import { useState } from "react"
import { TMA_NOW, WEEKDAYS } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

function calNav(p: ReturnType<typeof useTmaApp>) {
  return {
    width: 32,
    height: 32,
    borderRadius: 10,
    border: "none",
    background: p.cardAlt,
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  } as const
}

export function MiniCalendar({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const p = useTmaApp()
  const [view, setView] = useState(() => {
    const [y, m] = (value || TMA_NOW).split("-").map(Number)
    return { y, m: m - 1 }
  })
  const monthNames = [
    "Январь",
    "Февраль",
    "Март",
    "Апрель",
    "Май",
    "Июнь",
    "Июль",
    "Август",
    "Сентябрь",
    "Октябрь",
    "Ноябрь",
    "Декабрь",
  ]
  const first = new Date(view.y, view.m, 1)
  const startDow = (first.getDay() + 6) % 7
  const days = new Date(view.y, view.m + 1, 0).getDate()
  const cells: (number | null)[] = []
  for (let i = 0; i < startDow; i++) cells.push(null)
  for (let d = 1; d <= days; d++) cells.push(d)
  const iso = (d: number) =>
    `${view.y}-${String(view.m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`
  const nav = (dir: number) =>
    setView((v) => {
      let m = v.m + dir
      let y = v.y
      if (m < 0) {
        m = 11
        y--
      }
      if (m > 11) {
        m = 0
        y++
      }
      return { y, m }
    })

  return (
    <div
      style={{
        background: p.card,
        borderRadius: 18,
        border: `1px solid ${p.border}`,
        padding: 14,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 12,
        }}
      >
        <button type="button" onClick={() => nav(-1)} style={calNav(p)}>
          <CatIcon name="chevL" size={18} color={p.text} sw={2.2} />
        </button>
        <span
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 16,
            color: p.text,
          }}
        >
          {monthNames[view.m]} {view.y}
        </span>
        <button type="button" onClick={() => nav(1)} style={calNav(p)}>
          <CatIcon name="chevR" size={18} color={p.text} sw={2.2} />
        </button>
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7,1fr)",
          gap: 4,
          marginBottom: 6,
        }}
      >
        {WEEKDAYS.map((w) => (
          <div
            key={w}
            style={{
              textAlign: "center",
              fontSize: 11,
              fontWeight: 700,
              color: p.faint,
            }}
          >
            {w}
          </div>
        ))}
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7,1fr)",
          gap: 4,
        }}
      >
        {cells.map((d, i) => {
          if (!d) return <div key={i} />
          const dIso = iso(d)
          const active = dIso === value
          const isPast = dIso < TMA_NOW
          const isToday = dIso === TMA_NOW
          return (
            <button
              key={i}
              type="button"
              onClick={() => onChange(dIso)}
              style={{
                aspectRatio: "1",
                borderRadius: 11,
                border: "none",
                cursor: "pointer",
                background: active ? p.accent : "transparent",
                color: active ? p.accentText : isPast ? p.faint : p.text,
                fontWeight: active || isToday ? 800 : 600,
                fontSize: 14,
                fontFamily: "var(--font-display)",
                position: "relative",
              }}
            >
              {d}
              {isToday && !active && (
                <span
                  style={{
                    position: "absolute",
                    bottom: 5,
                    left: "50%",
                    transform: "translateX(-50%)",
                    width: 4,
                    height: 4,
                    borderRadius: 2,
                    background: p.accent,
                  }}
                />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
