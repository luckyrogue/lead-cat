import { useState, type CSSProperties, type ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"

type BtnVariant = "primary" | "soft" | "ghost" | "outline" | "danger" | "dark"

export function CatBtn({
  children,
  onClick,
  variant = "primary",
  size = "md",
  full = false,
  disabled = false,
  style = {},
  icon = null,
}: {
  children: ReactNode
  onClick?: () => void
  variant?: BtnVariant
  size?: "sm" | "md" | "lg"
  full?: boolean
  disabled?: boolean
  style?: CSSProperties
  icon?: ReactNode
}) {
  const p = useTmaApp()
  const [press, setPress] = useState(false)
  const sizes = {
    sm: { h: 38, px: 14, fs: 14, r: 12 },
    md: { h: 50, px: 20, fs: 16, r: 15 },
    lg: { h: 58, px: 24, fs: 17, r: 18 },
  }[size]
  const base: CSSProperties = {
    height: sizes.h,
    padding: `0 ${sizes.px}px`,
    borderRadius: sizes.r,
    fontSize: sizes.fs,
    fontWeight: 700,
    fontFamily: "var(--font-display)",
    border: "none",
    cursor: disabled ? "default" : "pointer",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    width: full ? "100%" : "auto",
    whiteSpace: "nowrap",
    transition: "transform .12s ease, box-shadow .2s ease, background .2s ease",
    transform: press && !disabled ? "scale(0.96)" : "scale(1)",
    opacity: disabled ? 0.5 : 1,
  }
  const variants: Record<BtnVariant, CSSProperties> = {
    primary: {
      background: p.accent,
      color: p.accentText,
      boxShadow: press ? "none" : p.shadowSm,
    },
    soft: { background: p.accentSoft, color: p.accent },
    ghost: { background: "transparent", color: p.text },
    outline: {
      background: "transparent",
      color: p.text,
      border: `1.5px solid ${p.borderStrong}`,
    },
    danger: { background: p.dangerSoft, color: p.danger },
    dark: { background: p.text, color: p.bg },
  }
  return (
    <button
      type="button"
      onClick={disabled ? undefined : onClick}
      onPointerDown={() => setPress(true)}
      onPointerUp={() => setPress(false)}
      onPointerLeave={() => setPress(false)}
      style={{ ...base, ...variants[variant], ...style }}
    >
      {icon}
      {children}
    </button>
  )
}
