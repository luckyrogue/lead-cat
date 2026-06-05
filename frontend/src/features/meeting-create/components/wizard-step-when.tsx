import { WEEKDAYS } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import { RECURRENCE } from "@/shared/tma/mock-data"
import type { MeetingDraft } from "@/shared/tma/types"
import { DurationPicker } from "@/components/duration-picker"
import { Field } from "@/shared/ui/cat/primitives"
import {
  DURATION_OPTIONS,
  TIME_SLOTS,
} from "../lib/wizard-constants"
import { ChipGrid } from "./chip-grid"
import { MiniCalendar } from "./mini-calendar"
import { WizardStepTitle } from "./wizard-step-title"

export function WizardStepWhen({
  draft,
  set,
  endTime,
}: {
  draft: MeetingDraft
  set: <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) => void
  endTime: string
}) {
  const p = useTmaApp()
  const t = p.t

  return (
    <div>
      <WizardStepTitle>🗓️ Когда встречаемся?</WizardStepTitle>
      <Field label={t("dateT")}>
        <MiniCalendar value={draft.date} onChange={(v) => set("date", v)} />
      </Field>
      <div style={{ height: 18 }} />
      <Field
        label={`${t("timeT")} — ${t("from")} ${draft.start}, ${t("to")} ${endTime}`}
      >
        <div
          className="lc-scroll"
          style={{
            display: "flex",
            gap: 8,
            overflowX: "auto",
            paddingBottom: 4,
          }}
        >
          {TIME_SLOTS.map((tm) => {
            const active = tm === draft.start
            return (
              <button
                key={tm}
                type="button"
                onClick={() => set("start", tm)}
                style={{
                  flexShrink: 0,
                  padding: "9px 13px",
                  borderRadius: 11,
                  border: `1.5px solid ${active ? p.accent : p.border}`,
                  background: active ? p.accent : p.card,
                  color: active ? p.accentText : p.text,
                  fontWeight: 700,
                  fontSize: 14,
                  fontFamily: "var(--font-display)",
                  cursor: "pointer",
                }}
              >
                {tm}
              </button>
            )
          })}
        </div>
      </Field>
      <div style={{ height: 16 }} />
      <Field label={t("duration")}>
        <DurationPicker
          value={draft.dur}
          onChange={(v) => set("dur", v)}
          options={[...DURATION_OPTIONS]}
          t={t}
        />
      </Field>
      <div style={{ height: 18 }} />
      <Field label={t("recur")}>
        <ChipGrid
          value={draft.rec}
          onChange={(v) => set("rec", v)}
          cols={2}
          options={RECURRENCE.map((r) => ({
            value: r.key,
            label: r.label,
          }))}
        />
        {draft.rec === "custom" && (
          <div style={{ display: "flex", gap: 7, marginTop: 11 }}>
            {WEEKDAYS.map((w, i) => {
              const on = draft.recDays.includes(i + 1)
              return (
                <button
                  key={w}
                  type="button"
                  onClick={() =>
                    set(
                      "recDays",
                      on
                        ? draft.recDays.filter((x) => x !== i + 1)
                        : [...draft.recDays, i + 1]
                    )
                  }
                  style={{
                    flex: 1,
                    aspectRatio: "1",
                    borderRadius: 11,
                    border: `1.5px solid ${on ? p.accent : p.border}`,
                    background: on ? p.accent : p.card,
                    color: on ? p.accentText : p.muted,
                    fontWeight: 800,
                    fontSize: 12.5,
                    fontFamily: "var(--font-display)",
                    cursor: "pointer",
                  }}
                >
                  {w}
                </button>
              )
            })}
          </div>
        )}
      </Field>
    </div>
  )
}
