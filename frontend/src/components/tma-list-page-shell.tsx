import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { Spinner } from "@/components/ui/spinner"

type TmaListPageShellProps = {
  title: string
  actions?: ReactNode
  isLoading?: boolean
  filters?: ReactNode
  empty?: boolean
  emptyState: ReactNode
  children: ReactNode
  detail?: ReactNode
}

export function TmaListPageShell({
  title,
  actions,
  isLoading,
  filters,
  empty,
  emptyState,
  children,
  detail,
}: TmaListPageShellProps) {
  const p = useTmaApp()

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          margin: "2px 4px 14px",
          gap: 12,
        }}
      >
        <h2
          style={{
            margin: 0,
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 26,
            color: p.text,
          }}
        >
          {title}
        </h2>
        {actions}
      </div>
      {filters}
      {isLoading ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
          <Spinner />
        </div>
      ) : empty ? (
        <div
          style={{
            background: p.card,
            borderRadius: 20,
            border: `1px solid ${p.border}`,
            overflow: "hidden",
          }}
        >
          {emptyState}
        </div>
      ) : (
        children
      )}
      {detail}
    </div>
  )
}
