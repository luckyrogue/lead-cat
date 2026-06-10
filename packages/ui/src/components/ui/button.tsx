import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "../../lib/cn"

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center gap-2 rounded-[calc(var(--radius)*0.95)] border border-transparent text-sm font-semibold whitespace-nowrap transition-[transform,box-shadow,background-color,border-color,color,opacity] duration-300 [transition-timing-function:var(--ease-spring)] outline-none select-none focus-visible:border-ring focus-visible:ring-4 focus-visible:ring-ring/30 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-[linear-gradient(135deg,var(--color-coral-400),var(--color-coral-500))] text-primary-foreground shadow-[0_18px_40px_-22px_oklch(0.64_0.2_28_/_0.55)] hover:-translate-y-0.5 hover:shadow-[0_24px_48px_-22px_oklch(0.64_0.2_28_/_0.5)]",
        secondary:
          "border border-border/70 bg-secondary text-secondary-foreground hover:-translate-y-0.5 hover:bg-secondary/80",
        outline:
          "border-border/80 bg-background/80 text-foreground hover:-translate-y-0.5 hover:border-border hover:bg-background",
        ghost: "hover:bg-accent/70 hover:text-accent-foreground",
        destructive:
          "bg-destructive text-white shadow-[0_16px_36px_-22px_oklch(0.62_0.2_27_/_0.5)] hover:-translate-y-0.5",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-4",
        sm: "h-8 gap-1.5 rounded-[calc(var(--radius)*0.75)] px-3 text-[0.82rem]",
        lg: "h-12 rounded-[calc(var(--radius)*1.1)] px-6 text-base",
        icon: "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
