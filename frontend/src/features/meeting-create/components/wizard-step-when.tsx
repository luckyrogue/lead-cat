import { WEEKDAYS } from "@/entities/meeting/constants"
import { useTmaApp } from "@/shared/tma/context"
import { RECURRENCE } from "@/entities/meeting/constants"
import type { MeetingDraft } from "@/shared/tma/types"
import { cn } from "@/shared/lib/cn"
import { ChipGrid, DurationPicker, Field } from "@/shared/ui/cat/primitives"
import { DURATION_OPTIONS, TIME_SLOTS } from "../lib/wizard-constants"
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
  const { t } = useTmaApp()

  return (
    <div>
      <WizardStepTitle>🗓️ Когда встречаемся?</WizardStepTitle>
      <Field label={t("dateT")}>
        <MiniCalendar value={draft.date} onChange={(v) => set("date", v)} />
      </Field>
      <div className="h-[18px]" />
      <Field
        label={`${t("timeT")} — ${t("from")} ${draft.start}, ${t("to")} ${endTime}`}
      >
        <div className="lc-scroll flex gap-2 overflow-x-auto pb-1">
          {TIME_SLOTS.map((tm) => {
            const active = tm === draft.start
            return (
              <button
                key={tm}
                type="button"
                onClick={() => set("start", tm)}
                className={cn(
                  "font-display shrink-0 cursor-pointer rounded-[11px] border-[1.5px] px-[13px] py-[9px] text-sm font-bold",
                  active
                    ? "border-tma-accent bg-tma-accent text-tma-accent-text"
                    : "border-tma-border bg-tma-card text-tma-text"
                )}
              >
                {tm}
              </button>
            )
          })}
        </div>
      </Field>
      <div className="h-4" />
      <Field label={t("duration")}>
        <DurationPicker
          value={draft.dur}
          onChange={(v) => set("dur", v)}
          options={[...DURATION_OPTIONS]}
          t={t}
        />
      </Field>
      <div className="h-[18px]" />
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
          <div className="mt-[11px] flex gap-[7px]">
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
                  className={cn(
                    "font-display aspect-square flex-1 cursor-pointer rounded-[11px] border-[1.5px] text-[12.5px] font-extrabold",
                    on
                      ? "border-tma-accent bg-tma-accent text-tma-accent-text"
                      : "border-tma-border bg-tma-card text-tma-muted"
                  )}
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
