import { useState, type CSSProperties, type ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { typeAccent } from "@/shared/tma/constants"
import type { CatPalette } from "@/shared/tma/types"
import { CatIcon } from "./icon"

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

export function CatCard({
  children,
  onClick,
  style = {},
  pad = 16,
  interactive = false,
}: {
  children: ReactNode
  onClick?: () => void
  style?: CSSProperties
  pad?: number
  interactive?: boolean
}) {
  const p = useTmaApp()
  const [press, setPress] = useState(false)
  return (
    <div
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={onClick ? (e) => e.key === "Enter" && onClick() : undefined}
      onPointerDown={() => interactive && setPress(true)}
      onPointerUp={() => setPress(false)}
      onPointerLeave={() => setPress(false)}
      style={{
        background: p.card,
        borderRadius: 20,
        padding: pad,
        border: `1px solid ${p.border}`,
        boxShadow: p.shadowSm,
        transition: "transform .14s ease, box-shadow .2s ease",
        transform: press ? "scale(0.985)" : "scale(1)",
        cursor: onClick ? "pointer" : "default",
        ...style,
      }}
    >
      {children}
    </div>
  )
}

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
  const a = typeAccent(typeKey, dark)
  const s =
    size === "sm"
      ? { fs: 11.5, px: 8, py: 3, gap: 4 }
      : { fs: 13, px: 10, py: 5, gap: 5 }
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: s.gap,
        background: a.soft,
        color: a.text,
        padding: `${s.py}px ${s.px}px`,
        borderRadius: 999,
        fontSize: s.fs,
        fontWeight: 700,
        fontFamily: "var(--font-display)",
        lineHeight: 1,
        whiteSpace: "nowrap",
      }}
    >
      <span style={{ fontSize: s.fs }}>{a.emoji}</span>
      {label}
    </span>
  )
}

export function CatToggle({
  on,
  onChange,
}: {
  on: boolean
  onChange: (v: boolean) => void
}) {
  const p = useTmaApp()
  return (
    <button
      type="button"
      onClick={() => onChange(!on)}
      style={{
        width: 50,
        height: 30,
        borderRadius: 999,
        border: "none",
        cursor: "pointer",
        background: on
          ? p.accent
          : p.dark
            ? "rgba(255,255,255,0.16)"
            : "#E4D7C8",
        position: "relative",
        transition: "background .22s ease",
        flexShrink: 0,
        padding: 0,
      }}
    >
      <span
        style={{
          position: "absolute",
          top: 3,
          left: on ? 23 : 3,
          width: 24,
          height: 24,
          borderRadius: "50%",
          background: "#fff",
          transition: "left .22s cubic-bezier(.34,1.56,.64,1)",
          boxShadow: "0 1px 4px rgba(0,0,0,0.25)",
        }}
      />
    </button>
  )
}

export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        background: p.dark ? "rgba(255,255,255,0.06)" : "#F2E6D8",
        borderRadius: 13,
        padding: 4,
        gap: 2,
      }}
    >
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            style={{
              flex: 1,
              height: 36,
              border: "none",
              borderRadius: 10,
              cursor: "pointer",
              background: active ? p.card : "transparent",
              color: active ? p.text : p.muted,
              fontWeight: 700,
              fontSize: 13.5,
              fontFamily: "var(--font-display)",
              boxShadow: active ? p.shadowSm : "none",
              transition: "all .18s ease",
              whiteSpace: "nowrap",
              padding: "0 6px",
            }}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}

export function Field({
  label,
  children,
}: {
  label?: string
  children: ReactNode
}) {
  const p = useTmaApp()
  return (
    <label style={{ display: "block" }}>
      {label && (
        <div
          style={{
            fontSize: 13,
            fontWeight: 700,
            color: p.muted,
            marginBottom: 7,
            fontFamily: "var(--font-display)",
          }}
        >
          {label}
        </div>
      )}
      {children}
    </label>
  )
}

export function inputStyle(p: CatPalette): CSSProperties {
  return {
    width: "100%",
    boxSizing: "border-box",
    height: 50,
    padding: "0 14px",
    borderRadius: 14,
    border: `1.5px solid ${p.border}`,
    background: p.dark ? p.cardAlt : "#fff",
    color: p.text,
    fontSize: 16,
    fontFamily: "var(--font-body)",
    outline: "none",
  }
}

export function tgIconBtn(p: CatPalette): CSSProperties {
  return {
    height: 34,
    minWidth: 34,
    padding: "0 7px",
    borderRadius: 999,
    border: "none",
    background: p.dark ? "rgba(255,255,255,0.07)" : "#F2E9DE",
    cursor: "pointer",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 3,
    marginLeft: 6,
  }
}

export { CatIcon }
