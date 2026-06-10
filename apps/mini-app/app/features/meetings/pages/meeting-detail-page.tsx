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
import { MeetingEditDialog } from "~/features/meetings/components/meeting-edit-dialog"
import { myMeetingsQuery } from "~/entities/meeting/queries"
import { useDeleteMeeting } from "~/entities/meeting/mutations"
import type { Meeting } from "~/entities/meeting/types"
import { useAuth } from "~/shared/auth/auth-context"
import { formatDateLong, formatTimeRange } from "~/shared/lib/format"
import { toast } from "@leadcat/ui"

export function MeetingDetailPage() {
  const { meetingId = "" } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const meetings = useQuery(myMeetingsQuery("all"))
  const [editing, setEditing] = useState(false)
  const del = useDeleteMeeting()

  const meeting = (meetings.data ?? []).find((m) => m.id === meetingId)
  const canManage = Boolean(meeting && user && meeting.organizer === user.name)

  function onDelete() {
    if (!meeting) {
      return
    }
    if (!window.confirm("Cancel this meeting?")) {
      return
    }
    del.mutate(
      { id: meeting.id, scope: "this" },
      {
        onSuccess: () => {
          toast.success("Meeting cancelled")
          void navigate("/meetings")
        },
        onError: () => toast.error("Couldn't cancel meeting"),
      }
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <button
        type="button"
        onClick={() => navigate(-1)}
        className="-ml-1 flex w-fit items-center gap-1 text-sm font-medium text-muted-foreground"
      >
        <ChevronLeft className="size-4" />
        Back
      </button>

      {meetings.isLoading ? (
        <LoadingState />
      ) : meetings.isError ? (
        <ErrorState title="Couldn't load meeting" onRetry={() => meetings.refetch()} />
      ) : !meeting ? (
        <EmptyState title="Meeting not found" />
      ) : (
        <MeetingDetail
          meeting={meeting}
          canManage={canManage}
          deleting={del.isPending}
          onEdit={() => setEditing(true)}
          onDelete={onDelete}
        />
      )}

      {meeting ? (
        <MeetingEditDialog open={editing} onOpenChange={setEditing} meeting={meeting} />
      ) : null}
    </div>
  )
}

type DetailProps = {
  meeting: Meeting
  canManage: boolean
  deleting: boolean
  onEdit: () => void
  onDelete: () => void
}

function MeetingDetail({ meeting, canManage, deleting, onEdit, onDelete }: DetailProps) {
  const title = meeting.type || meeting.dept || "Meeting"
  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-xl font-bold text-foreground">{title}</h1>
        {meeting.rec ? (
          <span className="mt-1 inline-block rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {meeting.rec}
          </span>
        ) : null}
      </div>

      <div className="flex flex-col gap-3 rounded-[var(--radius)] border border-border/60 bg-card p-4">
        <Row icon={<CalendarClock className="size-4" />} text={formatDateLong(meeting.date)} />
        <Row icon={<Clock className="size-4" />} text={formatTimeRange(meeting.start, meeting.end)} />
        {meeting.host ? <Row icon={<MapPin className="size-4" />} text={meeting.host} /> : null}
        {meeting.meet_link ? (
          <a
            href={meeting.meet_link}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 text-sm font-medium text-primary"
          >
            <Link2 className="size-4" />
            Join Google Meet
          </a>
        ) : null}
      </div>

      {meeting.desc ? (
        <p className="whitespace-pre-wrap text-sm text-muted-foreground">{meeting.desc}</p>
      ) : null}

      {meeting.participants.length > 0 ? (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5 text-sm font-semibold text-foreground">
            <Users className="size-4" />
            Participants ({meeting.participants.length})
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
            Edit
          </Button>
          <Button variant="destructive" className="flex-1" onClick={onDelete} disabled={deleting}>
            <Trash2 className="size-4" />
            Cancel
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
