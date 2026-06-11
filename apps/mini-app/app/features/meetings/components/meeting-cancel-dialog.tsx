import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  toast,
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

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting: Meeting
}

export function MeetingCancelDialog({ open, onOpenChange, meeting }: Props) {
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
          toast.success("Meeting cancelled")
          onOpenChange(false)
          void navigate("/meetings")
        },
        onError: () => toast.error("Couldn't cancel meeting"),
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Cancel meeting?</DialogTitle>
          <DialogDescription>
            This removes the meeting from connected calendars and can't be
            undone.
          </DialogDescription>
        </DialogHeader>
        {series ? (
          <div className="flex flex-col gap-3">
            <ScopeToggle value={scope} onChange={setScope} />
          </div>
        ) : null}
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Keep
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={del.isPending}
          >
            Cancel meeting
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
