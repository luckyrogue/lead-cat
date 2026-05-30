import type { ReactNode } from "react"
import { TMA_NOW } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import {
  emailsToPeople,
  MEETING_TYPES,
  RECURRENCE,
} from "@/shared/tma/mock-data"
import { fmtDate, meetCount, partWord } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/shared/tma/types"
import { typeAccent } from "@/shared/tma/constants"
import { Avatar, CatCard, CatIcon, TypeTag } from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"
import { hexToRgba } from "@/shared/tma/palette"

export function SectionTitle({
  children,
  action,
  onAction,
}: {
  children: ReactNode
  action?: string | null
  onAction?: () => void
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        margin: "0 4px 10px",
      }}
    >
      <h3
        style={{
          margin: 0,
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 18,
          color: p.text,
        }}
      >
        {children}
      </h3>
      {action && onAction && (
        <button
          type="button"
          onClick={onAction}
          style={{
            background: "none",
            border: "none",
            color: p.accent,
            fontWeight: 700,
            fontSize: 13.5,
            cursor: "pointer",
            fontFamily: "var(--font-display)",
            display: "flex",
            alignItems: "center",
            gap: 2,
          }}
        >
          {action}
          <CatIcon name="chevR" size={14} color={p.accent} sw={2.4} />
        </button>
      )}
    </div>
  )
}

export function ParticipantStack({
  emails,
  max = 4,
  size = 28,
}: {
  emails: string[]
  max?: number
  size?: number
}) {
  const p = useTmaApp()
  const people = emailsToPeople(emails)
  const shown = people.slice(0, max)
  const extra = people.length - shown.length
  return (
    <div style={{ display: "flex", alignItems: "center" }}>
      {shown.map((per, i) => (
        <div key={i} style={{ marginLeft: i === 0 ? 0 : -9 }}>
          <Avatar name={per.name} size={size} ring />
        </div>
      ))}
      {extra > 0 && (
        <div
          style={{
            marginLeft: -9,
            width: size,
            height: size,
            borderRadius: "50%",
            background: p.dark ? "#2C3A47" : "#EFE3D5",
            color: p.muted,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: size * 0.34,
            fontWeight: 800,
            fontFamily: "var(--font-display)",
            boxShadow: `0 0 0 2px ${p.dark ? "#1E2A35" : "#fff"}`,
          }}
        >
          +{extra}
        </div>
      )}
    </div>
  )
}

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

export function EmptyState({
  emoji = "🐈",
  title,
  sub,
}: {
  emoji?: string
  title: string
  sub?: string
}) {
  const p = useTmaApp()
  return (
    <div style={{ textAlign: "center", padding: "44px 24px" }}>
      <div
        style={{
          fontSize: 52,
          marginBottom: 12,
          animation: "lc-bob 3s ease-in-out infinite",
        }}
      >
        {emoji}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 17,
          color: p.text,
          marginBottom: 6,
        }}
      >
        {title}
      </div>
      {sub && (
        <div
          style={{
            color: p.muted,
            fontSize: 14,
            maxWidth: 240,
            margin: "0 auto",
            lineHeight: 1.4,
          }}
        >
          {sub}
        </div>
      )}
    </div>
  )
}

export { meetCount, fmtDate, partWord, Paw, hexToRgba }
