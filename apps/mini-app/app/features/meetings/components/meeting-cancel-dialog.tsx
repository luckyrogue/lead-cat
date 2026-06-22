import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  toast,
  toastError,
} from "@leadcat/ui"
import { useEffect, useState } from "react"
import { useNavigate } from "react-router"

import { useDeleteMeeting } from "~/entities/meeting/mutations"
import {
  isSeriesMeeting,
  type Meeting,
  type MeetingMutationScope,
} from "~/entities/meeting/types"
import { ScopeToggle } from "~/features/meetings/components/scope-toggle"
import { useT } from "~/shared/i18n/context"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting: Meeting
}

export function MeetingCancelDialog({ open, onOpenChange, meeting }: Props) {
  const t = useT()
  const navigate = useNavigate()
  const del = useDeleteMeeting()
  const [scope, setScope] = useState<MeetingMutationScope>("this")
  const series = isSeriesMeeting(meeting)

  useEffect(() => {
    if (open) {
      setScope("this")
    }
  }, [open])

  function onConfirm() {
    del.mutate(
      { id: meeting.id, scope },
      {
        onSuccess: () => {
          toast.success(t("meetings.cancel.toastSuccess"))
          onOpenChange(false)
          void navigate("/meetings")
        },
        onError: (error) => toastError(error, t, "meetings.cancel.toastError"),
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("meetings.cancel.title")}</DialogTitle>
          <DialogDescription>
            {t("meetings.cancel.description")}
          </DialogDescription>
        </DialogHeader>
        {series ? (
          <div className="flex flex-col gap-3">
            <ScopeToggle value={scope} onChange={setScope} />
          </div>
        ) : null}
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("meetings.cancel.keepBtn")}
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={del.isPending}
          >
            {t("meetings.cancel.confirmBtn")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
