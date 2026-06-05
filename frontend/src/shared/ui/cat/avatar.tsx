import { useTmaApp } from "@/shared/tma/context"

const AV_HUES = [25, 150, 255, 300, 95, 180, 45, 350]

export function avatarColor(name: string): number {
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) % 9973
  return AV_HUES[h % AV_HUES.length]
}

export function initials(name: string): string {
  const p = name.trim().split(/\s+/)
  return ((p[0]?.[0] ?? "") + (p[1]?.[0] ?? "")).toUpperCase()
}

export function Avatar({
  name,
  size = 38,
  ring = false,
}: {
  name: string
  size?: number
  ring?: boolean
}) {
  const { dark } = useTmaApp()
  const hue = avatarColor(name || "?")
  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: "50%",
        flexShrink: 0,
        background: `oklch(${dark ? 0.42 : 0.92} ${dark ? 0.09 : 0.07} ${hue})`,
        color: `oklch(${dark ? 0.92 : 0.45} 0.13 ${hue})`,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontWeight: 700,
        fontSize: size * 0.38,
        fontFamily: "var(--font-display)",
        boxShadow: ring ? `0 0 0 2px ${dark ? "#1E2A35" : "#fff"}` : "none",
      }}
    >
      {initials(name || "?")}
    </div>
  )
}
