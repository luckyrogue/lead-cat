import { ParticipantsEditorPanel, toastError } from "@leadcat/ui"
import { useState } from "react"

import {
  useAddParticipant,
  useRemoveParticipant,
} from "~/entities/meeting/mutations"
import { useMeeting } from "~/entities/meeting/queries"
import { useT } from "~/shared/i18n/context"

type Props = {
  orgId: string
  meetingId: string
  series?: boolean
}

export function ParticipantsEditor({ orgId, meetingId, series }: Props) {
  const t = useT()
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
        onError: (error) =>
          toastError(error, t, "meetings.participants.addFailed"),
      }
    )
  }

  function handleRemove(target: string) {
    remove.mutate(
      { meetingId, email: target },
      {
        onError: (error) =>
          toastError(error, t, "meetings.participants.removeFailed"),
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
      labels={{
        title: t("meetings.participants.title"),
        loading: t("meetings.participants.loading"),
        empty: t("meetings.participants.empty"),
        placeholder: t("meetings.participants.placeholder"),
        removeAria: (email) => t("meetings.participants.removeAria", { email }),
        addButton: t("meetings.participants.addButton"),
        seriesHint: t("meetings.participants.seriesHint"),
      }}
    />
  )
}
