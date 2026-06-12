import { ParticipantsEditorPanel, toast } from "@leadcat/ui"
import { useState } from "react"

import {
  useAddParticipant,
  useRemoveParticipant,
} from "~/entities/meeting/mutations"

type Props = {
  meetingId: string
  participants: string[]
  series?: boolean
}

export function ParticipantsEditor({ meetingId, participants, series }: Props) {
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
        onError: () => toast.error("Couldn't add participant"),
      }
    )
  }

  function handleRemove(target: string) {
    remove.mutate(
      { id: meetingId, email: target },
      { onError: () => toast.error("Couldn't remove participant") }
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
