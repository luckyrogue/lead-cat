import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"

type BtnVariant = "primary" | "soft" | "ghost" | "outline" | "danger" | "dark"

const variantClasses: Record<BtnVariant, string> = {
  primary:
    "bg-tma-accent text-tma-accent-text shadow-tma-sm active:shadow-none",
  soft: "bg-tma-accent-soft text-tma-accent",
  ghost: "bg-transparent text-tma-text",
  outline:
    "border-[1.5px] border-tma-border-strong bg-transparent text-tma-text",
  danger: "bg-tma-danger-soft text-tma-danger",
  dark: "bg-tma-text text-tma-bg",
}

const sizeClasses = {
  sm: "h-[38px] rounded-xl px-3.5 text-sm",
  md: "h-[50px] rounded-[15px] px-5 text-base",
  lg: "h-[58px] rounded-[18px] px-6 text-[17px]",
}

export function CatBtn({
  children,
  onClick,
  variant = "primary",
  size = "md",
  full = false,
  disabled = false,
  icon = null,
  className,
}: {
  children: ReactNode
  onClick?: () => void
  variant?: BtnVariant
  size?: "sm" | "md" | "lg"
  full?: boolean
  disabled?: boolean
  icon?: ReactNode
  className?: string
}) {
  return (
    <button
      type="button"
      onClick={disabled ? undefined : onClick}
      disabled={disabled}
      className={cn(
        "font-display inline-flex items-center justify-center gap-2 whitespace-nowrap border-none font-bold transition-[transform,box-shadow,background] duration-150 ease-out active:scale-[0.96]",
        sizeClasses[size],
        variantClasses[variant],
        full ? "w-full" : "w-auto",
        disabled ? "cursor-default opacity-50" : "cursor-pointer",
        className
      )}
    >
      {icon}
      {children}
    </button>
  )
}
