import { useState } from "react"
import { EMPLOYEES, FREE_SLOTS } from "@/shared/tma/mock-data"
import { useTmaApp } from "@/shared/tma/context"
import type { Employee, FreeSlot } from "@/shared/tma/types"
import {
  Avatar,
  CatBtn,
  CatCard,
  CatIcon,
  Field,
  inputStyle,
  Segmented,
} from "@/shared/ui/cat/primitives"
import { EmptyState } from "../meeting-ui"

export function CheckerScreen({
  onCreateFromSlot,
}: {
  onCreateFromSlot: (slot: FreeSlot, people: Employee[]) => void
}) {
  const p = useTmaApp()
  const t = p.t
  const [people, setPeople] = useState<Employee[]>([EMPLOYEES[1], EMPLOYEES[2]])
  const [search, setSearch] = useState("")
  const [range, setRange] = useState("7")
  const [dur, setDur] = useState(60)
  const [searched, setSearched] = useState(false)

  const matches = EMPLOYEES.filter((e) => {
    const q = search.toLowerCase()
    return (
      q &&
      (e.name.toLowerCase().includes(q) || e.email.toLowerCase().includes(q)) &&
      !people.find((x) => x.id === e.id)
    )
  }).slice(0, 5)

  const slots = FREE_SLOTS.filter((s) => s.mins >= dur)

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <h2
        style={{
          margin: "2px 4px 4px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 26,
          color: p.text,
        }}
      >
        {t("findTime")} 🔍
      </h2>
      <p style={{ margin: "0 4px 18px", color: p.muted, fontSize: 14 }}>
        Кот найдёт окно, когда все свободны
      </p>

      <Field label={`${t("addPeople")} · ${people.length}`}>
        <div style={{ position: "relative" }}>
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("searchPeople")}
            style={{ ...inputStyle(p), paddingLeft: 42 }}
          />
          <CatIcon
            name="search"
            size={19}
            color={p.faint}
            style={{ position: "absolute", left: 13, top: 15 }}
            sw={2}
          />
        </div>
        {matches.length > 0 && (
          <div
            style={{
              marginTop: 8,
              background: p.card,
              borderRadius: 14,
              border: `1px solid ${p.border}`,
              overflow: "hidden",
            }}
          >
            {matches.map((e, i) => (
              <button
                key={e.id}
                type="button"
                onClick={() => {
                  setPeople([...people, e])
                  setSearch("")
                  setSearched(false)
                }}
                style={{
                  width: "100%",
                  display: "flex",
                  alignItems: "center",
                  gap: 11,
                  padding: "10px 12px",
                  border: "none",
                  borderBottom:
                    i < matches.length - 1 ? `1px solid ${p.border}` : "none",
                  background: "transparent",
                  cursor: "pointer",
                  textAlign: "left",
                }}
              >
                <Avatar name={e.name} size={34} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      fontWeight: 700,
                      fontSize: 14,
                      color: p.text,
                      fontFamily: "var(--font-display)",
                    }}
                  >
                    {e.name}
                  </div>
                  <div style={{ fontSize: 12, color: p.muted }}>{e.dept}</div>
                </div>
                <CatIcon name="plus" size={18} color={p.accent} sw={2.4} />
              </button>
            ))}
          </div>
        )}
        {people.length > 0 && (
          <div
            style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 12 }}
          >
            {people.map((e, i) => (
              <span
                key={e.id}
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 7,
                  background: p.cardAlt,
                  borderRadius: 999,
                  padding: "5px 6px 5px 5px",
                  border: `1px solid ${p.border}`,
                }}
              >
                <Avatar name={e.name} size={24} />
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 700,
                    color: p.text,
                    fontFamily: "var(--font-display)",
                  }}
                >
                  {e.name.split(" ")[0]}
                </span>
                <button
                  type="button"
                  onClick={() => {
                    setPeople(people.filter((_, j) => j !== i))
                    setSearched(false)
                  }}
                  style={{
                    border: "none",
                    background: "none",
                    cursor: "pointer",
                    display: "flex",
                    padding: 2,
                  }}
                >
                  <CatIcon name="x" size={14} color={p.muted} sw={2.4} />
                </button>
              </span>
            ))}
          </div>
        )}
      </Field>

      <div style={{ height: 18 }} />
      <Field label={t("dateRange")}>
        <Segmented
          value={range}
          onChange={(v) => {
            setRange(v)
            setSearched(false)
          }}
          options={[
            { value: "7", label: "7 дн" },
            { value: "14", label: "14 дн" },
            { value: "30", label: "30 дн" },
          ]}
        />
      </Field>

      <div style={{ height: 16 }} />
      <Field label={t("duration")}>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          {[30, 45, 60, 90, 120].map((d) => {
            const active = d === dur
            return (
              <button
                key={d}
                type="button"
                onClick={() => {
                  setDur(d)
                  setSearched(false)
                }}
                style={{
                  padding: "9px 14px",
                  borderRadius: 11,
                  border: `1.5px solid ${active ? p.accent : p.border}`,
                  background: active ? p.accentSoft : p.card,
                  color: active ? p.accent : p.text,
                  fontWeight: 700,
                  fontSize: 14,
                  fontFamily: "var(--font-display)",
                  cursor: "pointer",
                }}
              >
                {d < 60
                  ? `${d} ${t("min")}`
                  : d === 60
                    ? `1 ${t("hour")}`
                    : `${d / 60} ${t("hour")}`}
              </button>
            )
          })}
        </div>
      </Field>

      <div style={{ height: 22 }} />
      <CatBtn
        variant="primary"
        full
        size="lg"
        disabled={people.length === 0}
        onClick={() => setSearched(true)}
        icon={<CatIcon name="search" size={20} color={p.accentText} sw={2.2} />}
      >
        {t("findSlots")}
      </CatBtn>

      {searched && (
        <div style={{ marginTop: 22 }}>
          {slots.length === 0 ? (
            <div
              style={{
                background: p.card,
                borderRadius: 20,
                border: `1px solid ${p.border}`,
                overflow: "hidden",
              }}
            >
              <EmptyState
                emoji="🙀"
                title={t("noSlots")}
                sub="Попробуй расширить диапазон или уменьшить длительность"
              />
            </div>
          ) : (
            <div>
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 7,
                  color: p.ok,
                  fontWeight: 800,
                  fontFamily: "var(--font-display)",
                  fontSize: 15,
                  margin: "0 4px 12px",
                }}
              >
                ✅ {t("freeFor")} {people.length} {t("people")}
              </div>
              <div
                style={{ display: "flex", flexDirection: "column", gap: 10 }}
              >
                {slots.map((s, i) => (
                  <CatCard
                    key={i}
                    interactive
                    onClick={() => onCreateFromSlot(s, people)}
                    pad={14}
                    style={{ display: "flex", alignItems: "center", gap: 13 }}
                  >
                    <div
                      style={{
                        width: 46,
                        height: 46,
                        borderRadius: 13,
                        background: p.okSoft,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        justifyContent: "center",
                        flexShrink: 0,
                      }}
                    >
                      <span style={{ fontSize: 16 }}>📅</span>
                    </div>
                    <div style={{ flex: 1 }}>
                      <div
                        style={{
                          fontFamily: "var(--font-display)",
                          fontWeight: 800,
                          fontSize: 15.5,
                          color: p.text,
                        }}
                      >
                        {s.start} – {s.end}
                      </div>
                      <div
                        style={{
                          fontSize: 13,
                          color: p.muted,
                          fontWeight: 600,
                        }}
                      >
                        {s.day} · {s.mins} {t("min")}
                      </div>
                    </div>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 4,
                        color: p.accent,
                        fontWeight: 700,
                        fontSize: 13,
                        fontFamily: "var(--font-display)",
                      }}
                    >
                      {t("createHere")}
                      <CatIcon
                        name="chevR"
                        size={15}
                        color={p.accent}
                        sw={2.4}
                      />
                    </div>
                  </CatCard>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
