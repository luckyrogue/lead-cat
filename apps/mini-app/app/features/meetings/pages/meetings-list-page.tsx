import { Button, cn, Plus } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { Link } from "react-router"

import { MeetingCard } from "~/components/meetings/meeting-card"
import { PageHeader } from "~/components/page-header"
import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import { myMeetingsQuery } from "~/entities/meeting/queries"

type Tab = "upcoming" | "past"

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
            onClick={() => setTab(t)}
            className={cn(
              "rounded-full py-1.5 text-sm font-medium capitalize transition-colors",
              tab === t ? "bg-background text-foreground shadow-sm" : "text-muted-foreground"
            )}
          >
            {t}
          </button>
        ))}
      </div>

      {meetings.isLoading ? (
        <LoadingState />
      ) : meetings.isError ? (
        <ErrorState title="Couldn't load meetings" onRetry={() => meetings.refetch()} />
      ) : list.length === 0 ? (
        <EmptyState title={`No ${tab} meetings`} />
      ) : (
        <div className="flex flex-col gap-3">
          {list.map((m) => (
            <MeetingCard key={m.id} meeting={m} />
          ))}
        </div>
      )}
    </div>
  )
}
