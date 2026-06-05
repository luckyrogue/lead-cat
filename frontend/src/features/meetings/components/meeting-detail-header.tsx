import { TMA_NOW } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import { MEETING_TYPES } from "@/shared/tma/mock-data"
import type { Meeting } from "@/entities/meeting/types"
import { TypeTag } from "@/shared/ui/cat/primitives"

export function MeetingDetailHeader({ m }: { m: Meeting }) {
  const p = useTmaApp()
  const t = p.t
  const tObj = MEETING_TYPES.find((x) => x.key === m.type)
  const past = m.date < TMA_NOW

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        marginBottom: 14,
        marginTop: 2,
      }}
    >
      <TypeTag typeKey={m.type} label={tObj ? tObj.label : m.type} />
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: 5,
          padding: "5px 10px",
          borderRadius: 999,
          fontSize: 12,
          fontWeight: 700,
          background: past ? p.dangerSoft : p.okSoft,
          color: past ? p.danger : p.ok,
        }}
      >
        <span
          style={{
            width: 7,
            height: 7,
            borderRadius: 4,
            background: past ? p.danger : p.ok,
          }}
        />
        {past ? t("filter_past") : t("filter_up")}
      </span>
    </div>
  )
}
