import { useState } from "react"

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Separator,
} from "@leadcat/ui"

import { useChangeSeriesEnd } from "~/entities/meeting/mutations"
import {
  isSeries,
  type Meeting,
  type MeetingScope,
} from "~/entities/meeting/types"
import {
  MeetingForm,
  type MeetingFormDefaults,
  type MeetingFormValues,
} from "~/features/meetings/components/meeting-form"
import { ParticipantsEditor } from "~/features/meetings/components/participants-editor"
import { toastError, toastSuccess } from "~/shared/lib/toast"

type Props = {
  meeting: Meeting | null
  orgId: string | null
  pending: boolean
  defaults: MeetingFormDefaults | undefined
  onOpenChange: (open: boolean) => void
  onSubmit: (values: MeetingFormValues, scope: MeetingScope) => void
}

function SeriesEndEditor({
  meeting,
  orgId,
}: {
  meeting: Meeting
  orgId: string
}) {
  const [until, setUntil] = useState(
    (meeting.recurrence_until ?? "").slice(0, 10)
  )
  const { mutate, isPending } = useChangeSeriesEnd(orgId)

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium">Series end date</p>
      <div className="flex items-center gap-2">
        <Label htmlFor="series-end-date" className="sr-only">
          Series end date
        </Label>
        <Input
          id="series-end-date"
          type="date"
          value={until}
          onChange={(e) => setUntil(e.target.value)}
          className="w-44"
        />
        <Button
          type="button"
          size="sm"
          disabled={isPending || until === ""}
          onClick={() =>
            mutate(
              { meetingId: meeting.id, until },
              {
                onSuccess: () => toastSuccess("Series end date updated."),
                onError: (error) =>
                  toastError(error, "Could not update the series end date."),
              }
            )
          }
        >
          Update end date
        </Button>
      </div>
    </div>
  )
}

export function MeetingEditDialog({
  meeting,
  orgId,
  pending,
  defaults,
  onOpenChange,
  onSubmit,
}: Props) {
  return (
    <Dialog open={meeting !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Edit meeting</DialogTitle>
          <DialogDescription>
            Update the details of this meeting occurrence.
          </DialogDescription>
        </DialogHeader>
        {meeting && defaults ? (
          <>
            <MeetingForm
              mode="edit"
              pending={pending}
              series={isSeries(meeting)}
              defaults={defaults}
              onSubmit={onSubmit}
            />
            {orgId ? (
              <>
                <Separator />
                <ParticipantsEditor
                  orgId={orgId}
                  meetingId={meeting.id}
                  series={isSeries(meeting)}
                />
                {isSeries(meeting) ? (
                  <>
                    <Separator />
                    <SeriesEndEditor meeting={meeting} orgId={orgId} />
                  </>
                ) : null}
              </>
            ) : null}
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
