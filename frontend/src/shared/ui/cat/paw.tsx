import { cn } from "@/shared/lib/cn"

type PawTone = "hero" | "auto"

export function Paw({
  size = 22,
  tone = "hero",
  className,
}: {
  size?: number
  tone?: PawTone
  className?: string
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
      className={cn(
        tone === "auto"
          ? "text-[var(--tma-auto-paw)]"
          : "text-[var(--tma-hero-paw)]",
        className
      )}
      aria-hidden
    >
      <ellipse cx="12" cy="15.5" rx="5.4" ry="4.6" />
      <circle cx="5.6" cy="9.4" r="2.2" />
      <circle cx="10.1" cy="6.2" r="2.3" />
      <circle cx="14.6" cy="6" r="2.3" />
      <circle cx="18.6" cy="9.2" r="2.1" />
    </svg>
  )
}
