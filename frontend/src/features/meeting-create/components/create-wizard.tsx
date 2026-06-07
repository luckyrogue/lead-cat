import { useTmaApp } from "@/shared/tma/context"
import type { MeetingDraft } from "@/shared/tma/types"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"
import { useCreateWizard } from "../lib/use-create-wizard"
import { WIZARD_STEPS } from "../lib/wizard-constants"
import { WizardStepProgress } from "./wizard-step-progress"
import { WizardStepReview } from "./wizard-step-review"
import { WizardStepWhat } from "./wizard-step-what"
import { WizardStepWhen } from "./wizard-step-when"
import { WizardStepWho } from "./wizard-step-who"

export function CreateWizard({
  initial,
  onComplete,
}: {
  initial?: Partial<MeetingDraft>
  onComplete: (m: MeetingDraft & { end: string }) => void
}) {
  const { t } = useTmaApp()
  const wizard = useCreateWizard({ initial, onComplete })
  const {
    step,
    draft,
    set,
    endTime,
    canNext,
    go,
    conflictPeople,
    recurringBlocked,
    finalMeeting,
    pSearch,
    setPSearch,
  } = wizard

  return (
    <div className="flex h-full min-h-0 flex-col">
      <WizardStepProgress step={step} />

      <div className="lc-scroll flex-1 overflow-auto px-4 pb-4 pt-2.5">
        {WIZARD_STEPS[step] === "what" && (
          <WizardStepWhat draft={draft} set={set} />
        )}
        {WIZARD_STEPS[step] === "when" && (
          <WizardStepWhen draft={draft} set={set} endTime={endTime} />
        )}
        {WIZARD_STEPS[step] === "who" && (
          <WizardStepWho
            draft={draft}
            set={set}
            pSearch={pSearch}
            setPSearch={setPSearch}
          />
        )}
        {WIZARD_STEPS[step] === "review" && (
          <WizardStepReview
            draft={draft}
            endTime={endTime}
            finalMeeting={finalMeeting}
            conflictPeople={conflictPeople}
            recurringBlocked={recurringBlocked}
          />
        )}
      </div>

      <div className="flex shrink-0 gap-2.5 border-t border-tma-border bg-tma-tg-bar px-4 pt-3 pb-[max(12px,var(--tma-safe-bottom,0px))]">
        {step > 0 && (
          <CatBtn
            variant="outline"
            onClick={() => go(-1)}
            icon={
              <CatIcon
                name="chevL"
                size={18}
                className="text-tma-text"
                sw={2.2}
              />
            }
          >
            {t("back")}
          </CatBtn>
        )}
        <CatBtn
          variant="primary"
          full
          disabled={
            !canNext || (WIZARD_STEPS[step] === "review" && recurringBlocked)
          }
          onClick={() => go(1)}
        >
          {step === WIZARD_STEPS.length - 1
            ? conflictPeople.length
              ? `🐾 ${t("proceed")}`
              : `🐾 ${t("confirmCreate")}`
            : t("next")}
        </CatBtn>
      </div>
    </div>
  )
}
