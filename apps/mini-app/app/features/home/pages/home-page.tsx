import { Button, CatFace, Plus } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { Link } from "react-router"

import { MeetingCard } from "~/components/meetings/meeting-card"
import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import { myMeetingsQuery } from "~/entities/meeting/queries"
import { useAuth } from "~/shared/auth/auth-context"
import { useT } from "~/shared/i18n/context"
import { getGreetingName } from "~/shared/lib/display-name"
import { getTelegramUser } from "~/shared/tma/telegram-env"

export function HomePage() {
  const t = useT()
  const { user } = useAuth()
  const meetings = useQuery(myMeetingsQuery("upcoming"))
  const firstName =
    getGreetingName(user?.name, getTelegramUser()?.first_name) || "there"
  const list = meetings.data ?? []

  return (
    <div className="flex flex-col gap-5">
      <header className="flex items-center gap-3">
        <CatFace className="animate-float size-12" />
        <div className="min-w-0">
          <p className="text-sm font-semibold text-muted-foreground">
            {t("home.appSubtitle")}
          </p>
          <h1 className="truncate text-xl font-bold text-foreground">
            {t("home.greeting", { name: firstName })}
          </h1>
        </div>
      </header>

      <Button asChild size="lg" className="w-full">
        <Link to="/meetings/create">
          <Plus className="size-5" />
          {t("home.bookMeeting")}
        </Link>
      </Button>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-foreground">
            {t("home.upcomingTitle")}
          </h2>
          <Link to="/meetings" className="text-sm font-medium text-primary">
            {t("home.seeAll")}
          </Link>
        </div>

        {meetings.isLoading ? (
          <LoadingState />
        ) : meetings.isError ? (
          <ErrorState
            title={t("home.errorLoad")}
            onRetry={() => meetings.refetch()}
          />
        ) : list.length === 0 ? (
          <EmptyState title={t("home.emptyTitle")} hint={t("home.emptyHint")} />
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
