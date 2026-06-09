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
        <div className="font-display text-miniapp-muted mb-[7px] text-[13px] font-bold">
          {label}
        </div>
      )}
      {children}
    </label>
  )
}
