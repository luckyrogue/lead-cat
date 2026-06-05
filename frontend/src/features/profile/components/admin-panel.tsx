import { useState } from "react"
import { EMPLOYEES } from "@/shared/tma/mock-data"
import { useTmaApp } from "@/shared/tma/context"
import type { Meeting } from "@/entities/meeting/types"
import { Avatar, CatCard, Segmented } from "@/shared/ui/cat/primitives"
import { MeetingCard } from "@/features/meetings/components/meeting-ui"

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
