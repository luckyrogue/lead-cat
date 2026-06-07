import { cn } from "@/shared/lib/cn"
import { typeAccent } from "@/entities/meeting/constants"
import { typeAccentVars } from "@/shared/tma/surface-vars"
import { useTmaApp } from "@/shared/tma/context"

export function TypeTag({
  typeKey,
  label,
  size = "md",
}: {
  typeKey: string
  label: string
  size?: "sm" | "md"
}) {
  const { dark } = useTmaApp()
  const accent = typeAccent(typeKey, dark)
  return (
    <span
      className={cn(
        "font-display inline-flex items-center whitespace-nowrap rounded-full bg-type-soft font-bold leading-none text-type-accent",
        size === "sm"
          ? "gap-1 px-2 py-[3px] text-[11.5px]"
          : "gap-[5px] px-2.5 py-[5px] text-[13px]"
      )}
      style={typeAccentVars(typeKey, dark)}
    >
      <span className={size === "sm" ? "text-[11.5px]" : "text-[13px]"}>
        {accent.emoji}
      </span>
      {label}
    </span>
  )
}
