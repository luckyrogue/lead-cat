import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "../../lib/cn"

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold whitespace-nowrap [&_svg]:pointer-events-none [&_svg]:size-3.5",
  {
    variants: {
      tone: {
        sunny:
          "border-sunny-300/60 bg-sunny-200/70 text-kitty-800 shadow-[0_8px_20px_-14px_oklch(0.85_0.15_88_/_0.7)]",
        coral: "border-coral-300/50 bg-coral-100/70 text-coral-500",
        muted: "border-border/70 bg-muted/70 text-muted-foreground",
      },
    },
    defaultVariants: {
      tone: "sunny",
    },
  }
)

function Badge({
  className,
  tone = "sunny",
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span"

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ tone, className }))}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
