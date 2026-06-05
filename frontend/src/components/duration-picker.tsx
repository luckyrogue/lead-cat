import type { I18nKey } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"

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
  const p = useTmaApp()

  const label = (d: number) => {
    if (d < 60) return `${d} ${t("min")}`
    if (d === 60) return `1 ${t("hour")}`
    return `${d / 60} ${t("hour")}`
  }

  return (
    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
      {options.map((d) => {
        const active = d === value
        return (
          <button
            key={d}
            type="button"
            onClick={() => onChange(d)}
            style={{
              padding: "9px 14px",
              borderRadius: 11,
              border: `1.5px solid ${active ? p.accent : p.border}`,
              background: active ? p.accentSoft : p.card,
              color: active ? p.accent : p.text,
              fontWeight: 700,
              fontSize: 14,
              fontFamily: "var(--font-display)",
              cursor: "pointer",
            }}
          >
            {label(d)}
          </button>
        )
      })}
    </div>
  )
}
