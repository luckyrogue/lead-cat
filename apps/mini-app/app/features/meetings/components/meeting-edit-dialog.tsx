import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  toast,
} from "@leadcat/ui"
import { useEffect, useState } from "react"

import { useUpdateMeeting } from "~/entities/meeting/mutations"
import type { Meeting } from "~/entities/meeting/types"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting: Meeting
}

export function MeetingEditDialog({ open, onOpenChange, meeting }: Props) {
  const update = useUpdateMeeting()
  const [type, setType] = useState(meeting.type)
  const [date, setDate] = useState(meeting.date)
  const [start, setStart] = useState(meeting.start)
  const [end, setEnd] = useState(meeting.end)
  const [desc, setDesc] = useState(meeting.desc)

  useEffect(() => {
    if (open) {
      setType(meeting.type)
      setDate(meeting.date)
      setStart(meeting.start)
      setEnd(meeting.end)
      setDesc(meeting.desc)
    }
  }, [open, meeting])

  function onSave() {
    update.mutate(
      {
        id: meeting.id,
        scope: "this",
        input: { type, date, start, end, desc },
      },
      {
        onSuccess: () => {
          toast.success("Meeting updated")
          onOpenChange(false)
        },
        onError: () => toast.error("Couldn't update meeting"),
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Edit meeting</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <Field label="Title">
            <Input value={type} onChange={(e) => setType(e.target.value)} />
          </Field>
          <Field label="Date">
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Start">
              <Input type="time" value={start} onChange={(e) => setStart(e.target.value)} />
            </Field>
            <Field label="End">
              <Input type="time" value={end} onChange={(e) => setEnd(e.target.value)} />
            </Field>
          </div>
          <Field label="Description">
            <Input value={desc} onChange={(e) => setDesc(e.target.value)} />
          </Field>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onSave} disabled={update.isPending}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
