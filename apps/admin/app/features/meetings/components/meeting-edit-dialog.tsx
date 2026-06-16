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
import { useT } from "~/shared/i18n/context"

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
  const t = useT()
  const [until, setUntil] = useState(
    (meeting.recurrence_until ?? "").slice(0, 10)
  )
  const { mutate, isPending } = useChangeSeriesEnd(orgId)

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium">
        {t("meetings.dialog.seriesEndDate")}
      </p>
      <div className="flex items-center gap-2">
        <Label htmlFor="series-end-date" className="sr-only">
          {t("meetings.dialog.seriesEndDate")}
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
                onSuccess: () =>
                  toastSuccess(t("meetings.dialog.seriesEndSuccess")),
                onError: (error) =>
                  toastError(error, t("meetings.dialog.seriesEndFailed")),
              }
            )
          }
        >
          {t("meetings.dialog.updateEndDate")}
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
  const t = useT()
  return (
    <Dialog open={meeting !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("meetings.dialog.editTitle")}</DialogTitle>
          <DialogDescription>
            {t("meetings.dialog.editDescription")}
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
