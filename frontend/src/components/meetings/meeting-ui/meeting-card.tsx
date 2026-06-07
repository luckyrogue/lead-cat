import { cn } from "@/shared/lib/cn"
import { TMA_NOW } from "@/entities/meeting/constants"
import { typeAccentVars } from "@/shared/tma/surface-vars"
import { useTmaApp } from "@/shared/tma/context"
import { MEETING_TYPES, RECURRENCE } from "@/entities/meeting/constants"
import { fmtDate } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { CatCard, CatIcon, TypeTag } from "@/shared/ui/cat/primitives"
import { ParticipantStack } from "./participant-stack"

export function MeetingCard({
  m,
  onClick,
  compact = false,
}: {
  m: Meeting
  onClick?: () => void
  compact?: boolean
}) {
  const { dark, lang } = useTmaApp()
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const past = m.date < TMA_NOW
  const rec = RECURRENCE.find((r) => r.key === m.rec)

  return (
    <CatCard
      onClick={onClick}
      interactive
      className={cn("overflow-hidden p-0", past && "opacity-72")}
    >
      <div className="flex">
        <div
          className="w-[5px] shrink-0 bg-type-solid"
          style={typeAccentVars(m.type, dark)}
        />
        <div className="min-w-0 flex-1 px-3.5 py-[13px]">
          <div className="mb-2 flex items-center gap-[7px]">
            <TypeTag
              typeKey={m.type}
              label={tObj ? tObj.label : m.type}
              size="sm"
            />
            {rec?.short && (
              <span className="inline-flex items-center gap-[3px] text-[11.5px] font-bold text-tma-muted">
                <CatIcon name="repeat" size={12} className="text-tma-muted" sw={2} />
                {rec.short}
              </span>
            )}
            <div className="flex-1" />
            <span
              className={cn(
                "size-2 shrink-0 rounded",
                past ? "bg-tma-faint" : "bg-tma-ok"
              )}
            />
          </div>
          <div className="tma-heading mb-2 break-words text-[15.5px] leading-tight">
            {m.dept} · {tObj ? tObj.label : m.type}
          </div>
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2.5 text-[13px] font-semibold text-tma-muted">
              <span className="inline-flex items-center gap-1">
                <CatIcon name="calendar" size={14} className="text-tma-muted" sw={2} />
                {fmtDate(m.date, lang)}
              </span>
              <span className="inline-flex items-center gap-1">
                <CatIcon name="clock" size={14} className="text-tma-muted" sw={2} />
                {m.start}–{m.end}
              </span>
            </div>
            {!compact && (
              <ParticipantStack
                emails={[m.organizer, ...m.participants]}
                max={3}
                size={26}
              />
            )}
          </div>
        </div>
      </div>
    </CatCard>
  )
}
