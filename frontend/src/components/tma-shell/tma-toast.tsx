import { useTmaApp } from "@/shared/tma/context"
import { useMounted } from "./use-mounted"

export function TmaToast({
  data,
}: {
  data: { msg: string; emoji?: string } | null
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(!!data, 300)
  if (!mounted || !data) return null

  return (
    <div
      style={{
        position: "absolute",
        left: 0,
        right: 0,
        top: 60,
        display: "flex",
        justifyContent: "center",
        zIndex: 90,
        pointerEvents: "none",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 9,
          padding: "11px 16px",
          background: p.dark ? "#2A3A48" : "#2D2A26",
          color: "#fff",
          borderRadius: 14,
          fontWeight: 700,
          fontSize: 14,
          fontFamily: "var(--font-display)",
          boxShadow: "0 10px 30px rgba(0,0,0,0.3)",
          maxWidth: "84%",
          transform: shown
            ? "translateY(0) scale(1)"
            : "translateY(-16px) scale(0.92)",
          opacity: shown ? 1 : 0,
          transition: "all .3s cubic-bezier(.34,1.56,.64,1)",
        }}
      >
        <span style={{ fontSize: 17 }}>{data.emoji ?? "🐾"}</span>
        {data.msg}
      </div>
    </div>
  )
}
