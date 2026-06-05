import { useMemo } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { EMPLOYEES } from "@/shared/tma/mock-data"
import type { MeetingDraft } from "@/shared/tma/types"
import { EmployeePicker } from "@/components/employee-picker"
import { Avatar, Field, inputStyle } from "@/shared/ui/cat/primitives"
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
  const p = useTmaApp()
  const t = p.t

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
        <div
          style={{
            ...inputStyle(p),
            display: "flex",
            alignItems: "center",
            gap: 10,
            height: 54,
          }}
        >
          <Avatar name={draft.host} size={32} />
          <span
            style={{
              fontWeight: 700,
              fontFamily: "var(--font-display)",
              color: p.text,
            }}
          >
            {draft.host}
          </span>
          <span
            style={{
              marginLeft: "auto",
              fontSize: 11,
              fontWeight: 700,
              color: p.accent,
              background: p.accentSoft,
              padding: "3px 8px",
              borderRadius: 999,
            }}
          >
            я
          </span>
        </div>
      </Field>
      <div style={{ height: 18 }} />
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
      <div style={{ height: 18 }} />
      <Field label={t("descr")}>
        <textarea
          value={draft.desc}
          onChange={(e) => set("desc", e.target.value)}
          rows={3}
          placeholder="—"
          style={{
            ...inputStyle(p),
            height: "auto",
            padding: "12px 14px",
            resize: "none",
          }}
        />
      </Field>
    </div>
  )
}
