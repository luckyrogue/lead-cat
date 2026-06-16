import { useState } from "react"
import {
  Badge,
  Button,
  ChevronDown,
  ChevronRight,
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
import { formatDateTime, formatTimeRange } from "~/features/meetings/lib/format"
import { groupBySeries } from "~/features/meetings/lib/group-series"
import { useT } from "~/shared/i18n/context"

type MeetingsTableProps = {
  meetings: Meeting[]
  pendingId: string | null
  onEdit: (meeting: Meeting) => void
  onDelete: (meeting: Meeting) => void
  timeZone?: string
}

type OccurrenceRowProps = {
  meeting: Meeting
  pendingId: string | null
  onEdit: (meeting: Meeting) => void
  onDelete: (meeting: Meeting) => void
  indented?: boolean
  timeZone?: string
}

function OccurrenceRow({
  meeting,
  pendingId,
  onEdit,
  onDelete,
  indented = false,
  timeZone,
}: OccurrenceRowProps) {
  const t = useT()
  const isPending = pendingId === meeting.id
  const isCancelled = meeting.status === "cancelled"
  return (
    <TableRow key={meeting.id}>
      <TableCell>
        {indented ? (
          <span className="pl-6">
            <span className="font-medium text-foreground">
              {meeting.name || meeting.type || t("meetings.table.untitled")}
            </span>
            {meeting.dept ? (
              <span className="block text-xs text-muted-foreground">
                {meeting.dept}
              </span>
            ) : null}
          </span>
        ) : (
          <>
            <span className="font-medium text-foreground">
              {meeting.name || meeting.type || t("meetings.table.untitled")}
            </span>
            {meeting.dept ? (
              <span className="block text-xs text-muted-foreground">
                {meeting.dept}
              </span>
            ) : null}
          </>
        )}
      </TableCell>
      <TableCell>
        <span className="text-foreground">
          {formatDateTime(meeting.starts_at, timeZone)}
        </span>
        <span className="block text-xs text-muted-foreground">
          {formatTimeRange(meeting, timeZone)}
        </span>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {t(`meetings.recurrence.${meeting.recurrence}`)}
      </TableCell>
      <TableCell>
        <Badge tone={isCancelled ? "muted" : "sunny"}>{meeting.status}</Badge>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-1">
          <Button
            variant="ghost"
            size="sm"
            disabled={isPending || isCancelled}
            onClick={() => onEdit(meeting)}
            aria-label={t("meetings.table.editAriaLabel")}
          >
            <Pencil className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={isPending || isCancelled}
            onClick={() => onDelete(meeting)}
            aria-label={t("meetings.table.cancelAriaLabel")}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}

export function MeetingsTable({
  meetings,
  pendingId,
  onEdit,
  onDelete,
  timeZone,
}: MeetingsTableProps) {
  const t = useT()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const groups = groupBySeries(meetings)

  function toggleSeries(sid: string) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(sid)) {
        next.delete(sid)
      } else {
        next.add(sid)
      }
      return next
    })
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("meetings.table.colMeeting")}</TableHead>
          <TableHead>{t("meetings.table.colWhen")}</TableHead>
          <TableHead>{t("meetings.table.colRepeats")}</TableHead>
          <TableHead>{t("meetings.table.colStatus")}</TableHead>
          <TableHead className="text-right">
            {t("meetings.table.colActions")}
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {groups.map((group) => {
          if (group.kind === "single") {
            return (
              <OccurrenceRow
                key={group.meeting.id}
                meeting={group.meeting}
                pendingId={pendingId}
                onEdit={onEdit}
                onDelete={onDelete}
                timeZone={timeZone}
              />
            )
          }

          const { seriesId, meetings: occurrences } = group
          const first = occurrences[0]
          const isExpanded = expanded.has(seriesId)
          const earliest = occurrences.reduce((a, b) =>
            a.starts_at < b.starts_at ? a : b
          )
          const count = occurrences.length

          return [
            <TableRow
              key={`series-${seriesId}`}
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => toggleSeries(seriesId)}
            >
              <TableCell>
                <span className="flex items-center gap-2">
                  {isExpanded ? (
                    <ChevronDown className="size-4 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="size-4 text-muted-foreground" />
                  )}
                  <span>
                    <span className="font-medium text-foreground">
                      {first.name || first.type || t("meetings.table.untitled")}
                    </span>
                    {first.dept ? (
                      <span className="block text-xs text-muted-foreground">
                        {first.dept}
                      </span>
                    ) : null}
                  </span>
                </span>
              </TableCell>
              <TableCell>
                <span className="text-foreground">
                  {formatDateTime(earliest.starts_at, timeZone)}
                </span>
                <span className="block text-xs text-muted-foreground">
                  {formatTimeRange(earliest, timeZone)}
                </span>
              </TableCell>
              <TableCell className="text-muted-foreground">
                {t(`meetings.recurrence.${first.recurrence}`)}
              </TableCell>
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {t(
                    count === 1
                      ? "meetings.table.occurrences"
                      : "meetings.table.occurrencesPlural",
                    { count }
                  )}
                </span>
              </TableCell>
              <TableCell />
            </TableRow>,
            ...(isExpanded
              ? occurrences.map((occ) => (
                  <OccurrenceRow
                    key={occ.id}
                    meeting={occ}
                    pendingId={pendingId}
                    onEdit={onEdit}
                    onDelete={onDelete}
                    indented
                    timeZone={timeZone}
                  />
                ))
              : []),
          ]
        })}
      </TableBody>
    </Table>
  )
}
