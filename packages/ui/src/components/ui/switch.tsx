import * as React from "react"

import { cn } from "../../lib/cn"

type SwitchProps = Omit<React.ComponentProps<"button">, "onClick"> & {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  size?: "default" | "sm"
}

function Switch({
  checked,
  onCheckedChange,
  size = "default",
  className,
  ...props
}: SwitchProps) {
  const sm = size === "sm"
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      data-slot="switch"
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        "relative inline-flex shrink-0 items-center rounded-full border-2 transition-colors",
        sm ? "h-5 w-9" : "h-6 w-11",
        checked ? "border-primary bg-primary" : "border-input bg-input",
        className
      )}
      {...props}
    >
      <span
        className={cn(
          "inline-block rounded-full transition-transform",
          sm ? "h-3 w-3" : "h-4 w-4",
          checked ? "bg-primary-foreground" : "bg-background",
          checked ? (sm ? "translate-x-4" : "translate-x-5") : "translate-x-0.5"
        )}
      />
    </button>
  )
}

export { Switch }
