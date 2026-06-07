import { cn } from "@/shared/lib/cn"
import type { I18nKey } from "@/shared/tma/i18n"

export function DurationPicker({
  value,
  onChange,
  options,
  t,
}: {
  value: number
  onChange: (mins: number) => void
  options: number[]
  t: (key: I18nKey) => string
}) {
  const label = (d: number) => {
    if (d < 60) return `${d} ${t("min")}`
    if (d === 60) return `1 ${t("hour")}`
    return `${d / 60} ${t("hour")}`
  }

  return (
    <div className="flex flex-wrap gap-2">
      {options.map((d) => {
        const active = d === value
        return (
          <button
            key={d}
            type="button"
            onClick={() => onChange(d)}
            className={cn(
              "font-display cursor-pointer rounded-[11px] border-[1.5px] px-3.5 py-[9px] text-sm font-bold",
              active
                ? "border-tma-accent bg-tma-accent-soft text-tma-accent"
                : "border-tma-border bg-tma-card text-tma-text"
            )}
          >
            {label(d)}
          </button>
        )
      })}
    </div>
  )
}
