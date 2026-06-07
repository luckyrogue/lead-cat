import { useEffect, useMemo, useState } from "react"
import { ME } from "@/shared/tma/mock-data"
import type { MeetingDraft } from "@/shared/tma/types"
import { useConflicts } from "@/features/meetings/queries"
import { WIZARD_STEPS } from "./wizard-constants"

export function useCreateWizard({
  initial,
  onComplete,
}: {
  initial?: Partial<MeetingDraft>
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
    })
    // intentionally NOT including conflictsMut in deps — useMutation's mutate is stable
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, draft.date, draft.start, endTime, draft.participants])

  const conflictPeople = useMemo(() => {
    const list = conflictsMut.data ?? []
    const names = new Set<string>()
    list.forEach((c) => {
      const parts = c.name.split(" ")
      names.add(parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : ""))
    })
    return [...names]
  }, [conflictsMut.data])

  const recurringBlocked = draft.rec !== "once"

  const finalMeeting = { ...draft, end: endTime, organizer: ME.email }

  return {
    step,
    draft,
    set,
    endTime,
    canNext,
    go,
    conflictPeople,
    recurringBlocked,
    finalMeeting,
    pSearch,
    setPSearch,
  }
}
