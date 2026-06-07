import { useState } from "react"
import { TMA_NOW, WEEKDAYS } from "@/entities/meeting/constants"
import { cn } from "@/shared/lib/cn"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function MiniCalendar({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
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

  const navBtnClass =
    "flex size-8 cursor-pointer items-center justify-center rounded-[10px] border-none bg-tma-card-alt"

  return (
    <div className="rounded-[18px] border border-tma-border bg-tma-card p-3.5">
      <div className="mb-3 flex items-center justify-between">
        <button type="button" onClick={() => nav(-1)} className={navBtnClass}>
          <CatIcon
            name="chevL"
            size={18}
            className="text-tma-text"
            sw={2.2}
          />
        </button>
        <span className="font-display text-base font-extrabold text-tma-text">
          {monthNames[view.m]} {view.y}
        </span>
        <button type="button" onClick={() => nav(1)} className={navBtnClass}>
          <CatIcon
            name="chevR"
            size={18}
            className="text-tma-text"
            sw={2.2}
          />
        </button>
      </div>
      <div className="mb-1.5 grid grid-cols-7 gap-1">
        {WEEKDAYS.map((w) => (
          <div
            key={w}
            className="text-center text-[11px] font-bold text-tma-faint"
          >
            {w}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
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
              className={cn(
                "font-display relative aspect-square cursor-pointer rounded-[11px] border-none text-sm",
                active
                  ? "bg-tma-accent font-extrabold text-tma-accent-text"
                  : cn(
                      "bg-transparent",
                      isPast
                        ? "font-semibold text-tma-faint"
                        : "font-semibold text-tma-text",
                      isToday && "font-extrabold"
                    )
              )}
            >
              {d}
              {isToday && !active && (
                <span className="absolute bottom-[5px] left-1/2 size-1 -translate-x-1/2 rounded-sm bg-tma-accent" />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
