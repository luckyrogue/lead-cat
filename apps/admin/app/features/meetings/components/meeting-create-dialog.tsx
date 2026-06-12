import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@leadcat/ui"

import { MeetingForm, type MeetingFormValues } from "~/features/meetings/components/meeting-form"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  pending: boolean
  onSubmit: (values: MeetingFormValues) => void
}

export function MeetingCreateDialog({ open, onOpenChange, pending, onSubmit }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>New meeting</DialogTitle>
          <DialogDescription>
            Schedule a Google Meet session and invite participants.
          </DialogDescription>
        </DialogHeader>
        <MeetingForm mode="create" pending={pending} onSubmit={onSubmit} />
      </DialogContent>
    </Dialog>
  )
}
