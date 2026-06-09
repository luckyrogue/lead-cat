import { cn } from "@/shared/lib/cn"
import { hueSurfaceVars } from "@/shared/tma/surface-vars"
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
  const { dark } = useTmaApp()
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-3 border-none bg-transparent px-3.5 py-[13px] text-left",
        !last && "border-tma-border border-b",
        onClick ? "cursor-pointer" : "cursor-default"
      )}
    >
      <div
        className="tma-hue-surface flex size-[34px] shrink-0 items-center justify-center rounded-[10px]"
        style={hueSurfaceVars(hue, dark)}
      >
        <CatIcon name={icon} size={18} className="text-tma-hue-fg" sw={2} />
      </div>
      <span className="font-display text-tma-text flex-1 text-[15px] font-bold">
        {label}
      </span>
      {right}
    </button>
  )
}
