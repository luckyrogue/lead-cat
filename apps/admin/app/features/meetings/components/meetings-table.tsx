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
import { MeetingIdentity } from "~/features/meetings/components/meeting-identity"
import {
  formatMeetingDate,
  formatTimeRange,
  meetingTitle,
} from "~/features/meetings/lib/format"
import { groupBySeries } from "~/features/meetings/lib/group-series"
import { useLocale, useT } from "~/shared/i18n/context"

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
  locale: string
}

function meetingStatusLabel(
  status: string,
  t: ReturnType<typeof useT>
): string {
  const key = `meetings.status.${status}`
  const label = t(key)
  return label === key ? status : label
}

function OccurrenceRow({
  meeting,
  pendingId,
  onEdit,
  onDelete,
  indented = false,
  timeZone,
  locale,
}: OccurrenceRowProps) {
  const t = useT()
  const isPending = pendingId === meeting.id
  const isCancelled = meeting.status === "cancelled"
  const formatOpts = { timeZone, locale }

  return (
    <TableRow>
      <TableCell className="min-w-[12rem]">
        <MeetingIdentity
          meeting={meeting}
          title={meetingTitle(meeting, t("meetings.table.untitled"))}
          indented={indented}
        />
      </TableCell>
      <TableCell className="min-w-[9rem] whitespace-nowrap">
        <p className="font-medium text-foreground">
          {formatMeetingDate(meeting.starts_at, formatOpts)}
        </p>
        <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
          {formatTimeRange(meeting, formatOpts)}
        </p>
      </TableCell>
      <TableCell className="min-w-[7rem] text-foreground">
        {t(`meetings.recurrence.${meeting.recurrence}`)}
      </TableCell>
      <TableCell className="min-w-[8rem]">
        <Badge tone={isCancelled ? "muted" : "coral"}>
          {meetingStatusLabel(meeting.status, t)}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-2">
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
  const locale = useLocale()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const groups = groupBySeries(meetings)
  const formatOpts = { timeZone, locale }

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
          <TableHead className="min-w-[12rem]">
            {t("meetings.table.colMeeting")}
          </TableHead>
          <TableHead className="min-w-[9rem]">
            {t("meetings.table.colWhen")}
          </TableHead>
          <TableHead className="min-w-[7rem]">
            {t("meetings.table.colRepeats")}
          </TableHead>
          <TableHead className="min-w-[8rem]">
            {t("meetings.table.colStatus")}
          </TableHead>
          <TableHead className="min-w-[6rem] text-right">
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
                locale={locale}
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
              className="cursor-pointer"
              onClick={() => toggleSeries(seriesId)}
            >
              <TableCell className="min-w-[12rem]">
                <div className="flex items-start gap-2">
                  {isExpanded ? (
                    <ChevronDown className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                  )}
                  <MeetingIdentity
                    meeting={first}
                    title={meetingTitle(first, t("meetings.table.untitled"))}
                  />
                </div>
              </TableCell>
              <TableCell className="min-w-[9rem] whitespace-nowrap">
                <p className="font-medium text-foreground">
                  {formatMeetingDate(earliest.starts_at, formatOpts)}
                </p>
                <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
                  {formatTimeRange(earliest, formatOpts)}
                </p>
              </TableCell>
              <TableCell className="min-w-[7rem] text-foreground">
                {t(`meetings.recurrence.${first.recurrence}`)}
              </TableCell>
              <TableCell className="min-w-[8rem]">
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
                    locale={locale}
                  />
                ))
              : []),
          ]
        })}
      </TableBody>
    </Table>
  )
}
