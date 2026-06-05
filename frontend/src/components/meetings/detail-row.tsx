import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function DetailRow({
  icon,
  label,
  children,
}: {
  icon: Parameters<typeof CatIcon>[0]["name"]
  label: string
  children: React.ReactNode
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        gap: 12,
        padding: "12px 0",
        borderBottom: `1px solid ${p.border}`,
      }}
    >
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: 10,
          background: p.accentSoft,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <CatIcon name={icon} size={18} color={p.accent} sw={2} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            color: p.muted,
            fontWeight: 600,
            marginBottom: 2,
          }}
        >
          {label}
        </div>
        <div style={{ fontSize: 15, color: p.text, fontWeight: 600 }}>
          {children}
        </div>
      </div>
    </div>
  )
}
