import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"

export function TmaFrame({ children }: { children: ReactNode }) {
  const p = useTmaApp()
  return (
    <div
      className="tma-frame"
      style={{
        background: p.bg,
        backgroundImage: `radial-gradient(${p.pattern} 1.4px, transparent 1.4px)`,
        backgroundSize: "22px 22px",
      }}
    >
      {children}
    </div>
  )
}
