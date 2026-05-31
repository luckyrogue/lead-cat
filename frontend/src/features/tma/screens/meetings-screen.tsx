import { useState } from "react"
import { TMA_NOW } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import {
  MEETING_TYPES,
  RECURRENCE,
  emailsToPeople,
} from "@/shared/tma/mock-data"
import { useTmaAuth } from "@/shared/tma/auth-context"
import { buildTitle, fmtDate, partWord } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/shared/tma/types"
import {
  Avatar,
  CatBtn,
  CatCard,
  CatIcon,
  Segmented,
  TypeTag,
} from "@/shared/ui/cat/primitives"
import { EmptyState, MeetingCard } from "../meeting-ui"

export function MeetingsScreen({
  meetings,
  onMeeting,
}: {
  meetings: Meeting[]
  onMeeting: (m: Meeting) => void
}) {
  const p = useTmaApp()
  const t = p.t
  const [filter, setFilter] = useState<"up" | "past" | "all">("up")
  const sorted = [...meetings].sort((a, b) =>
    (a.date + a.start).localeCompare(b.date + b.start)
  )
  const list = sorted.filter((m) => {
    if (filter === "up") return m.date >= TMA_NOW
    if (filter === "past") return m.date < TMA_NOW
    return true
  })
  const groups: { date: string; items: Meeting[] }[] = []
  list.forEach((m) => {
    const g = groups.find((x) => x.date === m.date)
    if (g) g.items.push(m)
    else groups.push({ date: m.date, items: [m] })
  })

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <h2
        style={{
          margin: "2px 4px 14px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 26,
          color: p.text,
        }}
      >
        {t("nav_meetings")}
      </h2>
      <div style={{ marginBottom: 18 }}>
        <Segmented
          value={filter}
          onChange={setFilter}
          options={[
            { value: "up", label: t("filter_up") },
            { value: "past", label: t("filter_past") },
            { value: "all", label: t("filter_all") },
          ]}
        />
      </div>
      {list.length === 0 ? (
        <div
          style={{
            background: p.card,
            borderRadius: 20,
            border: `1px solid ${p.border}`,
            overflow: "hidden",
          }}
        >
          <EmptyState emoji="🐈" title={t("emptyMeet")} />
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          {groups.map((g) => (
            <div key={g.date}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 800,
                  color: p.muted,
                  margin: "0 4px 9px",
                  fontFamily: "var(--font-display)",
                  textTransform: "capitalize",
                }}
              >
                {g.date === TMA_NOW ? t("today") : fmtDate(g.date, p.lang)}
              </div>
              <div
                style={{ display: "flex", flexDirection: "column", gap: 11 }}
              >
                {g.items.map((m) => (
                  <MeetingCard key={m.id} m={m} onClick={() => onMeeting(m)} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function DetailRow({
  icon,
  label,
  children,
}: {
  icon: Parameters<typeof CatIcon>[0]["name"]
  label: string
  children: React.ReactNode
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "flex",
        gap: 12,
        padding: "12px 0",
        borderBottom: `1px solid ${p.border}`,
      }}
    >
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: 10,
          background: p.accentSoft,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <CatIcon name={icon} size={18} color={p.accent} sw={2} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            color: p.muted,
            fontWeight: 600,
            marginBottom: 2,
          }}
        >
          {label}
        </div>
        <div style={{ fontSize: 15, color: p.text, fontWeight: 600 }}>
          {children}
        </div>
      </div>
    </div>
  )
}

export function MeetingDetail({
  m,
  onEdit,
  onDelete,
}: {
  m: Meeting
  onEdit: () => void
  onDelete: () => void
}) {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const t = p.t
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const rec = RECURRENCE.find((r) => r.key === m.rec)
  const past = m.date < TMA_NOW
  const canManage = m.organizer === user?.email || user?.role === "admin"
  const allPeople = emailsToPeople([m.organizer, ...m.participants])

  return (
    <div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 14,
          marginTop: 2,
        }}
      >
        <TypeTag typeKey={m.type} label={tObj ? tObj.label : m.type} />
        <span
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 5,
            padding: "5px 10px",
            borderRadius: 999,
            fontSize: 12,
            fontWeight: 700,
            background: past ? p.dangerSoft : p.okSoft,
            color: past ? p.danger : p.ok,
          }}
        >
          <span
            style={{
              width: 7,
              height: 7,
              borderRadius: 4,
              background: past ? p.danger : p.ok,
            }}
          />
          {past ? t("filter_past") : t("filter_up")}
        </span>
      </div>

      <div
        style={{
          background: p.cardAlt,
          borderRadius: 16,
          padding: "13px 15px",
          marginBottom: 16,
          border: `1px dashed ${p.borderStrong}`,
        }}
      >
        <div
          style={{
            fontSize: 11,
            color: p.faint,
            fontWeight: 700,
            marginBottom: 5,
            letterSpacing: 0.3,
            textTransform: "uppercase",
          }}
        >
          {t("autoName")}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 17,
            color: p.text,
            lineHeight: 1.25,
            wordBreak: "break-word",
          }}
        >
          {buildTitle(m)}
        </div>
      </div>

      <button
        type="button"
        onClick={() => p.showToast("🔗 Google Meet", "🔗")}
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

      <DetailRow icon="calendar" label={t("dateT")}>
        {fmtDate(m.date, p.lang)} · {m.date.split("-").reverse().join(".")}
      </DetailRow>
      <DetailRow icon="clock" label={t("timeT")}>
        {m.start} – {m.end}{" "}
        <span style={{ color: p.faint, fontWeight: 600, fontSize: 13 }}>
          · UTC+5
        </span>
      </DetailRow>
      <DetailRow icon="repeat" label={t("recur")}>
        {rec ? rec.label : "—"}
        {m.rec === "weekly" && m.recDays
          ? ` (${m.recDays.map((d) => ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"][d - 1]).join(", ")})`
          : ""}
      </DetailRow>
      <DetailRow icon="user" label={t("host")}>
        {m.host}
      </DetailRow>
      {m.desc && (
        <DetailRow icon="doc" label={t("descr")}>
          {m.desc}
        </DetailRow>
      )}

      <div style={{ marginTop: 18 }}>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: 10,
          }}
        >
          <span
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 800,
              fontSize: 16,
              color: p.text,
            }}
          >
            {t("addPeople")}
          </span>
          <span style={{ fontSize: 13, color: p.muted, fontWeight: 700 }}>
            {allPeople.length} {partWord(allPeople.length, t)}
          </span>
        </div>
        <CatCard pad={6}>
          {allPeople.map((per, i) => (
            <div
              key={i}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 11,
                padding: "8px 8px",
                borderBottom:
                  i < allPeople.length - 1 ? `1px solid ${p.border}` : "none",
              }}
            >
              <Avatar name={per.name} size={36} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontWeight: 700,
                    fontSize: 14.5,
                    color: p.text,
                    fontFamily: "var(--font-display)",
                    whiteSpace: "nowrap",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                  }}
                >
                  {per.name}
                </div>
                <div
                  style={{
                    fontSize: 12,
                    color: p.muted,
                    whiteSpace: "nowrap",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                  }}
                >
                  {per.email}
                </div>
              </div>
              {per.email === m.organizer ? (
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 800,
                    color: p.accent,
                    background: p.accentSoft,
                    padding: "3px 8px",
                    borderRadius: 999,
                  }}
                >
                  👑
                </span>
              ) : per.tg ? (
                <span style={{ fontSize: 15 }} title="Telegram">
                  ✈️
                </span>
              ) : (
                <span style={{ fontSize: 11, color: p.faint, fontWeight: 700 }}>
                  email
                </span>
              )}
            </div>
          ))}
        </CatCard>
      </div>

      {canManage && !past && (
        <div style={{ display: "flex", gap: 10, marginTop: 18 }}>
          <CatBtn
            variant="outline"
            full
            icon={<CatIcon name="pencil" size={18} color={p.text} sw={2} />}
            onClick={onEdit}
          >
            {t("edit")}
          </CatBtn>
          <CatBtn
            variant="danger"
            icon={<CatIcon name="trash" size={18} color={p.danger} sw={2} />}
            onClick={onDelete}
            style={{ flex: "0 0 auto" }}
          >
            {t("del")}
          </CatBtn>
        </div>
      )}
    </div>
  )
}
