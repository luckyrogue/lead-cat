import { ChevronLeft } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { useNavigate } from "react-router"

import { EmployeePicker } from "~/components/employee-picker"
import { MeetingCard } from "~/components/meetings/meeting-card"
import { PageHeader } from "~/components/page-header"
import { EmptyState, ErrorState, LoadingState } from "~/components/states"
import { scheduleQuery } from "~/entities/meeting/queries"
import type { Employee } from "~/entities/employee/types"

export function ColleagueSchedulePage() {
  const navigate = useNavigate()
  const [selected, setSelected] = useState<Employee[]>([])
  const email = selected[0]?.email ?? ""
  const schedule = useQuery(scheduleQuery(email, "upcoming"))
  const list = schedule.data ?? []

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
      <PageHeader title="Colleague schedule" subtitle="Upcoming meetings" />

      <EmployeePicker
        selected={selected}
        onChange={(next) => setSelected(next.slice(-1))}
      />

      {!email ? (
        <EmptyState title="Pick a colleague" hint="Search to see their upcoming meetings." />
      ) : schedule.isLoading ? (
        <LoadingState />
      ) : schedule.isError ? (
        <ErrorState title="Couldn't load schedule" onRetry={() => schedule.refetch()} />
      ) : list.length === 0 ? (
        <EmptyState title="No upcoming meetings" />
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
