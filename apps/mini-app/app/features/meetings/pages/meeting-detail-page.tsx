import {
  Button,
  CalendarClock,
  ChevronLeft,
  Clock,
  Link2,
  MapPin,
  Pencil,
  Trash2,
  Users,
} from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { useNavigate, useParams } from "react-router"

import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import { MeetingCancelDialog } from "~/features/meetings/components/meeting-cancel-dialog"
import { MeetingEditDialog } from "~/features/meetings/components/meeting-edit-dialog"
import { canManageMeeting } from "~/entities/meeting/lib/can-manage"
import { meetingDetailQuery } from "~/entities/meeting/queries"
import type { Meeting } from "~/entities/meeting/types"
import { useAuth } from "~/shared/auth/auth-context"
import { useLocale, useT } from "~/shared/i18n/context"
import { formatDateLong, formatTimeRange } from "~/shared/lib/format"
import { recurrenceLabel } from "~/shared/lib/recurrence-label"

export function MeetingDetailPage() {
  const { meetingId = "" } = useParams()
  const navigate = useNavigate()
  const t = useT()
  const { user } = useAuth()
  const meetingQuery = useQuery(meetingDetailQuery(meetingId))
  const [editing, setEditing] = useState(false)
  const [cancelling, setCancelling] = useState(false)

  const meeting = meetingQuery.data
  const canManage = canManageMeeting(meeting, user)

  return (
    <div className="flex flex-col gap-4">
      <button
        type="button"
        onClick={() => navigate(-1)}
        className="-ml-1 flex w-fit items-center gap-1 text-sm font-medium text-muted-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("common.back")}
      </button>

      {meetingQuery.isLoading ? (
        <LoadingState />
      ) : meetingQuery.isError ? (
        <ErrorState
          title={t("meetings.detail.errorLoad")}
          onRetry={() => meetingQuery.refetch()}
        />
      ) : !meeting ? (
        <EmptyState title={t("meetings.detail.notFound")} />
      ) : (
        <MeetingDetail
          meeting={meeting}
          canManage={canManage}
          onEdit={() => setEditing(true)}
          onDelete={() => setCancelling(true)}
        />
      )}

      {meeting ? (
        <>
          <MeetingEditDialog
            open={editing}
            onOpenChange={setEditing}
            meeting={meeting}
          />
          <MeetingCancelDialog
            open={cancelling}
            onOpenChange={setCancelling}
            meeting={meeting}
          />
        </>
      ) : null}
    </div>
  )
}

type DetailProps = {
  meeting: Meeting
  canManage: boolean
  onEdit: () => void
  onDelete: () => void
}

function MeetingDetail({ meeting, canManage, onEdit, onDelete }: DetailProps) {
  const t = useT()
  const locale = useLocale()
  const title = meeting.type || meeting.dept || t("meetings.fallbackTitle")
  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-xl font-bold text-foreground">{title}</h1>
        {meeting.rec ? (
          <span className="mt-1 inline-block rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {recurrenceLabel(t, meeting.rec)}
          </span>
        ) : null}
      </div>

      <div className="flex flex-col gap-3 rounded-[var(--radius)] border border-border/60 bg-card p-4">
        <Row
          icon={<CalendarClock className="size-4" />}
          text={formatDateLong(meeting.date, locale)}
        />
        <Row
          icon={<Clock className="size-4" />}
          text={formatTimeRange(meeting.start, meeting.end)}
        />
        {meeting.host ? (
          <Row icon={<MapPin className="size-4" />} text={meeting.host} />
        ) : null}
        {meeting.meet_link ? (
          <a
            href={meeting.meet_link}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 text-sm font-medium text-primary"
          >
            <Link2 className="size-4" />
            {t("meetings.detail.joinMeet")}
          </a>
        ) : null}
      </div>

      {meeting.desc ? (
        <p className="text-sm whitespace-pre-wrap text-muted-foreground">
          {meeting.desc}
        </p>
      ) : null}

      {meeting.participants.length > 0 ? (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5 text-sm font-semibold text-foreground">
            <Users className="size-4" />
            {t("meetings.detail.participants", {
              count: meeting.participants.length,
            })}
          </div>
          <ul className="flex flex-col gap-1">
            {meeting.participants.map((email) => (
              <li key={email} className="text-sm text-muted-foreground">
                {email}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {canManage ? (
        <div className="mt-2 flex gap-2">
          <Button variant="outline" className="flex-1" onClick={onEdit}>
            <Pencil className="size-4" />
            {t("meetings.detail.editBtn")}
          </Button>
          <Button variant="destructive" className="flex-1" onClick={onDelete}>
            <Trash2 className="size-4" />
            {t("meetings.detail.cancelBtn")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function Row({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-foreground">
      <span className="text-muted-foreground">{icon}</span>
      <span>{text}</span>
    </div>
  )
}
