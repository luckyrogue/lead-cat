import { useMemo } from "react"
import { useNavigate, useParams } from "@tanstack/react-router"
import { CreateWizard } from "@/features/meeting-create/components/create-wizard"
import { detailToDraft } from "@/entities/meeting/lib/format"
import { writeErrorKey } from "@/entities/meeting/lib/write-error"
import {
  useCreateMeeting,
  useUpdateMeeting,
} from "@/entities/meeting/mutations"
import { useMyMeetings } from "@/entities/meeting/queries"
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import type { MeetingDraft } from "@/entities/meeting/types"
import type { MeetingInput, MeetingPatch } from "@/entities/meeting/write-api"
import { Overlay } from "@/components/tma-shell"

export function CreateMeetingPage() {
  const p = useTmaApp()
  const navigate = useNavigate()
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

  const createMut = useCreateMeeting()
  const updateMut = useUpdateMeeting()

  const goBack = () => {
    void navigate({ to: "/meetings", search: { scope: "upcoming" } })
  }

  const completeCreate = async (m: MeetingDraft & { end: string }) => {
    try {
      if (editId) {
        const patch: MeetingPatch = {
          dept: m.dept,
          type: m.type,
          host: m.host,
          date: m.date,
          start: m.start,
          end: m.end,
          desc: m.desc,
        }
        await updateMut.mutateAsync({ id: editId, patch })
        toastSuccess(p.t("updated"))
        void navigate({ to: "/meetings", search: { scope: "upcoming" } })
      } else {
        const input: MeetingInput = {
          dept: m.dept,
          type: m.type,
          host: m.host,
          date: m.date,
          start: m.start,
          end: m.end,
          recurrence: m.rec,
          desc: m.desc,
          participants: m.participants.map((x) => x.email),
        }
        const created = await createMut.mutateAsync(input)
        void navigate({
          to: "/meetings",
          search: { scope: "upcoming", success: created.id },
        })
      }
    } catch (err) {
      toastError(err, p.t(writeErrorKey(err)))
    }
  }

  return (
    <Overlay open onClose={goBack} onBack={goBack} title={p.t("create")}>
      <CreateWizard initial={initial} onComplete={completeCreate} />
    </Overlay>
  )
}
