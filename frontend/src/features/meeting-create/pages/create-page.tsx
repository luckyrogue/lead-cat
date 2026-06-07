import { useMemo } from "react"
import { useNavigate, useParams } from "@tanstack/react-router"
import { useQueryClient } from "@tanstack/react-query"
import { CreateWizard } from "@/features/meeting-create/components/create-wizard"
import { useTmaAuth } from "@/features/auth/auth-context"
import { useMyMeetings } from "@/features/meetings/queries"
import { draftToMeeting, detailToDraft } from "@/entities/meeting/lib/format"
import { translate } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import type { Meeting, MeetingDraft } from "@/entities/meeting/types"
import { tmaKeys } from "@/shared/api/query-keys"
import { Overlay } from "@/components/tma-shell"

export function CreateMeetingPage() {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const params = useParams({ strict: false })
  const editId = "editId" in params ? (params.editId as string) : undefined
  const slotInitial = useMemo(() => {
    const raw = sessionStorage.getItem("tma-create-initial")
    if (!raw) return undefined
    sessionStorage.removeItem("tma-create-initial")
    try {
      return JSON.parse(raw) as Partial<MeetingDraft>
    } catch {
      return undefined
    }
  }, [])

  const { data: meetings = [] } = useMyMeetings("all")
  const editing = editId ? meetings.find((m) => m.id === editId) : undefined
  const initial = useMemo(
    () =>
      editing
        ? detailToDraft(editing)
        : slotInitial
          ? { ...slotInitial }
          : undefined,
    [editing, slotInitial]
  )

  const goBack = () => {
    void navigate({ to: "/meetings", search: { scope: "upcoming" } })
  }

  const completeCreate = (m: MeetingDraft & { end: string }) => {
    const nm = draftToMeeting(
      m,
      editId ?? `new-${Date.now()}`,
      user?.email ?? ""
    )
    queryClient.setQueryData<Meeting[]>(tmaKeys.meetings("all"), (old = []) => {
      if (editId) return old.map((x) => (x.id === editId ? nm : x))
      return [nm, ...old]
    })
    void navigate({
      to: "/meetings",
      search: { scope: "upcoming", success: nm.id },
    })
  }

  return (
    <Overlay
      open
      onClose={goBack}
      onBack={goBack}
      title={translate(p.lang, "create")}
    >
      <CreateWizard initial={initial} onComplete={completeCreate} />
    </Overlay>
  )
}
