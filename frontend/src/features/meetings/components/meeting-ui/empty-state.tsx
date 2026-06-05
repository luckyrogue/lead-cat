import { useTmaApp } from "@/shared/tma/context"

export function EmptyState({
  emoji = "🐈",
  title,
  sub,
}: {
  emoji?: string
  title: string
  sub?: string
}) {
  const p = useTmaApp()
  return (
    <div style={{ textAlign: "center", padding: "44px 24px" }}>
      <div
        style={{
          fontSize: 52,
          marginBottom: 12,
          animation: "lc-bob 3s ease-in-out infinite",
        }}
      >
        {emoji}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 17,
          color: p.text,
          marginBottom: 6,
        }}
      >
        {title}
      </div>
      {sub && (
        <div
          style={{
            color: p.muted,
            fontSize: 14,
            maxWidth: 240,
            margin: "0 auto",
            lineHeight: 1.4,
          }}
        >
          {sub}
        </div>
      )}
    </div>
  )
}
