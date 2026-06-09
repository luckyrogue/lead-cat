import { ChipGrid } from "@/shared/ui/cat/chip-grid"
import type { I18nKey } from "@/shared/miniapp/i18n"

export function DurationPicker({
  value,
  onChange,
  options,
  t,
}: {
  value: number
  onChange: (mins: number) => void
  options: number[]
  t: (key: I18nKey) => string
}) {
  const label = (d: number) => {
    if (d < 60) return `${d} ${t("min")}`
    if (d === 60) return `1 ${t("hour")}`
    return `${d / 60} ${t("hour")}`
  }

  return (
    <ChipGrid
      layout="wrap"
      value={value}
      onChange={onChange}
      options={options.map((d) => ({ value: d, label: label(d) }))}
    />
  )
}
