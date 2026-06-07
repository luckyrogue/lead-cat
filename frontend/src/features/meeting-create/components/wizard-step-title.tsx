import type { ReactNode } from "react"

export function WizardStepTitle({ children }: { children: ReactNode }) {
  return (
    <h2 className="tma-heading mb-[18px] text-[23px]">{children}</h2>
  )
}
