import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon, tgIconBtn } from "@/shared/ui/cat/primitives"
import { useMounted } from "./use-mounted"

export function Overlay({
  open,
  onClose,
  title,
  children,
  footer,
  onBack,
}: {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  footer?: ReactNode
  onBack?: () => void
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(open, 320)
  if (!mounted) return null

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 75,
        background: p.bg,
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
        transform: shown ? "translateX(0)" : "translateX(100%)",
        transition: "transform .32s cubic-bezier(.32,.72,0,1)",
        boxShadow: "-12px 0 40px rgba(0,0,0,0.12)",
      }}
    >
      <div
        style={{
          height: 52,
          flexShrink: 0,
          display: "flex",
          alignItems: "center",
          gap: 8,
          padding: "0 10px",
          borderBottom: `1px solid ${p.border}`,
          background: p.tgBar,
        }}
      >
        <button
          type="button"
          onClick={onBack ?? onClose}
          style={{ ...tgIconBtn(p), marginLeft: 0, width: 38 }}
        >
          <CatIcon
            name={onBack ? "chevL" : "x"}
            size={20}
            color={p.text}
            sw={2.2}
          />
        </button>
        <div
          style={{
            flex: 1,
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 17,
            color: p.text,
            textAlign: "center",
            marginRight: 38,
          }}
        >
          {title}
        </div>
      </div>
      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
      >
        {children}
      </div>
      {footer && (
        <div
          style={{
            flexShrink: 0,
            padding: "12px 16px max(12px, var(--tma-safe-bottom, 0px))",
            borderTop: `1px solid ${p.border}`,
            background: p.tgBar,
          }}
        >
          {footer}
        </div>
      )}
    </div>
  )
}
