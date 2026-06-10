import { Loader2 } from "@leadcat/ui"

import { flattenConflicts } from "~/entities/meeting/api"
import type { OccurrenceConflicts } from "~/entities/meeting/types"

type Props = {
  loading: boolean
  occurrences: OccurrenceConflicts[] | undefined
}

export function ConflictPreview({ loading, occurrences }: Props) {
  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Checking conflicts…
      </div>
    )
  }

  if (!occurrences) {
    return null
  }

  const conflicts = flattenConflicts(occurrences)
  if (conflicts.length === 0) {
    return (
      <p className="rounded-[calc(var(--radius)*0.7)] border border-border/60 bg-muted/40 px-3 py-2 text-sm text-foreground">
        No conflicts found for the selected participants.
      </p>
    )
  }

  return (
    <div className="rounded-[calc(var(--radius)*0.7)] border border-destructive/40 bg-destructive/5 px-3 py-2.5">
      <p className="text-sm font-semibold text-destructive">
        {conflicts.length} conflict{conflicts.length === 1 ? "" : "s"}
      </p>
      <ul className="mt-1.5 flex flex-col gap-1">
        {conflicts.map((c, i) => (
          <li key={`${c.email}-${i}`} className="text-xs text-muted-foreground">
            {c.name || c.email}: {c.title} ({c.start}–{c.end})
          </li>
        ))}
      </ul>
    </div>
  )
}
