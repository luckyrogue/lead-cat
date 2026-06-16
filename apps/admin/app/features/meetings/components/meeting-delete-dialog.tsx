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

import {
  isSeries,
  type Meeting,
  type MeetingScope,
} from "~/entities/meeting/types"
import { useT } from "~/shared/i18n/context"

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
  const t = useT()
  return (
    <Dialog open={meeting !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("meetings.dialog.deleteTitle")}</DialogTitle>
          <DialogDescription>
            {t("meetings.dialog.deleteDescription")}
          </DialogDescription>
        </DialogHeader>
        {meeting && isSeries(meeting) ? (
          <div className="space-y-2">
            <Label>{t("meetings.dialog.deleteApplyTo")}</Label>
            <Select
              value={scope}
              onValueChange={(value) => onScopeChange(value as MeetingScope)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="this">
                  {t("meetings.dialog.deleteScopeThis")}
                </SelectItem>
                <SelectItem value="whole">
                  {t("meetings.dialog.deleteScopeWhole")}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        ) : null}
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">
              {t("meetings.dialog.keepMeeting")}
            </Button>
          </DialogClose>
          <Button variant="destructive" disabled={pending} onClick={onConfirm}>
            {t("meetings.dialog.cancelMeeting")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
