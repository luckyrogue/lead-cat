import { useState, type CSSProperties, type ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"

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
