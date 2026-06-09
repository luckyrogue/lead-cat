import { cn } from "@/shared/lib/cn"
import { avatarVars } from "@/shared/miniapp/surface-vars"
import { useMiniApp } from "@/shared/miniapp/context"

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
  const { dark } = useMiniApp()
  const hue = avatarColor(name || "?")
  return (
    <div
      className={cn("miniapp-avatar", ring && "miniapp-avatar-ring")}
      style={avatarVars(size, dark, hue)}
    >
      {initials(name || "?")}
    </div>
  )
}
