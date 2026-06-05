import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"

export function WizardStepTitle({ children }: { children: ReactNode }) {
  const p = useTmaApp()
  return (
    <h2
      style={{
        margin: "0 0 18px",
        fontFamily: "var(--font-display)",
        fontWeight: 800,
        fontSize: 23,
        color: p.text,
      }}
    >
      {children}
    </h2>
  )
}
