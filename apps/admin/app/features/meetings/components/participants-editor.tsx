import { ParticipantsEditorPanel } from "@leadcat/ui"
import { useState } from "react"

import {
  useAddParticipant,
  useRemoveParticipant,
} from "~/entities/meeting/mutations"
import { useMeeting } from "~/entities/meeting/queries"
import { toastError } from "~/shared/lib/toast"

type Props = {
  orgId: string
  meetingId: string
  series?: boolean
}

export function ParticipantsEditor({ orgId, meetingId, series }: Props) {
  const meeting = useMeeting(orgId, meetingId)
  const add = useAddParticipant(orgId)
  const remove = useRemoveParticipant(orgId)
  const [email, setEmail] = useState("")

  const participants = (meeting.data?.participants ?? []).map((p) => p.email)

  function handleAdd() {
    const value = email.trim()
    if (!value) {
      return
    }
    add.mutate(
      { meetingId, email: value },
      {
        onSuccess: () => setEmail(""),
        onError: (error) => toastError(error, "Could not add the participant."),
      }
    )
  }

  function handleRemove(target: string) {
    remove.mutate(
      { meetingId, email: target },
      {
        onError: (error) =>
          toastError(error, "Could not remove the participant."),
      }
    )
  }

  return (
    <ParticipantsEditorPanel
      className="space-y-2"
      participants={participants}
      email={email}
      onEmailChange={setEmail}
      onAdd={handleAdd}
      onRemove={handleRemove}
      addPending={add.isPending}
      removePending={remove.isPending}
      loading={meeting.isPending}
      series={series}
      addButtonLabel="Add"
      seriesHint="Participant changes apply to this occurrence only."
    />
  )
}
