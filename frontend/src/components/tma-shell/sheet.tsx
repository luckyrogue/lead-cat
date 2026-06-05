import type { ReactNode } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { useMounted } from "./use-mounted"

export function Sheet({
  open,
  onClose,
  children,
  maxH = "88%",
}: {
  open: boolean
  onClose: () => void
  children: ReactNode
  maxH?: string
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(open, 320)
  if (!mounted) return null

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 70,
        display: "flex",
        flexDirection: "column",
        justifyContent: "flex-end",
      }}
    >
      <div
        onClick={onClose}
        style={{
          position: "absolute",
          inset: 0,
          background: shown ? "rgba(20,12,4,0.42)" : "rgba(20,12,4,0)",
          transition: "background .3s",
          backdropFilter: shown ? "blur(2px)" : "none",
        }}
      />
      <div
        style={{
          position: "relative",
          background: p.bg,
          borderRadius: "26px 26px 0 0",
          maxHeight: maxH,
          display: "flex",
          flexDirection: "column",
          transform: shown ? "translateY(0)" : "translateY(102%)",
          transition: "transform .34s cubic-bezier(.32,.72,0,1)",
          boxShadow: "0 -10px 40px rgba(0,0,0,0.25)",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "center",
            paddingTop: 10,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 40,
              height: 5,
              borderRadius: 3,
              background: p.borderStrong,
            }}
          />
        </div>
        <div
          className="lc-scroll"
          style={{ overflow: "auto", padding: "8px 16px 26px" }}
        >
          {children}
        </div>
      </div>
    </div>
  )
}
