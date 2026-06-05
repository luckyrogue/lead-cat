import { TMA_NOW } from "@/shared/tma/constants"
import { typeAccent } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import { MEETING_TYPES, RECURRENCE } from "@/shared/tma/mock-data"
import { fmtDate } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/entities/meeting/types"
import { CatCard, CatIcon, TypeTag } from "@/shared/ui/cat/primitives"
import { ParticipantStack } from "./participant-stack"

export function MeetingCard({
  m,
  onClick,
  compact = false,
}: {
  m: Meeting
  onClick?: () => void
  compact?: boolean
}) {
  const p = useTmaApp()
  const a = typeAccent(m.type, p.dark)
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const past = m.date < TMA_NOW
  const rec = RECURRENCE.find((r) => r.key === m.rec)

  return (
    <CatCard
      onClick={onClick}
      interactive
      pad={0}
      style={{ overflow: "hidden", opacity: past ? 0.72 : 1 }}
    >
      <div style={{ display: "flex" }}>
        <div style={{ width: 5, background: a.solid, flexShrink: 0 }} />
        <div style={{ flex: 1, padding: "13px 14px", minWidth: 0 }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 7,
              marginBottom: 8,
            }}
          >
            <TypeTag
              typeKey={m.type}
              label={tObj ? tObj.label : m.type}
              size="sm"
            />
            {rec?.short && (
              <span
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 3,
                  color: p.muted,
                  fontSize: 11.5,
                  fontWeight: 700,
                }}
              >
                <CatIcon name="repeat" size={12} color={p.muted} sw={2} />
                {rec.short}
              </span>
            )}
            <div style={{ flex: 1 }} />
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: 4,
                background: past ? p.faint : p.ok,
                flexShrink: 0,
              }}
            />
          </div>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 700,
              fontSize: 15.5,
              color: p.text,
              lineHeight: 1.2,
              marginBottom: 8,
              wordBreak: "break-word",
            }}
          >
            {m.dept} · {tObj ? tObj.label : m.type}
          </div>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              gap: 8,
            }}
          >
            <div
              style={{
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
            {!compact && (
              <ParticipantStack
                emails={[m.organizer, ...m.participants]}
                max={3}
                size={26}
              />
            )}
          </div>
        </div>
      </div>
    </CatCard>
  )
}
