import { Button, Input, Label, toast } from "@leadcat/ui"
import { useState } from "react"

import { EmployeePicker } from "~/components/employee-picker"
import { PageHeader } from "~/components/page-header"
import { EmptyState, LoadingState } from "~/components/states"
import { FreeSlotList } from "~/features/checker/components/free-slot-list"
import { fetchFreeSlots } from "~/entities/meeting/api"
import type { Employee } from "~/entities/employee/types"
import type { FreeSlot } from "~/entities/meeting/types"
import { addDaysIso, todayIso } from "~/shared/lib/format"

export function CheckerPage() {
  const [participants, setParticipants] = useState<Employee[]>([])
  const [from, setFrom] = useState(todayIso())
  const [to, setTo] = useState(addDaysIso(todayIso(), 6))
  const [duration, setDuration] = useState(30)
  const [slots, setSlots] = useState<FreeSlot[] | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  async function onCheck() {
    if (participants.length === 0) {
      toast.error("Add at least one participant")
      return
    }
    if (to < from || duration <= 0) {
      toast.error("Check the date range and duration")
      return
    }
    setLoading(true)
    setSlots(undefined)
    try {
      const result = await fetchFreeSlots({
        participants: participants.map((p) => p.email),
        from,
        to,
        duration_mins: duration,
      })
      setSlots(result)
    } catch {
      toast.error("Couldn't find free slots")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title="Slot checker" subtitle="Find common free time" />

      <div className="flex flex-col gap-1.5">
        <Label>Participants</Label>
        <EmployeePicker selected={participants} onChange={setParticipants} />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <Field label="From">
          <Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
        </Field>
        <Field label="To">
          <Input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
        </Field>
      </div>

      <Field label="Duration (minutes)">
        <Input
          type="number"
          min={5}
          step={5}
          value={duration}
          onChange={(e) => setDuration(Number(e.target.value))}
        />
      </Field>

      <Button onClick={onCheck} disabled={loading}>
        Find free slots
      </Button>

      {loading ? (
        <LoadingState label="Searching…" />
      ) : slots === undefined ? null : slots.length === 0 ? (
        <EmptyState title="No common free slots" hint="Try a wider range or shorter duration." />
      ) : (
        <FreeSlotList slots={slots} />
      )}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
