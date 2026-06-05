import { emailsToPeople } from "@/shared/tma/mock-data"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar } from "@/shared/ui/cat/primitives"

export function ParticipantStack({
  emails,
  max = 4,
  size = 28,
}: {
  emails: string[]
  max?: number
  size?: number
}) {
  const p = useTmaApp()
  const people = emailsToPeople(emails)
  const shown = people.slice(0, max)
  const extra = people.length - shown.length
  return (
    <div style={{ display: "flex", alignItems: "center" }}>
      {shown.map((per, i) => (
        <div key={i} style={{ marginLeft: i === 0 ? 0 : -9 }}>
          <Avatar name={per.name} size={size} ring />
        </div>
      ))}
      {extra > 0 && (
        <div
          style={{
            marginLeft: -9,
            width: size,
            height: size,
            borderRadius: "50%",
            background: p.dark ? "#2C3A47" : "#EFE3D5",
            color: p.muted,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: size * 0.34,
            fontWeight: 800,
            fontFamily: "var(--font-display)",
            boxShadow: `0 0 0 2px ${p.dark ? "#1E2A35" : "#fff"}`,
          }}
        >
          +{extra}
        </div>
      )}
    </div>
  )
}
