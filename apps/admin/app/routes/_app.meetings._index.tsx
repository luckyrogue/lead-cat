import { lazy, Suspense } from "react"

import { PageLoading } from "~/components/page-loading"

const MeetingsPage = lazy(() =>
  import("~/features/meetings/pages/meetings-page").then((m) => ({
    default: m.MeetingsPage,
  }))
)

export default function Meetings() {
  return (
    <Suspense fallback={<PageLoading>Loading meetings…</PageLoading>}>
      <MeetingsPage />
    </Suspense>
  )
}
