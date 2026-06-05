import { toastSuccess } from "@/shared/lib/toast"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingDetailMeetLink() {
  const p = useTmaApp()
  const t = p.t

  return (
    <button
      type="button"
      onClick={() => toastSuccess("🔗 Google Meet")}
      style={{
        width: "100%",
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "13px 15px",
        borderRadius: 16,
        border: "none",
        cursor: "pointer",
        marginBottom: 16,
        background: p.accent,
        color: p.accentText,
        boxShadow: p.shadowSm,
      }}
    >
      <span
        style={{
          width: 36,
          height: 36,
          borderRadius: 11,
          background: "rgba(255,255,255,0.22)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <CatIcon name="link" size={19} color={p.accentText} sw={2.2} />
      </span>
      <div style={{ flex: 1, textAlign: "left" }}>
        <div style={{ fontSize: 12, opacity: 0.85, fontWeight: 600 }}>
          {t("meetLink")}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 16,
          }}
        >
          {t("joinMeet")}
        </div>
      </div>
      <CatIcon name="arrowR" size={20} color={p.accentText} sw={2.2} />
    </button>
  )
}
