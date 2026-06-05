import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function SettingsRow({
  icon,
  hue = 45,
  label,
  right,
  onClick,
  last = false,
}: {
  icon: Parameters<typeof CatIcon>[0]["name"]
  hue?: number
  label: string
  right?: React.ReactNode
  onClick?: () => void
  last?: boolean
}) {
  const p = useTmaApp()
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        width: "100%",
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "13px 14px",
        border: "none",
        borderBottom: last ? "none" : `1px solid ${p.border}`,
        background: "transparent",
        cursor: onClick ? "pointer" : "default",
        textAlign: "left",
      }}
    >
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: 10,
          background: `oklch(${p.dark ? 0.36 : 0.94} ${p.dark ? 0.08 : 0.06} ${hue})`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <CatIcon
          name={icon}
          size={18}
          color={`oklch(${p.dark ? 0.82 : 0.52} 0.15 ${hue})`}
          sw={2}
        />
      </div>
      <span
        style={{
          flex: 1,
          fontWeight: 700,
          fontSize: 15,
          color: p.text,
          fontFamily: "var(--font-display)",
        }}
      >
        {label}
      </span>
      {right}
    </button>
  )
}
