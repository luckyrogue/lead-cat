import { cn } from "@/shared/lib/cn"
import { hueSurfaceVars } from "@/shared/miniapp/surface-vars"
import { useMiniApp } from "@/shared/miniapp/context"
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
  const { dark } = useMiniApp()
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-3 border-none bg-transparent px-3.5 py-[13px] text-left",
        !last && "border-miniapp-border border-b",
        onClick ? "cursor-pointer" : "cursor-default"
      )}
    >
      <div
        className="miniapp-hue-surface flex size-[34px] shrink-0 items-center justify-center rounded-[10px]"
        style={hueSurfaceVars(hue, dark)}
      >
        <CatIcon name={icon} size={18} className="text-miniapp-hue-fg" sw={2} />
      </div>
      <span className="font-display text-miniapp-text flex-1 text-[15px] font-bold">
        {label}
      </span>
      {right}
    </button>
  )
}
