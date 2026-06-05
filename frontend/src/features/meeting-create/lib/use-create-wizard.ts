import { useMemo, useState } from "react"
import { ME } from "@/shared/tma/mock-data"
import type { Meeting, MeetingDraft } from "@/shared/tma/types"
import { WIZARD_STEPS } from "./wizard-constants"

export function useCreateWizard({
  initial,
  meetings,
  onComplete,
}: {
  initial?: Partial<MeetingDraft>
  meetings: Meeting[]
  onComplete: (m: MeetingDraft & { end: string }) => void
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
    }[WIZARD_STEPS[step]] ?? false

  const go = (dir: number) => {
    if (dir > 0 && step === WIZARD_STEPS.length - 1) {
      onComplete({ ...draft, end: endTime })
      return
    }
    setStep((s) => Math.max(0, Math.min(WIZARD_STEPS.length - 1, s + dir)))
  }

  const conflictPeople = useMemo(() => {
    if (WIZARD_STEPS[step] !== "review" || !draft.date) return [] as string[]
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

  const finalMeeting = { ...draft, end: endTime, organizer: ME.email }

  return {
    step,
    draft,
    set,
    endTime,
    canNext,
    go,
    conflictPeople,
    finalMeeting,
    pSearch,
    setPSearch,
  }
}
