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

import {
  useChangeSeriesEnd,
  useUpdateMeeting,
} from "~/entities/meeting/mutations"
import {
  isSeriesMeeting,
  type Meeting,
  type MeetingMutationScope,
} from "~/entities/meeting/types"
import { ParticipantsEditor } from "~/features/meetings/components/participants-editor"
import { ScopeToggle } from "~/features/meetings/components/scope-toggle"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting: Meeting
}

export function MeetingEditDialog({ open, onOpenChange, meeting }: Props) {
  const update = useUpdateMeeting()
  const changeEnd = useChangeSeriesEnd()
  const [type, setType] = useState(meeting.type)
  const [date, setDate] = useState(meeting.date)
  const [start, setStart] = useState(meeting.start)
  const [end, setEnd] = useState(meeting.end)
  const [desc, setDesc] = useState(meeting.desc)
  const [scope, setScope] = useState<MeetingMutationScope>("this")
  const [seriesUntil, setSeriesUntil] = useState(
    (meeting.recurrence_until ?? "").slice(0, 10)
  )

  const series = isSeriesMeeting(meeting)
  const lockDate = series && scope === "whole"

  useEffect(() => {
    if (open) {
      setType(meeting.type)
      setDate(meeting.date)
      setStart(meeting.start)
      setEnd(meeting.end)
      setDesc(meeting.desc)
      setScope("this")
      setSeriesUntil((meeting.recurrence_until ?? "").slice(0, 10))
    }
  }, [open, meeting])

  function onSave() {
    update.mutate(
      {
        id: meeting.id,
        scope,
        input: {
          type,
          start,
          end,
          desc,
          ...(lockDate ? {} : { date }),
        },
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
          {series ? <ScopeToggle value={scope} onChange={setScope} /> : null}
          {series ? (
            <div className="flex flex-col gap-1.5">
              <Field label="Series ends">
                <Input
                  type="date"
                  value={seriesUntil}
                  onChange={(e) => setSeriesUntil(e.target.value)}
                />
              </Field>
              <Button
                type="button"
                variant="ghost"
                disabled={changeEnd.isPending || !seriesUntil}
                onClick={() =>
                  changeEnd.mutate(
                    { id: meeting.id, until: seriesUntil },
                    {
                      onSuccess: () => toast.success("Series end updated"),
                      onError: () => toast.error("Couldn't update series end"),
                    }
                  )
                }
              >
                Update end date
              </Button>
            </div>
          ) : null}
          <Field label="Title">
            <Input value={type} onChange={(e) => setType(e.target.value)} />
          </Field>
          <Field label={lockDate ? "Date (locked for series)" : "Date"}>
            <Input
              type="date"
              value={date}
              disabled={lockDate}
              onChange={(e) => setDate(e.target.value)}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Start">
              <Input
                type="time"
                value={start}
                onChange={(e) => setStart(e.target.value)}
              />
            </Field>
            <Field label="End">
              <Input
                type="time"
                value={end}
                onChange={(e) => setEnd(e.target.value)}
              />
            </Field>
          </div>
          <Field label="Description">
            <Input value={desc} onChange={(e) => setDesc(e.target.value)} />
          </Field>
          <ParticipantsEditor
            meetingId={meeting.id}
            participants={meeting.participants}
            series={series}
          />
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

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
