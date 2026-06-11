import {
  Badge,
  Button,
  Pencil,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Trash2,
} from "@leadcat/ui"

import type { Meeting } from "~/entities/meeting/types"
import {
  formatDateTime,
  formatTimeRange,
  recurrenceLabel,
} from "~/features/meetings/lib/format"

type MeetingsTableProps = {
  meetings: Meeting[]
  pendingId: string | null
  onEdit: (meeting: Meeting) => void
  onDelete: (meeting: Meeting) => void
}

export function MeetingsTable({
  meetings,
  pendingId,
  onEdit,
  onDelete,
}: MeetingsTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Meeting</TableHead>
          <TableHead>When</TableHead>
          <TableHead>Repeats</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {meetings.map((meeting) => {
          const isPending = pendingId === meeting.id
          const isCancelled = meeting.status === "cancelled"
          return (
            <TableRow key={meeting.id}>
              <TableCell>
                <span className="font-medium text-foreground">
                  {meeting.name || meeting.type || "Untitled"}
                </span>
                {meeting.dept ? (
                  <span className="block text-xs text-muted-foreground">
                    {meeting.dept}
                  </span>
                ) : null}
              </TableCell>
              <TableCell>
                <span className="text-foreground">
                  {formatDateTime(meeting.starts_at)}
                </span>
                <span className="block text-xs text-muted-foreground">
                  {formatTimeRange(meeting)}
                </span>
              </TableCell>
              <TableCell className="text-muted-foreground">
                {recurrenceLabel(meeting.recurrence)}
              </TableCell>
              <TableCell>
                <Badge tone={isCancelled ? "muted" : "sunny"}>
                  {meeting.status}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isPending || isCancelled}
                    onClick={() => onEdit(meeting)}
                    aria-label="Edit meeting"
                  >
                    <Pencil className="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isPending || isCancelled}
                    onClick={() => onDelete(meeting)}
                    aria-label="Cancel meeting"
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
