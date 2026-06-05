import { useTmaApp } from "@/shared/tma/context"

export function ChipGrid<T extends string>({
  options,
  value,
  onChange,
  cols = 2,
}: {
  options: { value: T; label: string; emoji?: string }[]
  value: T
  onChange: (v: T) => void
  cols?: number
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: `repeat(${cols},1fr)`,
        gap: 9,
      }}
    >
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              padding: "13px 13px",
              borderRadius: 14,
              border: `1.5px solid ${active ? p.accent : p.border}`,
              cursor: "pointer",
              textAlign: "left",
              background: active ? p.accentSoft : p.card,
              color: p.text,
            }}
          >
            {o.emoji && <span style={{ fontSize: 18 }}>{o.emoji}</span>}
            <span
              style={{
                fontFamily: "var(--font-display)",
                fontWeight: 700,
                fontSize: 14,
                lineHeight: 1.1,
              }}
            >
              {o.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}
