import { useTmaApp } from "@/shared/tma/context"
import { Paw } from "@/shared/ui/cat/paw"
import { WIZARD_STEPS } from "../lib/wizard-constants"

export function WizardStepProgress({ step }: { step: number }) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        gap: 7,
        padding: "14px 16px 6px",
        alignItems: "center",
      }}
    >
      {WIZARD_STEPS.map((s, i) => (
        <div
          key={s}
          style={{ flex: 1, display: "flex", alignItems: "center", gap: 7 }}
        >
          <Paw
            size={i <= step ? 20 : 16}
            color={i <= step ? p.accent : p.border}
            style={{ transition: "all .25s", flexShrink: 0 }}
          />
          {i < WIZARD_STEPS.length - 1 && (
            <div
              style={{
                flex: 1,
                height: 3,
                borderRadius: 2,
                background: i < step ? p.accent : p.border,
                transition: "background .25s",
              }}
            />
          )}
        </div>
      ))}
    </div>
  )
}
