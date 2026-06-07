import { cn } from "@/shared/lib/cn"
import { TMA_NOW } from "@/entities/meeting/constants"
import { useTmaApp } from "@/shared/tma/context"
import { MEETING_TYPES } from "@/entities/meeting/constants"
import type { Meeting } from "@/entities/meeting/types"
import { TypeTag } from "@/shared/ui/cat/primitives"

export function MeetingDetailHeader({ m }: { m: Meeting }) {
  const t = useTmaApp().t
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const past = m.date < TMA_NOW

  return (
    <div className="mb-3.5 mt-0.5 flex items-center gap-2">
      <TypeTag typeKey={m.type} label={tObj ? tObj.label : m.type} />
      <span
        className={cn(
          "inline-flex items-center gap-[5px] rounded-full px-2.5 py-[5px]",
          "text-xs font-bold",
          past
            ? "bg-tma-danger-soft text-tma-danger"
            : "bg-tma-ok-soft text-tma-ok"
        )}
      >
        <span
          className={cn(
            "size-[7px] rounded",
            past ? "bg-tma-danger" : "bg-tma-ok"
          )}
        />
        {past ? t("filter_past") : t("filter_up")}
      </span>
    </div>
  )
}
