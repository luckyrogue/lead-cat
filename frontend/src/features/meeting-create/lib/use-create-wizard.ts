import { useEffect, useMemo, useState } from "react"
import { ME } from "@/entities/employee/fixtures"
import type { MeetingDraft } from "@/shared/miniapp/types"
import { useConflicts } from "@/entities/meeting/mutations"
import { WIZARD_STEPS } from "./wizard-constants"

function defaultUntil(date: string, rec: string): string {
  if (!date || rec === "once") return ""
  const [y, m, d] = date.split("-").map(Number)
  const dt = new Date(y, m - 1, d)
  switch (rec) {
    case "daily":
      dt.setDate(dt.getDate() + 30)
      break
    case "weekly":
    case "custom":
      dt.setDate(dt.getDate() + 7 * 12)
      break
    case "monthly":
      dt.setMonth(dt.getMonth() + 12)
      break
    default:
      return ""
  }
  return `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, "0")}-${String(dt.getDate()).padStart(2, "0")}`
}

export function useCreateWizard({
  initial,
  onComplete,
  lockedFields,
}: {
  initial?: Partial<MeetingDraft>
  onComplete: (m: MeetingDraft & { end: string }) => void
  lockedFields?: {
    date?: boolean
    rec?: boolean
    until?: boolean
    participants?: boolean
  }
}) {
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
    until: "",
    participants: [],
    desc: "",
    ...initial,
  }))
  const [pSearch, setPSearch] = useState("")

  const set = <K extends keyof MeetingDraft>(k: K, v: MeetingDraft[K]) =>
    setDraft((d) => {
      const nd = { ...d, [k]: v }
      if (k === "rec") {
        nd.until = defaultUntil(nd.date, nd.rec)
      } else if (k === "date" && d.rec !== "once" && !d.until) {
        nd.until = defaultUntil(nd.date, nd.rec)
      }
      return nd
    })

  const endTime = useMemo(() => {
    const [h, mn] = draft.start.split(":").map(Number)
    const total = h * 60 + mn + draft.dur
    return `${String(Math.floor(total / 60) % 24).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`
  }, [draft.start, draft.dur])

  const canNext = (() => {
    const step_ = WIZARD_STEPS[step]
    if (step_ === "what") return Boolean(draft.dept && draft.type)
    if (step_ === "when") {
      if (!draft.date || !draft.start) return false
      if (draft.rec !== "once") {
        if (!draft.until) return false
        if (draft.until < draft.date) return false
        if (draft.rec === "custom" && draft.recDays.length === 0) return false
      }
      return true
    }
    if (step_ === "who") return Boolean(draft.host)
    if (step_ === "review") return true
    return false
  })()

  const go = (dir: number) => {
    if (dir > 0 && step === WIZARD_STEPS.length - 1) {
      onComplete({ ...draft, end: endTime })
      return
    }
    setStep((s) => Math.max(0, Math.min(WIZARD_STEPS.length - 1, s + dir)))
  }

  const conflictsMut = useConflicts()

  useEffect(() => {
    if (WIZARD_STEPS[step] !== "review") return
    if (!draft.date || !draft.start || !endTime) return
    if (!draft.participants.length) return
    const emails = draft.participants.map((p) => p.email)
    conflictsMut.mutate({
      participants: emails,
      date: draft.date,
      start: draft.start,
      end: endTime,
      recurrence: draft.rec !== "once" ? draft.rec : undefined,
      recurrenceUntil: draft.rec !== "once" ? draft.until : undefined,
      recurrenceDays: draft.rec === "custom" ? draft.recDays : undefined,
    })
    // intentionally NOT including conflictsMut in deps — useMutation's mutate is stable
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    step,
    draft.date,
    draft.start,
    endTime,
    draft.participants,
    draft.rec,
    draft.until,
    draft.recDays,
  ])

  const conflictOccurrences = conflictsMut.data ?? []

  const finalMeeting = { ...draft, end: endTime, organizer: ME.email }

  return {
    step,
    draft,
    set,
    endTime,
    canNext,
    go,
    conflictOccurrences,
    finalMeeting,
    pSearch,
    setPSearch,
    lockedFields: lockedFields ?? {},
  }
}
