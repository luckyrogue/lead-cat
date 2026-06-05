import { useTmaApp } from "@/shared/tma/context"

export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        background: p.dark ? "rgba(255,255,255,0.06)" : "#F2E6D8",
        borderRadius: 13,
        padding: 4,
        gap: 2,
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
              flex: 1,
              height: 36,
              border: "none",
              borderRadius: 10,
              cursor: "pointer",
              background: active ? p.card : "transparent",
              color: active ? p.text : p.muted,
              fontWeight: 700,
              fontSize: 13.5,
              fontFamily: "var(--font-display)",
              boxShadow: active ? p.shadowSm : "none",
              transition: "all .18s ease",
              whiteSpace: "nowrap",
              padding: "0 6px",
            }}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}
