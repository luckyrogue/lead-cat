import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"

export function Field({
  label,
  children,
  className,
}: {
  label?: string
  children: ReactNode
  className?: string
}) {
  return (
    <label className={cn("block", className)}>
      {label && (
        <div className="font-display mb-[7px] text-[13px] font-bold text-tma-muted">
          {label}
        </div>
      )}
      {children}
    </label>
  )
}
