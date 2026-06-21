import { Button } from "../ui/button"
import { Input } from "../ui/input"
import { Label } from "../ui/label"
import { Loader2, Mail, UserPlus, X } from "lucide-react"

export type ParticipantsEditorLabels = {
  title: string
  loading: string
  empty: string
  placeholder: string
  removeAria: (email: string) => string
  addButton?: string
  seriesHint?: string
}

export type ParticipantsEditorPanelProps = {
  participants: string[]
  email: string
  onEmailChange: (value: string) => void
  onAdd: () => void
  onRemove: (email: string) => void
  labels: ParticipantsEditorLabels
  addPending?: boolean
  removePending?: boolean
  loading?: boolean
  series?: boolean
  className?: string
}

export function ParticipantsEditorPanel({
  participants,
  email,
  onEmailChange,
  onAdd,
  onRemove,
  labels,
  addPending = false,
  removePending = false,
  loading = false,
  series = false,
  className,
}: ParticipantsEditorPanelProps) {
  return (
    <div className={className ?? "flex flex-col gap-1.5"}>
      <Label>{labels.title}</Label>
      {loading ? (
        <p className="text-sm text-muted-foreground">{labels.loading}</p>
      ) : participants.length === 0 ? (
        <p className="text-sm text-muted-foreground">{labels.empty}</p>
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
                aria-label={labels.removeAria(participant)}
                onClick={() => onRemove(participant)}
                disabled={removePending}
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
          placeholder={labels.placeholder}
          value={email}
          onChange={(event) => onEmailChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault()
              onAdd()
            }
          }}
        />
        <Button
          type="button"
          variant="outline"
          onClick={onAdd}
          disabled={addPending || email.trim() === ""}
        >
          {addPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <>
              <UserPlus className="size-4" />
              {labels.addButton}
            </>
          )}
        </Button>
      </div>
      {series && labels.seriesHint ? (
        <p className="text-xs text-muted-foreground">{labels.seriesHint}</p>
      ) : null}
    </div>
  )
}
