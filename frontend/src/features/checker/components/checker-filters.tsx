import { useTmaApp } from "@/shared/tma/context"
import { DurationPicker, Field, Segmented } from "@/shared/ui/cat/primitives"

export function CheckerFilters({
  range,
  onRangeChange,
  dur,
  onDurChange,
}: {
  range: string
  onRangeChange: (v: string) => void
  dur: number
  onDurChange: (v: number) => void
}) {
  const t = useTmaApp().t

  return (
    <>
      <div className="h-[18px]" />
      <Field label={t("dateRange")}>
        <Segmented
          value={range}
          onChange={onRangeChange}
          options={[
            { value: "7", label: "7 дн" },
            { value: "14", label: "14 дн" },
            { value: "30", label: "30 дн" },
          ]}
        />
      </Field>
      <div className="h-4" />
      <Field label={t("duration")}>
        <DurationPicker
          value={dur}
          onChange={onDurChange}
          options={[30, 45, 60, 90, 120]}
          t={t}
        />
      </Field>
    </>
  )
}
