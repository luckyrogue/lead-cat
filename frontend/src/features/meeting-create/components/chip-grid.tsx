import { cn } from "@/shared/lib/cn"

const COLS_CLASS = {
  2: "grid-cols-2",
  3: "grid-cols-3",
  4: "grid-cols-4",
} as const

export function ChipGrid<T extends string>({
  options,
  value,
  onChange,
  cols = 2,
}: {
  options: { value: T; label: string; emoji?: string }[]
  value: T
  onChange: (v: T) => void
  cols?: 2 | 3 | 4
}) {
  return (
    <div className={cn("grid gap-[9px]", COLS_CLASS[cols])}>
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            className={cn(
              "flex cursor-pointer items-center gap-2 rounded-[14px] border-[1.5px] px-[13px] py-[13px] text-left text-tma-text",
              active
                ? "border-tma-accent bg-tma-accent-soft"
                : "border-tma-border bg-tma-card"
            )}
          >
            {o.emoji && <span className="text-lg">{o.emoji}</span>}
            <span className="font-display text-sm font-bold leading-[1.1]">
              {o.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}
