import { Clock } from "@leadcat/ui"

import type { FreeSlot } from "~/entities/meeting/types"
import { useLocale } from "~/shared/i18n/context"
import { formatDate, formatTimeRange } from "~/shared/lib/format"

export function FreeSlotList({ slots }: { slots: FreeSlot[] }) {
  const locale = useLocale()
  const byDay = new Map<string, FreeSlot[]>()
  for (const slot of slots) {
    const group = byDay.get(slot.iso) ?? []
    group.push(slot)
    byDay.set(slot.iso, group)
  }

  return (
    <div className="flex flex-col gap-4">
      {[...byDay.entries()].map(([iso, daySlots]) => (
        <div key={iso} className="flex flex-col gap-2">
          <p className="text-sm font-semibold text-foreground">
            {formatDate(iso, locale)}
          </p>
          <div className="flex flex-wrap gap-2">
            {daySlots.map((slot, i) => (
              <span
                key={`${iso}-${i}`}
                className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-card px-3 py-1.5 text-sm text-foreground"
              >
                <Clock className="size-3.5 text-muted-foreground" />
                {formatTimeRange(slot.start, slot.end)}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
