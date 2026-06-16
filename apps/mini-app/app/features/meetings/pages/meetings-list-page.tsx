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
import { useT } from "~/shared/i18n/context"
import { formatDate } from "~/shared/lib/format"

type Tab = "upcoming" | "past"

function MeetingsList({ meetings }: { meetings: Meeting[] }) {
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
    <div className="flex flex-col gap-3">
      {groups.map((group) => {
        if (group.kind === "single") {
          return <MeetingCard key={group.meeting.id} meeting={group.meeting} />
        }

        const { seriesId, meetings: occurrences } = group
        const first = occurrences[0]
        const title = first.type || first.dept || t("meetings.fallbackTitle")
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
  const t = useT()
  const [tab, setTab] = useState<Tab>("upcoming")
  const meetings = useQuery(myMeetingsQuery(tab))
  const list = meetings.data ?? []

  const tabs: { value: Tab; label: string }[] = [
    { value: "upcoming", label: t("meetings.tabUpcoming") },
    { value: "past", label: t("meetings.tabPast") },
  ]

  const emptyTitle =
    tab === "upcoming" ? t("meetings.emptyUpcoming") : t("meetings.emptyPast")

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t("meetings.title")}
        action={
          <Button asChild size="sm">
            <Link to="/meetings/create">
              <Plus className="size-4" />
              {t("meetings.newBtn")}
            </Link>
          </Button>
        }
      />

      <div className="grid grid-cols-2 gap-1 rounded-full bg-muted/70 p-1">
        {tabs.map(({ value, label }) => (
          <button
            key={value}
            type="button"
            aria-pressed={tab === value}
            onClick={() => setTab(value)}
            className={cn(
              "rounded-full py-1.5 text-sm font-medium capitalize transition-colors",
              tab === value
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground"
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {meetings.isLoading ? (
        <LoadingState />
      ) : meetings.isError ? (
        <ErrorState
          title={t("meetings.errorLoad")}
          onRetry={() => meetings.refetch()}
        />
      ) : list.length === 0 ? (
        <EmptyState title={emptyTitle} />
      ) : (
        <MeetingsList meetings={list} />
      )}
    </div>
  )
}
