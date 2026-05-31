import { TMA_NOW } from "@/shared/tma/constants"
import { useTmaAuth } from "@/shared/tma/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import type { Meeting, TabKey } from "@/shared/tma/types"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"
import { hexToRgba } from "@/shared/tma/palette"
import { EmptyState, MeetingCard, meetCount, SectionTitle } from "../meeting-ui"

export function HomeScreen({
  meetings,
  onCreate,
  onMeeting,
  onColleague,
  onTab,
}: {
  meetings: Meeting[]
  onCreate: () => void
  onMeeting: (m: Meeting) => void
  onColleague: () => void
  onTab: (t: TabKey) => void
}) {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const t = p.t
  const today = meetings
    .filter((m) => m.date === TMA_NOW)
    .sort((a, b) => a.start.localeCompare(b.start))
  const upcoming = meetings
    .filter((m) => m.date > TMA_NOW)
    .sort((a, b) => (a.date + a.start).localeCompare(b.date + b.start))
    .slice(0, 4)
  const firstName = (user?.name ?? "").split(" ")[0]

  const actions = [
    {
      key: "create",
      icon: "calendar" as const,
      label: t("create"),
      hue: 45,
      onClick: onCreate,
    },
    {
      key: "find",
      icon: "search" as const,
      label: t("findTime"),
      hue: 180,
      onClick: () => onTab("checker"),
    },
    {
      key: "sched",
      icon: "users" as const,
      label: t("schedule"),
      hue: 300,
      onClick: onColleague,
    },
    {
      key: "auto",
      icon: "bolt" as const,
      label: t("automate"),
      hue: 25,
      onClick: () => onTab("auto"),
    },
  ]

  return (
    <div
      style={{
        padding: "16px 16px 28px",
        display: "flex",
        flexDirection: "column",
        gap: 22,
      }}
    >
      <div
        style={{
          position: "relative",
          overflow: "hidden",
          borderRadius: 24,
          padding: "20px 20px 22px",
          background: `linear-gradient(135deg, ${hexToRgba(p.accent, p.dark ? 0.28 : 0.16)}, ${hexToRgba(p.accent, p.dark ? 0.1 : 0.04)})`,
          border: `1px solid ${p.accentLine}`,
        }}
      >
        <Paw
          size={150}
          color={hexToRgba(p.accent, p.dark ? 0.12 : 0.1)}
          style={{
            position: "absolute",
            right: -28,
            bottom: -34,
            transform: "rotate(-18deg)",
          }}
        />
        <div style={{ position: "relative" }}>
          <div
            style={{
              color: p.dark ? hexToRgba(p.accent, 0.95) : p.accent,
              fontWeight: 700,
              fontSize: 13.5,
              marginBottom: 4,
            }}
          >
            {t("greet_morning")} 🐾
          </div>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 800,
              fontSize: 28,
              color: p.text,
              lineHeight: 1.05,
              marginBottom: 10,
            }}
          >
            {firstName}!
          </div>
          <div
            style={{
              color: p.text,
              fontSize: 14.5,
              fontWeight: 600,
              opacity: 0.85,
            }}
          >
            {today.length > 0
              ? `${t("today")}: ${meetCount(today.length, p.lang)}`
              : t("nothingToday")}
          </div>
        </div>
      </div>

      <div>
        <SectionTitle>{t("quick")}</SectionTitle>
        <div
          style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 11 }}
        >
          {actions.map((a) => (
            <button
              key={a.key}
              type="button"
              onClick={a.onClick}
              style={{
                background: p.card,
                border: `1px solid ${p.border}`,
                borderRadius: 18,
                padding: "15px 14px",
                cursor: "pointer",
                textAlign: "left",
                display: "flex",
                flexDirection: "column",
                gap: 11,
                boxShadow: p.shadowSm,
                transition: "transform .14s",
              }}
              onPointerDown={(e) => {
                e.currentTarget.style.transform = "scale(0.97)"
              }}
              onPointerUp={(e) => {
                e.currentTarget.style.transform = "scale(1)"
              }}
              onPointerLeave={(e) => {
                e.currentTarget.style.transform = "scale(1)"
              }}
            >
              <div
                style={{
                  width: 42,
                  height: 42,
                  borderRadius: 13,
                  background: `oklch(${p.dark ? 0.36 : 0.94} ${p.dark ? 0.08 : 0.06} ${a.hue})`,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                <CatIcon
                  name={a.icon}
                  size={23}
                  color={`oklch(${p.dark ? 0.82 : 0.52} 0.15 ${a.hue})`}
                  sw={2.1}
                />
              </div>
              <span
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 700,
                  fontSize: 14.5,
                  color: p.text,
                  lineHeight: 1.15,
                }}
              >
                {a.label}
              </span>
            </button>
          ))}
        </div>
      </div>

      <div>
        <SectionTitle
          action={today.length ? t("seeAll") : null}
          onAction={() => onTab("meetings")}
        >
          {t("today")}
        </SectionTitle>
        {today.length === 0 ? (
          <div
            style={{
              background: p.card,
              borderRadius: 20,
              border: `1px solid ${p.border}`,
              overflow: "hidden",
            }}
          >
            <EmptyState emoji="😴" title={t("nothingToday")} />
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
            {today.map((m) => (
              <MeetingCard key={m.id} m={m} onClick={() => onMeeting(m)} />
            ))}
          </div>
        )}
      </div>

      {upcoming.length > 0 && (
        <div>
          <SectionTitle action={t("seeAll")} onAction={() => onTab("meetings")}>
            {t("upcoming")}
          </SectionTitle>
          <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
            {upcoming.map((m) => (
              <MeetingCard key={m.id} m={m} onClick={() => onMeeting(m)} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
