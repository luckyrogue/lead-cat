import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@leadcat/ui"

import {
  MeetingForm,
  type MeetingFormValues,
} from "~/features/meetings/components/meeting-form"
import { useT } from "~/shared/i18n/context"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  pending: boolean
  onSubmit: (values: MeetingFormValues) => void
}

export function MeetingCreateDialog({
  open,
  onOpenChange,
  pending,
  onSubmit,
}: Props) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("meetings.dialog.createTitle")}</DialogTitle>
          <DialogDescription>
            {t("meetings.dialog.createDescription")}
          </DialogDescription>
        </DialogHeader>
        <MeetingForm mode="create" pending={pending} onSubmit={onSubmit} />
      </DialogContent>
    </Dialog>
  )
}
