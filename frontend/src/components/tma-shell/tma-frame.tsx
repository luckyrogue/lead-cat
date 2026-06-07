import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { paletteCssVars } from "@/shared/tma/palette-to-css-vars"

export function TmaFrame({ children }: { children: ReactNode }) {
  const p = useTmaApp()
  return (
    <div
      className="tma-frame bg-tma-bg bg-tma-pattern"
      style={paletteCssVars(p)}
    >
      {children}
    </div>
  )
}
