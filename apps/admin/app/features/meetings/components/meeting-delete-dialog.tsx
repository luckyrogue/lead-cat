import {
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import { isSeries, type Meeting, type MeetingScope } from "~/entities/meeting/types"

type Props = {
  meeting: Meeting | null
  scope: MeetingScope
  pending: boolean
  onScopeChange: (scope: MeetingScope) => void
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}

export function MeetingDeleteDialog({
  meeting,
  scope,
  pending,
  onScopeChange,
  onOpenChange,
  onConfirm,
}: Props) {
  return (
    <Dialog open={meeting !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Cancel meeting?</DialogTitle>
          <DialogDescription>
            This cancels the meeting and removes it from connected calendars. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        {meeting && isSeries(meeting) ? (
          <div className="space-y-2">
            <Label>Apply to</Label>
            <Select value={scope} onValueChange={(value) => onScopeChange(value as MeetingScope)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="this">This meeting only</SelectItem>
                <SelectItem value="whole">The whole series</SelectItem>
              </SelectContent>
            </Select>
          </div>
        ) : null}
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Keep meeting</Button>
          </DialogClose>
          <Button variant="destructive" disabled={pending} onClick={onConfirm}>
            Cancel meeting
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
