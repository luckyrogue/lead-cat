import { Card, CardContent, Clock, Users } from "@leadcat/ui"
import { Link } from "react-router"

import type { Meeting } from "~/entities/meeting/types"
import { formatDate, formatTimeRange } from "~/shared/lib/format"

export function MeetingCard({ meeting }: { meeting: Meeting }) {
  const title = meeting.type || meeting.dept || "Meeting"
  const participantCount = meeting.participants.length

  return (
    <Link to={`/meetings/${meeting.id}`} className="block">
      <Card className="transition-colors active:bg-muted/50">
        <CardContent className="flex flex-col gap-2">
          <div className="flex items-start justify-between gap-3">
            <p className="min-w-0 truncate font-semibold text-foreground">{title}</p>
            {meeting.rec ? (
              <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary">
                {meeting.rec}
              </span>
            ) : null}
          </div>
          <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Clock className="size-3.5" />
            <span>{formatDate(meeting.date)}</span>
            <span>·</span>
            <span>{formatTimeRange(meeting.start, meeting.end)}</span>
          </div>
          {participantCount > 0 ? (
            <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Users className="size-3.5" />
              <span>
                {participantCount} {participantCount === 1 ? "participant" : "participants"}
              </span>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </Link>
  )
}
