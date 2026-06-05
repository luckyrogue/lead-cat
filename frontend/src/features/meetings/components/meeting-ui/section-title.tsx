import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
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
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        margin: "0 4px 10px",
      }}
    >
      <h3
        style={{
          margin: 0,
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 18,
          color: p.text,
        }}
      >
        {children}
      </h3>
      {action && onAction && (
        <button
          type="button"
          onClick={onAction}
          style={{
            background: "none",
            border: "none",
            color: p.accent,
            fontWeight: 700,
            fontSize: 13.5,
            cursor: "pointer",
            fontFamily: "var(--font-display)",
            display: "flex",
            alignItems: "center",
            gap: 2,
          }}
        >
          {action}
          <CatIcon name="chevR" size={14} color={p.accent} sw={2.4} />
        </button>
      )}
    </div>
  )
}
