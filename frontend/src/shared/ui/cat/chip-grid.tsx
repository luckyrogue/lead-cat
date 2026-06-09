import { cn } from "@/shared/lib/cn"

const COLS_CLASS = {
  2: "grid-cols-2",
  3: "grid-cols-3",
  4: "grid-cols-4",
} as const

type ChipOption<T> = { value: T; label: string; emoji?: string }

export function ChipGrid<T extends string | number>({
  options,
  value,
  onChange,
  cols = 2,
  layout = "grid",
  className,
}: {
  options: ChipOption<T>[]
  value: T
  onChange: (v: T) => void
  cols?: 2 | 3 | 4
  layout?: "grid" | "wrap"
  className?: string
}) {
  return (
    <div
      role="group"
      className={cn(
        layout === "wrap"
          ? "flex flex-wrap gap-2"
          : cn("grid gap-[9px]", COLS_CLASS[cols]),
        className
      )}
    >
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={String(o.value)}
            type="button"
            onClick={() => onChange(o.value)}
            aria-pressed={active}
            className={cn(
              "font-display text-miniapp-text flex cursor-pointer items-center gap-2 rounded-[14px] border-[1.5px] text-left text-sm font-bold leading-[1.1]",
              layout === "wrap"
                ? "rounded-[11px] px-3.5 py-[9px]"
                : "px-[13px] py-[13px]",
              active
                ? layout === "wrap"
                  ? "border-miniapp-accent bg-miniapp-accent-soft text-miniapp-accent"
                  : "border-miniapp-accent bg-miniapp-accent-soft"
                : "border-miniapp-border bg-miniapp-card"
            )}
          >
            {o.emoji && <span className="text-lg">{o.emoji}</span>}
            <span
              className={
                layout === "grid"
                  ? "font-display text-sm font-bold leading-[1.1]"
                  : undefined
              }
            >
              {o.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}
