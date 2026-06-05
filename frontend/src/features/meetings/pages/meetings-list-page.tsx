import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams, useSearch } from "@tanstack/react-router"
import { useQueryClient } from "@tanstack/react-query"
import { TmaListPageShell } from "@/components/tma-list-page-shell"
import { toastSuccess } from "@/shared/lib/toast"
import { TMA_NOW } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import { fmtDate } from "@/shared/tma/meeting-utils"
import type { Meeting } from "@/entities/meeting/types"
import { tmaKeys } from "@/shared/api/query-keys"
import { useListUrlState } from "@/shared/lib/use-list-url-state"
import {
  buildMeetingsSearchParams,
  parseMeetingsScopeFilter,
} from "@/features/meetings/list-url"
import { useMyMeetings } from "@/features/meetings/queries"
import { MeetingDetail } from "@/features/meetings/components/meeting-detail-sheet"
import { EmptyState, MeetingCard } from "@/features/meetings/components/meeting-ui"
import { MeetingCreatedSuccess } from "@/features/meetings/pages/meeting-created-success"
import { Segmented } from "@/shared/ui/cat/primitives"
import { Sheet, PawBurst } from "@/components/tma-shell"
import type { MeetingsSearch } from "@/routes/_tma/meetings"

export function MeetingsListPage() {
  const p = useTmaApp()
  const t = p.t
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const search = useSearch({ strict: false }) as MeetingsSearch
  const params = useParams({ strict: false })
  const meetingId =
    "meetingId" in params ? (params.meetingId as string) : undefined

  const { filter, setFilter } = useListUrlState({
    readFilter: parseMeetingsScopeFilter,
    buildSearchParams: buildMeetingsSearchParams,
  })

  const { data: meetings = [], isLoading } = useMyMeetings("all")
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
    ? meetings.find((m) => m.id === meetingId) ?? null
    : null

  const successMeeting = search.success
    ? meetings.find((m) => m.id === search.success) ?? null
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

  const deleteMeeting = (id: string) => {
    queryClient.setQueryData<Meeting[]>(tmaKeys.meetings("all"), (old = []) =>
      old.filter((m) => m.id !== id)
    )
    closeDetail()
    toastSuccess(
      p.lang === "en"
        ? "Meeting deleted"
        : p.lang === "kk"
          ? "Кездесу жойылды"
          : "Встреча удалена"
    )
  }

  return (
    <>
      <TmaListPageShell
        title={t("nav_meetings")}
        isLoading={isLoading}
        empty={list.length === 0}
        filters={
          <div style={{ marginBottom: 18 }}>
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
                  onDelete={() => deleteMeeting(detail.id)}
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
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          {groups.map((g) => (
            <div key={g.date}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 800,
                  color: p.muted,
                  margin: "0 4px 9px",
                  fontFamily: "var(--font-display)",
                  textTransform: "capitalize",
                }}
              >
                {g.date === TMA_NOW ? t("today") : fmtDate(g.date, p.lang)}
              </div>
              <div
                style={{ display: "flex", flexDirection: "column", gap: 11 }}
              >
                {g.items.map((m) => (
                  <MeetingCard key={m.id} m={m} onClick={() => openMeeting(m)} />
                ))}
              </div>
            </div>
          ))}
        </div>
      </TmaListPageShell>
    </>
  )
}
