import {
  Button,
  Input,
  Label,
  Loader2,
  Mail,
  UserPlus,
  X,
  toast,
} from "@leadcat/ui"
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
    <div className="flex flex-col gap-1.5">
      <Label>Participants</Label>
      {participants.length === 0 ? (
        <p className="text-sm text-muted-foreground">No participants yet.</p>
      ) : (
        <ul className="flex flex-wrap gap-2">
          {participants.map((participant) => (
            <li
              key={participant}
              className="flex items-center gap-1.5 rounded-full border border-border/60 bg-muted/40 py-1 pr-1.5 pl-3 text-sm"
            >
              <Mail className="size-3.5 text-muted-foreground" />
              <span>{participant}</span>
              <button
                type="button"
                aria-label={`Remove ${participant}`}
                onClick={() => handleRemove(participant)}
                disabled={remove.isPending}
                className="rounded-full p-0.5 text-muted-foreground disabled:opacity-50"
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
        </Button>
      </div>
      {series ? (
        <p className="text-xs text-muted-foreground">
          Applies to this occurrence only.
        </p>
      ) : null}
    </div>
  )
}
