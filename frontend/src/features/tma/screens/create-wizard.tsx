import { useMemo, useState } from "react"
import { TMA_NOW, WEEKDAYS, typeAccent } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import {
  DEPARTMENTS,
  EMPLOYEES,
  ME,
  MEETING_TYPES,
  RECURRENCE,
} from "@/shared/tma/mock-data"
import { buildTitle } from "@/shared/tma/meeting-utils"
import type { Meeting, MeetingDraft } from "@/shared/tma/types"
import {
  Avatar,
  CatBtn,
  CatCard,
  CatIcon,
  Field,
  inputStyle,
} from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"
import { hexToRgba } from "@/shared/tma/palette"
import { DetailRow } from "./meetings-screen"
import { fmtDate, ParticipantStack } from "../meeting-ui"

function MiniCalendar({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const p = useTmaApp()
  const [view, setView] = useState(() => {
    const [y, m] = (value || TMA_NOW).split("-").map(Number)
    return { y, m: m - 1 }
  })
  const monthNames = [
    "Январь",
    "Февраль",
    "Март",
    "Апрель",
    "Май",
    "Июнь",
    "Июль",
    "Август",
    "Сентябрь",
    "Октябрь",
    "Ноябрь",
    "Декабрь",
  ]
  const first = new Date(view.y, view.m, 1)
  const startDow = (first.getDay() + 6) % 7
  const days = new Date(view.y, view.m + 1, 0).getDate()
  const cells: (number | null)[] = []
  for (let i = 0; i < startDow; i++) cells.push(null)
  for (let d = 1; d <= days; d++) cells.push(d)
  const iso = (d: number) =>
    `${view.y}-${String(view.m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`
  const nav = (dir: number) =>
    setView((v) => {
      let m = v.m + dir
      let y = v.y
      if (m < 0) {
        m = 11
        y--
      }
      if (m > 11) {
        m = 0
        y++
      }
      return { y, m }
    })

  return (
    <div
      style={{
        background: p.card,
        borderRadius: 18,
        border: `1px solid ${p.border}`,
        padding: 14,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 12,
        }}
      >
        <button type="button" onClick={() => nav(-1)} style={calNav(p)}>
          <CatIcon name="chevL" size={18} color={p.text} sw={2.2} />
        </button>
        <span
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 16,
            color: p.text,
          }}
        >
          {monthNames[view.m]} {view.y}
        </span>
        <button type="button" onClick={() => nav(1)} style={calNav(p)}>
          <CatIcon name="chevR" size={18} color={p.text} sw={2.2} />
        </button>
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7,1fr)",
          gap: 4,
          marginBottom: 6,
        }}
      >
        {WEEKDAYS.map((w) => (
          <div
            key={w}
            style={{
              textAlign: "center",
              fontSize: 11,
              fontWeight: 700,
              color: p.faint,
            }}
          >
            {w}
          </div>
        ))}
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7,1fr)",
          gap: 4,
        }}
      >
        {cells.map((d, i) => {
          if (!d) return <div key={i} />
          const dIso = iso(d)
          const active = dIso === value
          const isPast = dIso < TMA_NOW
          const isToday = dIso === TMA_NOW
          return (
            <button
              key={i}
              type="button"
              onClick={() => onChange(dIso)}
              style={{
                aspectRatio: "1",
                borderRadius: 11,
                border: "none",
                cursor: "pointer",
                background: active ? p.accent : "transparent",
                color: active ? p.accentText : isPast ? p.faint : p.text,
                fontWeight: active || isToday ? 800 : 600,
                fontSize: 14,
                fontFamily: "var(--font-display)",
                position: "relative",
              }}
            >
              {d}
              {isToday && !active && (
                <span
                  style={{
                    position: "absolute",
                    bottom: 5,
                    left: "50%",
                    transform: "translateX(-50%)",
                    width: 4,
                    height: 4,
                    borderRadius: 2,
                    background: p.accent,
                  }}
                />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function calNav(p: ReturnType<typeof useTmaApp>) {
  return {
    width: 32,
    height: 32,
    borderRadius: 10,
    border: "none",
    background: p.cardAlt,
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  } as const
}

function ChipGrid<T extends string>({
  options,
  value,
  onChange,
  cols = 2,
}: {
  options: { value: T; label: string; emoji?: string }[]
  value: T
  onChange: (v: T) => void
  cols?: number
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: `repeat(${cols},1fr)`,
        gap: 9,
      }}
    >
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              padding: "13px 13px",
              borderRadius: 14,
              border: `1.5px solid ${active ? p.accent : p.border}`,
              cursor: "pointer",
              textAlign: "left",
              background: active ? p.accentSoft : p.card,
              color: p.text,
            }}
          >
            {o.emoji && <span style={{ fontSize: 18 }}>{o.emoji}</span>}
            <span
              style={{
                fontFamily: "var(--font-display)",
                fontWeight: 700,
                fontSize: 14,
                lineHeight: 1.1,
              }}
            >
              {o.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}

const STEPS = ["what", "when", "who", "review"] as const

export function CreateWizard({
  initial,
  onComplete,
  meetings,
}: {
  initial?: Partial<MeetingDraft>
  onComplete: (m: MeetingDraft & { end: string }) => void
  meetings: Meeting[]
}) {
  const p = useTmaApp()
  const t = p.t
  const [step, setStep] = useState(0)
  const [draft, setDraft] = useState<MeetingDraft>(() => ({
    dept: "",
    type: "",
    host: ME.name,
    date: "",
    start: "10:00",
    dur: 30,
    rec: "once",
    recDays: [],
    participants: [],
    desc: "",
    ...initial,
  }))
  const [pSearch, setPSearch] = useState("")
  const set = <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) =>
    setDraft((d) => ({ ...d, [k]: v }))

  const endTime = useMemo(() => {
    const [h, mn] = draft.start.split(":").map(Number)
    const total = h * 60 + mn + draft.dur
    return `${String(Math.floor(total / 60) % 24).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`
  }, [draft.start, draft.dur])

  const canNext =
    {
      what: Boolean(draft.dept && draft.type),
      when: Boolean(draft.date && draft.start),
      who: Boolean(draft.host),
      review: true,
    }[STEPS[step]] ?? false

  const go = (dir: number) => {
    if (dir > 0 && step === STEPS.length - 1) {
      onComplete({ ...draft, end: endTime })
      return
    }
    setStep((s) => Math.max(0, Math.min(STEPS.length - 1, s + dir)))
  }

  const conflictPeople = useMemo(() => {
    if (STEPS[step] !== "review" || !draft.date) return [] as string[]
    const overlaps = (s1: string, e1: string, s2: string, e2: string) =>
      s1 < e2 && s2 < e1
    const names = new Set<string>()
    draft.participants.forEach((pp) => {
      meetings.forEach((m) => {
        if (
          m.date === draft.date &&
          (m.organizer === pp.email || m.participants.includes(pp.email)) &&
          overlaps(draft.start, endTime, m.start, m.end)
        ) {
          const parts = pp.name.split(" ")
          names.add(parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : ""))
        }
      })
    })
    return [...names]
  }, [step, draft, endTime, meetings])

  const matches = EMPLOYEES.filter((e) => {
    const q = pSearch.toLowerCase()
    return (
      q &&
      (e.name.toLowerCase().includes(q) || e.email.toLowerCase().includes(q)) &&
      !draft.participants.find((x) => x.email === e.email)
    )
  }).slice(0, 5)

  const times: string[] = []
  for (let h = 8; h <= 19; h++) {
    for (const mn of [0, 30]) {
      times.push(`${String(h).padStart(2, "0")}:${String(mn).padStart(2, "0")}`)
    }
  }

  const finalMeeting = { ...draft, end: endTime, organizer: ME.email }

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        minHeight: 0,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: 7,
          padding: "14px 16px 6px",
          alignItems: "center",
        }}
      >
        {STEPS.map((s, i) => (
          <div
            key={s}
            style={{ flex: 1, display: "flex", alignItems: "center", gap: 7 }}
          >
            <Paw
              size={i <= step ? 20 : 16}
              color={i <= step ? p.accent : p.border}
              style={{ transition: "all .25s", flexShrink: 0 }}
            />
            {i < STEPS.length - 1 && (
              <div
                style={{
                  flex: 1,
                  height: 3,
                  borderRadius: 2,
                  background: i < step ? p.accent : p.border,
                  transition: "background .25s",
                }}
              />
            )}
          </div>
        ))}
      </div>

      <div
        className="lc-scroll"
        style={{ flex: 1, overflow: "auto", padding: "10px 16px 16px" }}
      >
        {STEPS[step] === "what" && (
          <div>
            <h2
              style={{
                margin: "0 0 18px",
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 23,
                color: p.text,
              }}
            >
              🗂️ О чём встреча?
            </h2>
            <Field label={t("department")}>
              <ChipGrid
                value={draft.dept}
                onChange={(v) => set("dept", v)}
                options={DEPARTMENTS.map((d) => ({ value: d, label: d }))}
              />
            </Field>
            <div style={{ height: 20 }} />
            <Field label={t("mType")}>
              <ChipGrid
                value={draft.type}
                onChange={(v) => set("type", v)}
                options={MEETING_TYPES.map((m) => ({
                  value: m.key,
                  label: m.label,
                  emoji: typeAccent(m.key, p.dark).emoji,
                }))}
              />
            </Field>
          </div>
        )}

        {STEPS[step] === "when" && (
          <div>
            <h2
              style={{
                margin: "0 0 18px",
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 23,
                color: p.text,
              }}
            >
              🗓️ Когда встречаемся?
            </h2>
            <Field label={t("dateT")}>
              <MiniCalendar
                value={draft.date}
                onChange={(v) => set("date", v)}
              />
            </Field>
            <div style={{ height: 18 }} />
            <Field
              label={`${t("timeT")} — ${t("from")} ${draft.start}, ${t("to")} ${endTime}`}
            >
              <div
                className="lc-scroll"
                style={{
                  display: "flex",
                  gap: 8,
                  overflowX: "auto",
                  paddingBottom: 4,
                }}
              >
                {times.map((tm) => {
                  const active = tm === draft.start
                  return (
                    <button
                      key={tm}
                      type="button"
                      onClick={() => set("start", tm)}
                      style={{
                        flexShrink: 0,
                        padding: "9px 13px",
                        borderRadius: 11,
                        border: `1.5px solid ${active ? p.accent : p.border}`,
                        background: active ? p.accent : p.card,
                        color: active ? p.accentText : p.text,
                        fontWeight: 700,
                        fontSize: 14,
                        fontFamily: "var(--font-display)",
                        cursor: "pointer",
                      }}
                    >
                      {tm}
                    </button>
                  )
                })}
              </div>
            </Field>
            <div style={{ height: 16 }} />
            <Field label={t("duration")}>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                {[15, 30, 45, 60, 90, 120].map((d) => {
                  const active = d === draft.dur
                  return (
                    <button
                      key={d}
                      type="button"
                      onClick={() => set("dur", d)}
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
            <div style={{ height: 18 }} />
            <Field label={t("recur")}>
              <ChipGrid
                value={draft.rec}
                onChange={(v) => set("rec", v)}
                cols={2}
                options={RECURRENCE.map((r) => ({
                  value: r.key,
                  label: r.label,
                }))}
              />
              {draft.rec === "custom" && (
                <div style={{ display: "flex", gap: 7, marginTop: 11 }}>
                  {WEEKDAYS.map((w, i) => {
                    const on = draft.recDays.includes(i + 1)
                    return (
                      <button
                        key={w}
                        type="button"
                        onClick={() =>
                          set(
                            "recDays",
                            on
                              ? draft.recDays.filter((x) => x !== i + 1)
                              : [...draft.recDays, i + 1]
                          )
                        }
                        style={{
                          flex: 1,
                          aspectRatio: "1",
                          borderRadius: 11,
                          border: `1.5px solid ${on ? p.accent : p.border}`,
                          background: on ? p.accent : p.card,
                          color: on ? p.accentText : p.muted,
                          fontWeight: 800,
                          fontSize: 12.5,
                          fontFamily: "var(--font-display)",
                          cursor: "pointer",
                        }}
                      >
                        {w}
                      </button>
                    )
                  })}
                </div>
              )}
            </Field>
          </div>
        )}

        {STEPS[step] === "who" && (
          <div>
            <h2
              style={{
                margin: "0 0 18px",
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 23,
                color: p.text,
              }}
            >
              🧑‍🤝‍🧑 Кто участвует?
            </h2>
            <Field label={t("host")}>
              <div
                style={{
                  ...inputStyle(p),
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  height: 54,
                }}
              >
                <Avatar name={draft.host} size={32} />
                <span
                  style={{
                    fontWeight: 700,
                    fontFamily: "var(--font-display)",
                    color: p.text,
                  }}
                >
                  {draft.host}
                </span>
                <span
                  style={{
                    marginLeft: "auto",
                    fontSize: 11,
                    fontWeight: 700,
                    color: p.accent,
                    background: p.accentSoft,
                    padding: "3px 8px",
                    borderRadius: 999,
                  }}
                >
                  я
                </span>
              </div>
            </Field>
            <div style={{ height: 18 }} />
            <Field label={t("addPeople")}>
              <div style={{ position: "relative" }}>
                <input
                  value={pSearch}
                  onChange={(e) => setPSearch(e.target.value)}
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
                        set("participants", [...draft.participants, e])
                        setPSearch("")
                      }}
                      style={{
                        width: "100%",
                        display: "flex",
                        alignItems: "center",
                        gap: 11,
                        padding: "10px 12px",
                        border: "none",
                        borderBottom:
                          i < matches.length - 1
                            ? `1px solid ${p.border}`
                            : "none",
                        background: "transparent",
                        cursor: "pointer",
                        textAlign: "left",
                      }}
                    >
                      <Avatar name={e.name} size={34} />
                      <div style={{ flex: 1 }}>
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
                        <div style={{ fontSize: 12, color: p.muted }}>
                          {e.dept} · {e.email}
                        </div>
                      </div>
                      <CatIcon
                        name="plus"
                        size={18}
                        color={p.accent}
                        sw={2.4}
                      />
                    </button>
                  ))}
                </div>
              )}
              {draft.participants.length > 0 && (
                <div
                  style={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: 8,
                    marginTop: 12,
                  }}
                >
                  {draft.participants.map((e, i) => (
                    <span
                      key={i}
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
                        onClick={() =>
                          set(
                            "participants",
                            draft.participants.filter((_, j) => j !== i)
                          )
                        }
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
            <Field label={t("descr")}>
              <textarea
                value={draft.desc}
                onChange={(e) => set("desc", e.target.value)}
                rows={3}
                placeholder="—"
                style={{
                  ...inputStyle(p),
                  height: "auto",
                  padding: "12px 14px",
                  resize: "none",
                }}
              />
            </Field>
          </div>
        )}

        {STEPS[step] === "review" && (
          <div>
            <h2
              style={{
                margin: "0 0 18px",
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 23,
                color: p.text,
              }}
            >
              ✨ {t("preview")}
            </h2>
            <div
              style={{
                background: p.cardAlt,
                borderRadius: 16,
                padding: "14px 15px",
                marginBottom: 14,
                border: `1px dashed ${p.borderStrong}`,
              }}
            >
              <div
                style={{
                  fontSize: 11,
                  color: p.faint,
                  fontWeight: 700,
                  marginBottom: 5,
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
                }}
              >
                {buildTitle({ ...finalMeeting, type: draft.type })}
              </div>
            </div>
            <CatCard>
              <DetailRow icon="calendar" label={t("dateT")}>
                {fmtDate(draft.date, p.lang)}
              </DetailRow>
              <DetailRow icon="clock" label={t("timeT")}>
                {draft.start} – {endTime} · UTC+5
              </DetailRow>
              <DetailRow icon="user" label={t("host")}>
                {draft.host}
              </DetailRow>
              <div
                style={{
                  display: "flex",
                  gap: 12,
                  padding: "12px 0 0",
                  alignItems: "center",
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
                  <CatIcon name="users" size={18} color={p.accent} sw={2} />
                </div>
                <div style={{ flex: 1 }}>
                  <div
                    style={{
                      fontSize: 12,
                      color: p.muted,
                      fontWeight: 600,
                      marginBottom: 4,
                    }}
                  >
                    {t("addPeople")}
                  </div>
                  {draft.participants.length ? (
                    <ParticipantStack
                      emails={draft.participants.map((x) => x.email)}
                      max={6}
                    />
                  ) : (
                    <span style={{ color: p.faint, fontSize: 14 }}>—</span>
                  )}
                </div>
              </div>
            </CatCard>
            {conflictPeople.length > 0 && (
              <div
                style={{
                  marginTop: 14,
                  background: p.dangerSoft,
                  borderRadius: 16,
                  padding: "13px 15px",
                  border: `1px solid ${hexToRgba(p.danger, 0.3)}`,
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 7,
                    color: p.danger,
                    fontWeight: 800,
                    fontFamily: "var(--font-display)",
                    fontSize: 14.5,
                    marginBottom: 6,
                  }}
                >
                  ⚠️ {t("conflict")}
                </div>
                <div
                  style={{
                    fontSize: 13,
                    color: p.text,
                    opacity: 0.85,
                    lineHeight: 1.4,
                  }}
                >
                  {t("conflictSub")} {conflictPeople.join(", ")}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      <div
        style={{
          flexShrink: 0,
          padding: "12px 16px max(12px, var(--tma-safe-bottom, 0px))",
          borderTop: `1px solid ${p.border}`,
          background: p.tgBar,
          display: "flex",
          gap: 10,
        }}
      >
        {step > 0 && (
          <CatBtn
            variant="outline"
            onClick={() => go(-1)}
            icon={<CatIcon name="chevL" size={18} color={p.text} sw={2.2} />}
          >
            {t("back")}
          </CatBtn>
        )}
        <CatBtn
          variant="primary"
          full
          disabled={!canNext}
          onClick={() => go(1)}
        >
          {step === STEPS.length - 1
            ? conflictPeople.length
              ? `🐾 ${t("proceed")}`
              : `🐾 ${t("confirmCreate")}`
            : t("next")}
        </CatBtn>
      </div>
    </div>
  )
}
