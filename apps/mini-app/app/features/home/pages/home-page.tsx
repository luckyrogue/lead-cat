import { Button, CatFace, Plus } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { Link } from "react-router"

import { MeetingCard } from "~/components/meetings/meeting-card"
import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import { myMeetingsQuery } from "~/entities/meeting/queries"
import { useAuth } from "~/shared/auth/auth-context"

export function HomePage() {
  const { user } = useAuth()
  const meetings = useQuery(myMeetingsQuery("upcoming"))
  const firstName = (user?.name ?? "").split(" ")[0] || "there"
  const list = meetings.data ?? []

  return (
    <div className="flex flex-col gap-5">
      <header className="flex items-center gap-3">
        <CatFace className="size-12 animate-float" />
        <div className="min-w-0">
          <p className="text-sm font-semibold text-muted-foreground">Lead Cat</p>
          <h1 className="truncate text-xl font-bold text-foreground">Hi, {firstName}</h1>
        </div>
      </header>

      <Button asChild size="lg" className="w-full">
        <Link to="/meetings/create">
          <Plus className="size-5" />
          Book a meeting
        </Link>
      </Button>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-foreground">Upcoming</h2>
          <Link to="/meetings" className="text-sm font-medium text-primary">
            See all
          </Link>
        </div>

        {meetings.isLoading ? (
          <LoadingState />
        ) : meetings.isError ? (
          <ErrorState title="Couldn't load meetings" onRetry={() => meetings.refetch()} />
        ) : list.length === 0 ? (
          <EmptyState title="No upcoming meetings" hint="Book one to get started." />
        ) : (
          <div className="flex flex-col gap-3">
            {list.slice(0, 5).map((m) => (
              <MeetingCard key={m.id} meeting={m} />
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
