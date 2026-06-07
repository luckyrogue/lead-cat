import { DEPARTMENTS } from "@/entities/employee/fixtures"
import { MEETING_TYPES, typeAccent } from "@/entities/meeting/constants"
import { useTmaApp } from "@/shared/tma/context"
import type { MeetingDraft } from "@/shared/tma/types"
import { Field } from "@/shared/ui/cat/primitives"
import { ChipGrid } from "./chip-grid"
import { WizardStepTitle } from "./wizard-step-title"

export function WizardStepWhat({
  draft,
  set,
}: {
  draft: MeetingDraft
  set: <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) => void
}) {
  const { dark, t } = useTmaApp()

  return (
    <div>
      <WizardStepTitle>🗂️ О чём встреча?</WizardStepTitle>
      <Field label={t("department")}>
        <ChipGrid
          value={draft.dept}
          onChange={(v) => set("dept", v)}
          options={DEPARTMENTS.map((d) => ({ value: d, label: d }))}
        />
      </Field>
      <div className="h-5" />
      <Field label={t("mType")}>
        <ChipGrid
          value={draft.type}
          onChange={(v) => set("type", v)}
          options={MEETING_TYPES.map((m) => ({
            value: m.key,
            label: m.label,
            emoji: typeAccent(m.key, dark).emoji,
          }))}
        />
      </Field>
    </div>
  )
}
