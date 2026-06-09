import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"

export function CatCard({
  children,
  onClick,
  interactive = false,
  className,
}: {
  children: ReactNode
  onClick?: () => void
  interactive?: boolean
  className?: string
}) {
  return (
    <div
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={onClick ? (e) => e.key === "Enter" && onClick() : undefined}
      className={cn(
        "border-miniapp-border bg-miniapp-card shadow-miniapp-sm rounded-[20px] border p-4 transition-[transform,box-shadow] duration-150",
        onClick ? "cursor-pointer" : "cursor-default",
        interactive && "active:scale-[0.985]",
        className
      )}
    >
      {children}
    </div>
  )
}
