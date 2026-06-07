import { cn } from "@/shared/lib/cn"

export function CatToggle({
  on,
  onChange,
}: {
  on: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(!on)}
      className={cn(
        "relative h-[30px] w-[50px] shrink-0 cursor-pointer rounded-full border-none p-0 transition-colors duration-[220ms] ease-out",
        on ? "bg-tma-accent" : "bg-[var(--tma-toggle-off)]"
      )}
    >
      <span
        className={cn(
          "absolute top-[3px] size-6 rounded-full bg-white shadow-[0_1px_4px_rgba(0,0,0,0.25)] transition-[left] duration-[220ms] ease-[cubic-bezier(.34,1.56,.64,1)]",
          on ? "left-[23px]" : "left-[3px]"
        )}
      />
    </button>
  )
}
