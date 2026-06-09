import { cn } from "@/shared/lib/cn"

export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
}) {
  return (
    <div
      role="tablist"
      className="flex gap-0.5 rounded-[13px] bg-[var(--tma-segmented-track)] p-1"
    >
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            role="tab"
            aria-selected={active}
            className={cn(
              "font-display h-9 flex-1 cursor-pointer whitespace-nowrap rounded-[10px] border-none px-1.5 text-[13.5px] font-bold transition-all duration-[180ms] ease-out",
              active
                ? "bg-tma-card text-tma-text shadow-tma-sm"
                : "text-tma-muted bg-transparent shadow-none"
            )}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}
