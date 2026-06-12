import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Separator,
} from "@leadcat/ui"

import { isSeries, type Meeting, type MeetingScope } from "~/entities/meeting/types"
import {
  MeetingForm,
  type MeetingFormDefaults,
  type MeetingFormValues,
} from "~/features/meetings/components/meeting-form"
import { ParticipantsEditor } from "~/features/meetings/components/participants-editor"

type Props = {
  meeting: Meeting | null
  orgId: string | null
  pending: boolean
  defaults: MeetingFormDefaults | undefined
  onOpenChange: (open: boolean) => void
  onSubmit: (values: MeetingFormValues, scope: MeetingScope) => void
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
              </>
            ) : null}
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
