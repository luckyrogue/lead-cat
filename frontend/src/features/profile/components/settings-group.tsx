import { useTmaApp } from "@/shared/tma/context"
import { CatCard } from "@/shared/ui/cat/primitives"

export function SettingsGroup({
  title,
  children,
}: {
  title?: string
  children: React.ReactNode
}) {
  const p = useTmaApp()
  return (
    <div style={{ marginBottom: 20 }}>
      {title && (
        <div
          style={{
            fontSize: 13,
            fontWeight: 800,
            color: p.muted,
            margin: "0 4px 9px",
            fontFamily: "var(--font-display)",
          }}
        >
          {title}
        </div>
      )}
      <CatCard pad={0} style={{ overflow: "hidden" }}>
        {children}
      </CatCard>
    </div>
  )
}
