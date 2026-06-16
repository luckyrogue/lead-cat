import { ParticipantsEditorPanel, toast } from "@leadcat/ui"
import { useState } from "react"

import {
  useAddParticipant,
  useRemoveParticipant,
} from "~/entities/meeting/mutations"
import { useT } from "~/shared/i18n/context"

type Props = {
  meetingId: string
  participants: string[]
  series?: boolean
}

export function ParticipantsEditor({ meetingId, participants, series }: Props) {
  const t = useT()
  const add = useAddParticipant()
  const remove = useRemoveParticipant()
  const [email, setEmail] = useState("")

  function handleAdd() {
    const value = email.trim()
    if (!value) {
      return
    }
    add.mutate(
      { id: meetingId, email: value },
      {
        onSuccess: () => setEmail(""),
        onError: () => toast.error(t("meetings.edit.participantAddError")),
      }
    )
  }

  function handleRemove(target: string) {
    remove.mutate(
      { id: meetingId, email: target },
      { onError: () => toast.error(t("meetings.edit.participantRemoveError")) }
    )
  }

  return (
    <ParticipantsEditorPanel
      participants={participants}
      email={email}
      onEmailChange={setEmail}
      onAdd={handleAdd}
      onRemove={handleRemove}
      addPending={add.isPending}
      removePending={remove.isPending}
      series={series}
    />
  )
}
