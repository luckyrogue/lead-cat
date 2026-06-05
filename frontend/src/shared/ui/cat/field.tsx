import type { CSSProperties, ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import type { CatPalette } from "@/shared/tma/types"

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
