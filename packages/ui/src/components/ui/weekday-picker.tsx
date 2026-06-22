import { WEEKDAYS, toggleDay } from "@leadcat/types"

import { cn } from "../../lib/cn"

type WeekdayPickerProps = {
  value: number[]
  onChange: (value: number[]) => void
  label: (day: number) => string
}

function WeekdayPicker({ value, onChange, label }: WeekdayPickerProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {WEEKDAYS.map((day) => {
        const active = value.includes(day.value)
        return (
          <button
            key={day.value}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(toggleDay(value, day.value))}
            className={cn(
              "rounded-[calc(var(--radius)*0.75)] border px-3 py-1.5 text-sm transition",
              active
                ? "border-primary bg-primary font-medium text-primary-foreground"
                : "border-border bg-background text-foreground hover:bg-muted"
            )}
          >
            {label(day.value)}
          </button>
        )
      })}
    </div>
  )
}

export { WeekdayPicker }
