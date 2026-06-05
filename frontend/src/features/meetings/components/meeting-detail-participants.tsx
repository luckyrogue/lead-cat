import { useTmaApp } from "@/shared/tma/context"
import { emailsToPeople } from "@/shared/tma/mock-data"
import { partWord } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/entities/meeting/types"
import { Avatar, CatCard } from "@/shared/ui/cat/primitives"

export function MeetingDetailParticipants({ m }: { m: Meeting }) {
  const p = useTmaApp()
  const t = p.t
  const allPeople = emailsToPeople([m.organizer, ...m.participants])

  return (
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
  )
}
