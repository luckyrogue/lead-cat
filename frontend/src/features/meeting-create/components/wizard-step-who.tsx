import { useMemo } from "react"
import { useMiniApp } from "@/shared/miniapp/context"
import { EMPLOYEES } from "@/entities/employee/fixtures"
import type { MeetingDraft } from "@/shared/miniapp/types"
import { EmployeePicker } from "@/components/employee-picker"
import { cn } from "@/shared/lib/cn"
import { Avatar, Field } from "@/shared/ui/cat/primitives"
import { WizardStepTitle } from "./wizard-step-title"

export function WizardStepWho({
  draft,
  set,
  pSearch,
  setPSearch,
}: {
  draft: MeetingDraft
  set: <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) => void
  pSearch: string
  setPSearch: (q: string) => void
}) {
  const { t } = useMiniApp()

  const matches = useMemo(
    () =>
      EMPLOYEES.filter((e) => {
        const q = pSearch.toLowerCase()
        return (
          q &&
          (e.name.toLowerCase().includes(q) ||
            e.email.toLowerCase().includes(q)) &&
          !draft.participants.find((x) => x.email === e.email)
        )
      }).slice(0, 5),
    [pSearch, draft.participants]
  )

  return (
    <div>
      <WizardStepTitle>🧑‍🤝‍🧑 Кто участвует?</WizardStepTitle>
      <Field label={t("host")}>
        <div className="miniapp-input flex h-[54px] items-center gap-2.5">
          <Avatar name={draft.host} size={32} />
          <span className="font-display text-miniapp-text font-bold">
            {draft.host}
          </span>
          <span className="bg-miniapp-accent-soft text-miniapp-accent ml-auto rounded-full px-2 py-[3px] text-[11px] font-bold">
            я
          </span>
        </div>
      </Field>
      <div className="h-[18px]" />
      <Field label={t("addPeople")}>
        <EmployeePicker
          value={draft.participants}
          onChange={(next) => set("participants", next)}
          search={pSearch}
          onSearchChange={setPSearch}
          matches={matches}
          searchPlaceholder={t("searchPeople")}
          showEmail
        />
      </Field>
      <div className="h-[18px]" />
      <Field label={t("descr")}>
        <textarea
          value={draft.desc}
          onChange={(e) => set("desc", e.target.value)}
          rows={3}
          placeholder="—"
          className={cn("miniapp-input h-auto resize-none py-3")}
        />
      </Field>
    </div>
  )
}
