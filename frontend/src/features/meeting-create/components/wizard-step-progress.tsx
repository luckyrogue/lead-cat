import { cn } from "@/shared/lib/cn"
import { Paw } from "@/shared/ui/cat/paw"
import { WIZARD_STEPS } from "../lib/wizard-constants"

export function WizardStepProgress({ step }: { step: number }) {
  return (
    <div className="flex items-center gap-[7px] px-4 pb-1.5 pt-3.5">
      {WIZARD_STEPS.map((s, i) => (
        <div key={s} className="flex flex-1 items-center gap-[7px]">
          <span className="shrink-0 transition-all duration-[250ms]">
            <Paw
              size={i <= step ? 20 : 16}
              className={i <= step ? "text-tma-accent" : "text-tma-border"}
            />
          </span>
          {i < WIZARD_STEPS.length - 1 && (
            <div
              className={cn(
                "h-[3px] flex-1 rounded-sm transition-[background] duration-[250ms]",
                i < step ? "bg-tma-accent" : "bg-tma-border"
              )}
            />
          )}
        </div>
      ))}
    </div>
  )
}
