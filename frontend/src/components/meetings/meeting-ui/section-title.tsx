import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function SectionTitle({
  children,
  action,
  onAction,
}: {
  children: ReactNode
  action?: string | null
  onAction?: () => void
}) {
  return (
    <div className="mb-2.5 flex items-center justify-between px-1">
      <h3 className="miniapp-heading m-0 text-lg">{children}</h3>
      {action && onAction && (
        <button
          type="button"
          onClick={onAction}
          className={cn(
            "flex cursor-pointer items-center gap-0.5 border-none bg-transparent",
            "font-display text-miniapp-accent text-[13.5px] font-bold"
          )}
        >
          {action}
          <CatIcon
            name="chevR"
            size={14}
            className="text-miniapp-accent"
            sw={2.4}
          />
        </button>
      )}
    </div>
  )
}
