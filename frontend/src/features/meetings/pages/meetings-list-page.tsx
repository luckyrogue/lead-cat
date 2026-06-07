import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams, useSearch } from "@tanstack/react-router"
import { TmaListPageShell } from "@/components/tma-list-page-shell"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { TMA_NOW } from "@/entities/meeting/constants"
import { useTmaApp } from "@/shared/tma/context"
import { fmtDate } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { useListUrlState } from "@/shared/lib/use-list-url-state"
import {
  buildMeetingsSearchParams,
  parseMeetingsScopeFilter,
} from "@/features/meetings/list-url"
import { writeErrorKey } from "@/entities/meeting/lib/write-error"
import { useDeleteMeeting } from "@/entities/meeting/mutations"
import { useMyMeetings } from "@/entities/meeting/queries"
import { MeetingDetail } from "@/features/meetings/components/meeting-detail"
import {
  EmptyState,
  MeetingCard,
} from "@/components/meetings/meeting-ui"
import { MeetingCreatedSuccess } from "@/features/meetings/pages/meeting-created-success"
import { Segmented } from "@/shared/ui/cat/primitives"
import { Sheet, PawBurst } from "@/components/tma-shell"
import type { MeetingsSearch } from "@/features/meetings/search-schema"

export function MeetingsListPage() {
  const p = useTmaApp()
  const t = p.t
  const navigate = useNavigate()
  const search = useSearch({ strict: false }) as MeetingsSearch
  const params = useParams({ strict: false })
  const meetingId =
    "meetingId" in params ? (params.meetingId as string) : undefined

  const { filter, setFilter } = useListUrlState({
    readFilter: parseMeetingsScopeFilter,
    buildSearchParams: buildMeetingsSearchParams,
  })

  const { data: meetings = [], isLoading } = useMyMeetings("all")
  const deleteMut = useDeleteMeeting()
  const [burst, setBurst] = useState(false)

  const sorted = useMemo(
    () =>
      [...meetings].sort((a, b) =>
        (a.date + a.start).localeCompare(b.date + b.start)
      ),
    [meetings]
  )

  const list = useMemo(
    () =>
      sorted.filter((m) => {
        if (filter === "up") return m.date >= TMA_NOW
        if (filter === "past") return m.date < TMA_NOW
        return true
      }),
    [sorted, filter]
  )

  const groups = useMemo(() => {
    const out: { date: string; items: Meeting[] }[] = []
    list.forEach((m) => {
      const g = out.find((x) => x.date === m.date)
      if (g) g.items.push(m)
      else out.push({ date: m.date, items: [m] })
    })
    return out
  }, [list])

  const detail = meetingId
    ? (meetings.find((m) => m.id === meetingId) ?? null)
    : null

  const successMeeting = search.success
    ? (meetings.find((m) => m.id === search.success) ?? null)
    : null

  useEffect(() => {
    if (search.success) {
      setBurst(true)
      const timer = setTimeout(() => setBurst(false), 1100)
      return () => clearTimeout(timer)
    }
  }, [search.success])

  const preserveSearch = () => {
    const params = buildMeetingsSearchParams({
      q: search.q ?? "",
      page: search.page ?? 1,
      filter: filter ?? "up",
    })
    const out: Record<string, string> = {}
    params.forEach((v, k) => {
      out[k] = v
    })
    return out
  }

  const openMeeting = (m: Meeting) => {
    void navigate({
      to: "/meetings/$meetingId",
      params: { meetingId: m.id },
      search: preserveSearch(),
    })
  }

  const closeDetail = () => {
    void navigate({ to: "/meetings", search: preserveSearch() })
  }

  const closeSuccess = () => {
    void navigate({ to: "/meetings", search: preserveSearch() })
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteMut.mutateAsync(id)
      closeDetail()
      toastSuccess(t("deleted"))
    } catch (err) {
      toastError(err, t(writeErrorKey(err)))
    }
  }

  return (
    <>
      <TmaListPageShell
        title={t("nav_meetings")}
        isLoading={isLoading}
        empty={list.length === 0}
        filters={
          <div className="mb-[18px]">
            <Segmented
              value={filter ?? "up"}
              onChange={setFilter}
              options={[
                { value: "up", label: t("filter_up") },
                { value: "past", label: t("filter_past") },
                { value: "all", label: t("filter_all") },
              ]}
            />
          </div>
        }
        emptyState={<EmptyState emoji="🐈" title={t("emptyMeet")} />}
        detail={
          <>
            <Sheet open={!!detail} onClose={closeDetail}>
              {detail && (
                <MeetingDetail
                  m={detail}
                  onEdit={() => {
                    void navigate({
                      to: "/meetings/create/$editId",
                      params: { editId: detail.id },
                    })
                  }}
                  onDelete={() => void handleDelete(detail.id)}
                />
              )}
            </Sheet>
            <Sheet open={!!successMeeting} onClose={closeSuccess} maxH="70%">
              {successMeeting && (
                <MeetingCreatedSuccess
                  m={successMeeting}
                  onDone={closeSuccess}
                  onView={() => {
                    closeSuccess()
                    openMeeting(successMeeting)
                  }}
                />
              )}
            </Sheet>
            <PawBurst show={burst} />
          </>
        }
      >
        <div className="flex flex-col gap-5">
          {groups.map((g) => (
            <div key={g.date}>
              <div className="tma-section-title mb-[9px] px-1">
                {g.date === TMA_NOW ? t("today") : fmtDate(g.date, p.lang)}
              </div>
              <div className="flex flex-col gap-[11px]">
                {g.items.map((m) => (
                  <MeetingCard
                    key={m.id}
                    m={m}
                    onClick={() => openMeeting(m)}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      </TmaListPageShell>
    </>
  )
}
