import { Button, Input, Label, Loader2, Mail, UserPlus, X } from "@leadcat/ui"
import { useState } from "react"

import {
  useAddParticipant,
  useMeeting,
  useRemoveParticipant,
} from "~/entities/meeting/queries"
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

  const participants = meeting.data?.participants ?? []

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
    <div className="space-y-2">
      <Label>Participants</Label>
      {meeting.isPending ? (
        <p className="text-sm text-muted-foreground">Loading participants…</p>
      ) : participants.length === 0 ? (
        <p className="text-sm text-muted-foreground">No participants yet.</p>
      ) : (
        <ul className="flex flex-wrap gap-2">
          {participants.map((participant) => (
            <li
              key={participant.email}
              className="flex items-center gap-1.5 rounded-full border border-border/70 bg-muted/40 py-1 pr-1.5 pl-3 text-sm"
            >
              <Mail className="size-3.5 text-muted-foreground" />
              <span>{participant.email}</span>
              <button
                type="button"
                aria-label={`Remove ${participant.email}`}
                onClick={() => handleRemove(participant.email)}
                disabled={remove.isPending}
                className="rounded-full p-0.5 text-muted-foreground transition hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
              >
                <X className="size-3.5" />
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="flex gap-2">
        <Input
          type="email"
          placeholder="name@company.com"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault()
              handleAdd()
            }
          }}
        />
        <Button
          type="button"
          variant="outline"
          onClick={handleAdd}
          disabled={add.isPending || email.trim() === ""}
        >
          {add.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <UserPlus className="size-4" />
          )}
          Add
        </Button>
      </div>
      {series ? (
        <p className="text-xs text-muted-foreground">
          Participant changes apply to this occurrence only.
        </p>
      ) : null}
    </div>
  )
}
