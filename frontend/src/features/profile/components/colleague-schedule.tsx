import { useState } from "react"
import { useEmployeeSearch } from "@/features/checker/queries"
import { useColleagueSchedule } from "@/features/meetings/queries"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar, CatCard, CatIcon } from "@/shared/ui/cat/primitives"
import {
  EmptyState,
  MeetingCard,
} from "@/features/meetings/components/meeting-ui"

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
