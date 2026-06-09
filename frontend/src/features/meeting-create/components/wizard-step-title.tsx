import type { ReactNode } from "react"

export function WizardStepTitle({ children }: { children: ReactNode }) {
  return <h2 className="miniapp-heading mb-[18px] text-[23px]">{children}</h2>
}
