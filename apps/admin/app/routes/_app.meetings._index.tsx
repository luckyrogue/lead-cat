import { lazy, Suspense } from "react"

import { PageLoading } from "~/components/page-loading"
import { useT } from "~/shared/i18n/context"

const MeetingsPage = lazy(() =>
  import("~/features/meetings/pages/meetings-page").then((m) => ({
    default: m.MeetingsPage,
  }))
)

function MeetingsLoading() {
  const t = useT()
  return <PageLoading>{t("meetings.loadingMeetings")}</PageLoading>
}

export default function Meetings() {
  return (
    <Suspense fallback={<MeetingsLoading />}>
      <MeetingsPage />
    </Suspense>
  )
}
