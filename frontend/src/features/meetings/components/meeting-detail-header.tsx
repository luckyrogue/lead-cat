import { cn } from "@/shared/lib/cn"
import { MINIAPP_NOW } from "@/entities/meeting/constants"
import { useMiniApp } from "@/shared/miniapp/context"
import { MEETING_TYPES } from "@/entities/meeting/constants"
import type { Meeting } from "@/entities/meeting/types"
import { TypeTag } from "@/shared/ui/cat/primitives"

export function MeetingDetailHeader({ m }: { m: Meeting }) {
  const t = useMiniApp().t
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const past = m.date < MINIAPP_NOW

  return (
    <div className="mb-3.5 mt-0.5 flex items-center gap-2">
      <TypeTag typeKey={m.type} label={tObj ? tObj.label : m.type} />
      <span
        className={cn(
          "inline-flex items-center gap-[5px] rounded-full px-2.5 py-[5px]",
          "text-xs font-bold",
          past
            ? "bg-miniapp-danger-soft text-miniapp-danger"
            : "bg-miniapp-ok-soft text-miniapp-ok"
        )}
      >
        <span
          className={cn(
            "size-[7px] rounded",
            past ? "bg-miniapp-danger" : "bg-miniapp-ok"
          )}
        />
        {past ? t("filter_past") : t("filter_up")}
      </span>
    </div>
  )
}
