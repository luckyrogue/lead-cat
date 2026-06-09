import type { ReactNode } from "react"
import { useMiniApp } from "@/shared/miniapp/context"
import { paletteCssVars } from "@/shared/miniapp/palette-to-css-vars"

export function MiniAppFrame({ children }: { children: ReactNode }) {
  const p = useMiniApp()
  return (
    <div
      className="miniapp-frame bg-miniapp-bg bg-miniapp-pattern"
      style={paletteCssVars(p)}
    >
      {children}
    </div>
  )
}
