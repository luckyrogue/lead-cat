import { buildTitle, fmtDate } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/entities/meeting/types"
import { useTmaApp } from "@/shared/tma/context"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingCreatedSuccess({
  m,
  onDone,
  onView,
}: {
  m: Meeting
  onDone: () => void
  onView: () => void
}) {
  const p = useTmaApp()
  return (
    <div style={{ textAlign: "center", padding: "10px 6px 6px" }}>
      <div
        style={{
          fontSize: 60,
          marginBottom: 8,
          animation: "lc-pop .5s cubic-bezier(.34,1.56,.64,1)",
        }}
      >
        🐱
      </div>
      <h2
        style={{
          margin: "0 0 6px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 24,
          color: p.text,
        }}
      >
        {p.t("created")}
      </h2>
      <p style={{ margin: "0 0 18px", color: p.muted, fontSize: 14.5 }}>
        {p.t("createdSub")}
      </p>
      <div
        style={{
          background: p.cardAlt,
          borderRadius: 16,
          padding: "13px 15px",
          marginBottom: 18,
          textAlign: "left",
          border: `1px solid ${p.border}`,
        }}
      >
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 15.5,
            color: p.text,
            lineHeight: 1.3,
          }}
        >
          {buildTitle(m)}
        </div>
        <div
          style={{
            marginTop: 8,
            display: "flex",
            alignItems: "center",
            gap: 10,
            color: p.muted,
            fontSize: 13,
            fontWeight: 600,
          }}
        >
          <span
            style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
          >
            <CatIcon name="calendar" size={14} color={p.muted} sw={2} />
            {fmtDate(m.date, p.lang)}
          </span>
          <span
            style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
          >
            <CatIcon name="clock" size={14} color={p.muted} sw={2} />
            {m.start}–{m.end}
          </span>
        </div>
      </div>
      <div style={{ display: "flex", gap: 10 }}>
        <CatBtn variant="outline" full onClick={onView}>
          {p.t("preview")}
        </CatBtn>
        <CatBtn variant="primary" full onClick={onDone}>
          OK 🐾
        </CatBtn>
      </div>
    </div>
  )
}
