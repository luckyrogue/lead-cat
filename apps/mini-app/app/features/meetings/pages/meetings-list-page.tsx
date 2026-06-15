import { Button, ChevronDown, ChevronRight, cn, Plus } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { Link } from "react-router"

import { MeetingCard } from "~/components/meetings/meeting-card"
import { PageHeader } from "~/components/page-header"
import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import type { Meeting } from "~/entities/meeting/types"
import { myMeetingsQuery } from "~/entities/meeting/queries"
import { groupBySeries } from "~/features/meetings/lib/group-series"
import { formatDate } from "~/shared/lib/format"

type Tab = "upcoming" | "past"

function MeetingsList({ meetings }: { meetings: Meeting[] }) {
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
    <div className="flex flex-col gap-3">
      {groups.map((group) => {
        if (group.kind === "single") {
          return <MeetingCard key={group.meeting.id} meeting={group.meeting} />
        }

        const { seriesId, meetings: occurrences } = group
        const first = occurrences[0]
        const title = first.type || first.dept || "Meeting"
        const isOpen = expanded.has(seriesId)

        return (
          <div key={seriesId} className="flex flex-col gap-2">
            <button
              type="button"
              onClick={() => toggleSeries(seriesId)}
              className="flex w-full items-center gap-2 rounded-xl bg-muted/60 px-3 py-2.5 text-left transition-colors active:bg-muted"
            >
              {isOpen ? (
                <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
              ) : (
                <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
              )}
              <span className="min-w-0 flex-1 truncate font-semibold text-foreground">
                {title}
              </span>
              <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary">
                🔁 {occurrences.length}
              </span>
              <span className="shrink-0 text-sm text-muted-foreground">
                {formatDate(first.date)}
              </span>
            </button>
            {isOpen ? (
              <div className="flex flex-col gap-2 pl-3">
                {occurrences.map((m) => (
                  <MeetingCard key={m.id} meeting={m} />
                ))}
              </div>
            ) : null}
          </div>
        )
      })}
    </div>
  )
}

export function MeetingsListPage() {
  const [tab, setTab] = useState<Tab>("upcoming")
  const meetings = useQuery(myMeetingsQuery(tab))
  const list = meetings.data ?? []

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Meetings"
        action={
          <Button asChild size="sm">
            <Link to="/meetings/create">
              <Plus className="size-4" />
              New
            </Link>
          </Button>
        }
      />

      <div className="grid grid-cols-2 gap-1 rounded-full bg-muted/70 p-1">
        {(["upcoming", "past"] as const).map((t) => (
          <button
            key={t}
            type="button"
            aria-pressed={tab === t}
            onClick={() => setTab(t)}
            className={cn(
              "rounded-full py-1.5 text-sm font-medium capitalize transition-colors",
              tab === t
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground"
            )}
          >
            {t}
          </button>
        ))}
      </div>

      {meetings.isLoading ? (
        <LoadingState />
      ) : meetings.isError ? (
        <ErrorState
          title="Couldn't load meetings"
          onRetry={() => meetings.refetch()}
        />
      ) : list.length === 0 ? (
        <EmptyState title={`No ${tab} meetings`} />
      ) : (
        <MeetingsList meetings={list} />
      )}
    </div>
  )
}
