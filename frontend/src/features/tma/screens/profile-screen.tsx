import { useState, type ReactNode } from "react"
import { I18N } from "@/shared/tma/i18n"
import { EMPLOYEES } from "@/shared/tma/mock-data"
import { useColleagueSchedule, useEmployeeSearch } from "@/shared/tma/queries"
import { useTmaAuth } from "@/shared/tma/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import type { Meeting } from "@/shared/tma/types"
import {
  Avatar,
  CatCard,
  CatIcon,
  CatToggle,
  Segmented,
} from "@/shared/ui/cat/primitives"
import { EmptyState, MeetingCard } from "../meeting-ui"

function SettingsGroup({
  title,
  children,
}: {
  title?: string
  children: ReactNode
}) {
  const p = useTmaApp()
  return (
    <div style={{ marginBottom: 20 }}>
      {title && (
        <div
          style={{
            fontSize: 13,
            fontWeight: 800,
            color: p.muted,
            margin: "0 4px 9px",
            fontFamily: "var(--font-display)",
          }}
        >
          {title}
        </div>
      )}
      <CatCard pad={0} style={{ overflow: "hidden" }}>
        {children}
      </CatCard>
    </div>
  )
}

function Row({
  icon,
  hue = 45,
  label,
  right,
  onClick,
  last = false,
}: {
  icon: Parameters<typeof CatIcon>[0]["name"]
  hue?: number
  label: string
  right?: ReactNode
  onClick?: () => void
  last?: boolean
}) {
  const p = useTmaApp()
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        width: "100%",
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "13px 14px",
        border: "none",
        borderBottom: last ? "none" : `1px solid ${p.border}`,
        background: "transparent",
        cursor: onClick ? "pointer" : "default",
        textAlign: "left",
      }}
    >
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: 10,
          background: `oklch(${p.dark ? 0.36 : 0.94} ${p.dark ? 0.08 : 0.06} ${hue})`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <CatIcon
          name={icon}
          size={18}
          color={`oklch(${p.dark ? 0.82 : 0.52} 0.15 ${hue})`}
          sw={2}
        />
      </div>
      <span
        style={{
          flex: 1,
          fontWeight: 700,
          fontSize: 15,
          color: p.text,
          fontFamily: "var(--font-display)",
        }}
      >
        {label}
      </span>
      {right}
    </button>
  )
}

export function ProfileScreen({
  onLang,
  onAdmin,
  reminders,
  setReminders,
  remOn,
  setRemOn,
}: {
  onLang: () => void
  onAdmin: () => void
  reminders: string[]
  setReminders: (v: string[]) => void
  remOn: boolean
  setRemOn: (v: boolean) => void
}) {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const t = p.t
  const intervals = [
    { v: "10m", l: `10 ${t("min")}` },
    { v: "15m", l: `15 ${t("min")}` },
    { v: "30m", l: `30 ${t("min")}` },
    { v: "1h", l: `1 ${t("hour")}` },
    { v: "2h", l: `2 ${t("hour")}` },
    { v: "1d", l: "1 день" },
  ]

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 14,
          marginBottom: 22,
          padding: "4px 4px",
        }}
      >
        <Avatar name={user?.name ?? ""} size={62} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 800,
              fontSize: 20,
              color: p.text,
              lineHeight: 1.1,
            }}
          >
            {user?.name ?? ""}
          </div>
          <div
            style={{
              fontSize: 13,
              color: p.muted,
              marginBottom: 6,
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
          >
            {user?.email ?? ""}
          </div>
          {user?.role === "admin" && (
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 5,
                fontSize: 11.5,
                fontWeight: 800,
                color: p.accent,
                background: p.accentSoft,
                padding: "3px 9px",
                borderRadius: 999,
                fontFamily: "var(--font-display)",
              }}
            >
              👑 {t("role_admin")}
            </span>
          )}
        </div>
      </div>

      <SettingsGroup title={t("reminders")}>
        <Row
          icon="bell"
          hue={45}
          label={t("reminders")}
          right={<CatToggle on={remOn} onChange={setRemOn} />}
          last
        />
      </SettingsGroup>
      {remOn && (
        <div
          style={{
            margin: "-12px 4px 20px",
            display: "flex",
            flexWrap: "wrap",
            gap: 8,
          }}
        >
          {intervals.map((it) => {
            const on = reminders.includes(it.v)
            return (
              <button
                key={it.v}
                type="button"
                onClick={() =>
                  setReminders(
                    on
                      ? reminders.filter((x) => x !== it.v)
                      : [...reminders, it.v]
                  )
                }
                style={{
                  padding: "8px 13px",
                  borderRadius: 11,
                  border: `1.5px solid ${on ? p.accent : p.border}`,
                  background: on ? p.accentSoft : p.card,
                  color: on ? p.accent : p.muted,
                  fontWeight: 700,
                  fontSize: 13.5,
                  fontFamily: "var(--font-display)",
                  cursor: "pointer",
                }}
              >
                {on ? "✓ " : ""}
                {it.l}
              </button>
            )
          })}
        </div>
      )}

      <SettingsGroup title="Предпочтения">
        <Row
          icon="globe"
          hue={180}
          label={t("language")}
          onClick={onLang}
          right={
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                color: p.muted,
                fontWeight: 700,
                fontSize: 14,
              }}
            >
              {I18N[p.lang]._flag} {I18N[p.lang]._label}
              <CatIcon name="chevR" size={16} color={p.faint} sw={2.2} />
            </span>
          }
        />
        <Row
          icon="clock"
          hue={300}
          label={t("timezone")}
          right={
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                color: p.muted,
                fontWeight: 700,
                fontSize: 14,
              }}
            >
              Алматы UTC+5
              <CatIcon name="chevR" size={16} color={p.faint} sw={2.2} />
            </span>
          }
          last
        />
      </SettingsGroup>

      {user?.role === "admin" && (
        <SettingsGroup title={t("admin")}>
          <Row
            icon="shield"
            hue={25}
            label={t("admin")}
            onClick={onAdmin}
            right={<CatIcon name="chevR" size={18} color={p.faint} sw={2.2} />}
            last
          />
        </SettingsGroup>
      )}

      <SettingsGroup>
        <Row
          icon="sparkle"
          hue={95}
          label={t("help")}
          onClick={() => p.showToast("Мяу! Чем помочь? 🐾")}
          right={<CatIcon name="chevR" size={18} color={p.faint} sw={2.2} />}
          last
        />
      </SettingsGroup>

      <div
        style={{
          textAlign: "center",
          marginTop: 8,
          color: p.faint,
          fontSize: 12,
          fontWeight: 600,
        }}
      >
        Lead Cat · v2.0 · {I18N[p.lang].appSub} 🐾
      </div>
    </div>
  )
}

export function ColleagueSchedule() {
  const p = useTmaApp()
  const t = p.t
  const [picked, setPicked] = useState<{
    id: string
    name: string
    email: string
    dept?: string
  } | null>(null)
  const [search, setSearch] = useState("")

  const { data: searchResults = [] } = useEmployeeSearch(search)
  const matches = searchResults.slice(0, 6)

  const { data: colleagueMeetings = [], isLoading: loadingSchedule } =
    useColleagueSchedule(picked?.email ?? "", "all")

  if (picked) {
    const list = [...colleagueMeetings].sort((a, b) =>
      (a.date + a.start).localeCompare(b.date + b.start)
    )
    return (
      <div style={{ padding: "16px 16px 28px" }}>
        <button
          type="button"
          onClick={() => setPicked(null)}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 4,
            background: "none",
            border: "none",
            color: p.accent,
            fontWeight: 700,
            fontSize: 14,
            cursor: "pointer",
            marginBottom: 14,
            fontFamily: "var(--font-display)",
          }}
        >
          <CatIcon name="chevL" size={16} color={p.accent} sw={2.4} />{" "}
          {t("searchColleague")}
        </button>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 12,
            marginBottom: 8,
          }}
        >
          <Avatar name={picked.name} size={52} />
          <div>
            <div
              style={{
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 19,
                color: p.text,
              }}
            >
              {picked.name}
            </div>
            <div style={{ fontSize: 13, color: p.muted }}>{picked.dept}</div>
          </div>
        </div>
        <div
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 6,
            background: p.cardAlt,
            padding: "5px 11px",
            borderRadius: 999,
            fontSize: 12,
            fontWeight: 700,
            color: p.muted,
            marginBottom: 18,
          }}
        >
          👁️ {t("viewOnly")}
        </div>
        {loadingSchedule ? (
          <div
            style={{
              textAlign: "center",
              color: p.muted,
              fontSize: 14,
              padding: 20,
            }}
          >
            Загрузка…
          </div>
        ) : list.length === 0 ? (
          <div
            style={{
              background: p.card,
              borderRadius: 20,
              border: `1px solid ${p.border}`,
              overflow: "hidden",
            }}
          >
            <EmptyState
              emoji="😺"
              title="Свободен"
              sub="Нет встреч в расписании"
            />
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
            {list.map((m) => (
              <MeetingCard key={m.id} m={m} compact />
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <h2
        style={{
          margin: "0 0 6px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 23,
          color: p.text,
        }}
      >
        {t("searchColleague")}
      </h2>
      <p style={{ margin: "0 0 18px", color: p.muted, fontSize: 14 }}>
        Посмотри расписание любого сотрудника
      </p>
      <div style={{ position: "relative" }}>
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("searchPeople")}
          style={{
            width: "100%",
            boxSizing: "border-box",
            height: 50,
            padding: "0 14px 0 42px",
            borderRadius: 14,
            border: `1.5px solid ${p.border}`,
            background: p.dark ? p.cardAlt : "#fff",
            color: p.text,
            fontSize: 16,
          }}
          autoFocus
        />
        <CatIcon
          name="search"
          size={19}
          color={p.faint}
          style={{ position: "absolute", left: 13, top: 15 }}
          sw={2}
        />
      </div>
      {search === "" ? (
        <div
          style={{
            marginTop: 16,
            color: p.faint,
            fontSize: 14,
            textAlign: "center",
            padding: 20,
          }}
        >
          {t("typeToSearch")} 🐾
        </div>
      ) : (
        <div
          style={{
            marginTop: 12,
            display: "flex",
            flexDirection: "column",
            gap: 8,
          }}
        >
          {matches.map((e) => (
            <CatCard
              key={e.id}
              interactive
              onClick={() => {
                setPicked(e)
                setSearch("")
              }}
              pad={12}
              style={{ display: "flex", alignItems: "center", gap: 11 }}
            >
              <Avatar name={e.name} size={40} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontWeight: 700,
                    fontSize: 15,
                    color: p.text,
                    fontFamily: "var(--font-display)",
                  }}
                >
                  {e.name}
                </div>
                <div style={{ fontSize: 12.5, color: p.muted }}>
                  {e.dept} · {e.email}
                </div>
              </div>
              <CatIcon name="chevR" size={18} color={p.faint} sw={2.2} />
            </CatCard>
          ))}
        </div>
      )}
    </div>
  )
}

export function AdminPanel({ meetings }: { meetings: Meeting[] }) {
  const p = useTmaApp()
  const t = p.t
  const [tab, setTab] = useState<"meetings" | "users">("meetings")

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <div style={{ marginBottom: 18 }}>
        <Segmented
          value={tab}
          onChange={setTab}
          options={[
            {
              value: "meetings",
              label: t("allMeetings").split(" ")[0] ?? "Встречи",
            },
            { value: "users", label: t("users") },
          ]}
        />
      </div>
      {tab === "meetings" ? (
        <div>
          <div
            style={{
              fontSize: 13,
              fontWeight: 700,
              color: p.muted,
              margin: "0 4px 10px",
            }}
          >
            {meetings.length} {t("allMeetings").toLowerCase()}
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
            {[...meetings]
              .sort((a, b) =>
                (a.date + a.start).localeCompare(b.date + b.start)
              )
              .map((m) => (
                <MeetingCard key={m.id} m={m} compact />
              ))}
          </div>
        </div>
      ) : (
        <CatCard pad={6}>
          {EMPLOYEES.map((e, i) => (
            <div
              key={e.id}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 11,
                padding: "9px 8px",
                borderBottom:
                  i < EMPLOYEES.length - 1 ? `1px solid ${p.border}` : "none",
              }}
            >
              <Avatar name={e.name} size={38} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontWeight: 700,
                    fontSize: 14.5,
                    color: p.text,
                    fontFamily: "var(--font-display)",
                  }}
                >
                  {e.name}
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
                  {e.email}
                </div>
              </div>
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 800,
                  padding: "3px 9px",
                  borderRadius: 999,
                  fontFamily: "var(--font-display)",
                  color: e.id === "u1" ? p.accent : p.muted,
                  background: e.id === "u1" ? p.accentSoft : p.cardAlt,
                }}
              >
                {e.id === "u1" ? `👑 ${t("role_admin")}` : t("role_user")}
              </span>
            </div>
          ))}
        </CatCard>
      )}
    </div>
  )
}
