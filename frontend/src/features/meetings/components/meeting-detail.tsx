import { TMA_NOW } from "@/entities/meeting/constants"
import { useTmaAuth } from "@/shared/auth/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import { RECURRENCE } from "@/entities/meeting/constants"
import { buildTitle, fmtDate } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { DetailRow } from "@/components/meetings/detail-row"
import { MeetingTitlePreview } from "@/components/meetings/meeting-title-preview"
import { MeetingDetailActions } from "./meeting-detail-actions"
import { MeetingDetailHeader } from "./meeting-detail-header"
import { MeetingDetailMeetLink } from "./meeting-detail-meet-link"
import { MeetingDetailParticipants } from "./meeting-detail-participants"

export function MeetingDetail({
  m,
  onEdit,
  onDelete,
}: {
  m: Meeting
  onEdit: (scope: "this" | "whole") => void
  onDelete: (scope: "this" | "whole") => void
}) {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const t = p.t
  const rec = RECURRENCE.find((r) => r.key === m.rec)
  const past = m.date < TMA_NOW
  const canManage = m.organizer === user?.email || user?.role === "admin"
  const isSeries = Boolean(m.seriesId)

  return (
    <div>
      <MeetingDetailHeader m={m} />
      <MeetingTitlePreview label={t("autoName")} title={buildTitle(m)} />
      <MeetingDetailMeetLink />
      <DetailRow icon="calendar" label={t("dateT")}>
        {fmtDate(m.date, p.lang)} · {m.date.split("-").reverse().join(".")}
      </DetailRow>
      <DetailRow icon="clock" label={t("timeT")}>
        {m.start} – {m.end}{" "}
        <span className="text-tma-faint text-[13px] font-semibold">
          · UTC+5
        </span>
      </DetailRow>
      <DetailRow icon="repeat" label={t("recur")}>
        {rec ? rec.label : "—"}
        {m.rec === "weekly" && m.recDays
          ? ` (${m.recDays.map((d) => ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"][d - 1]).join(", ")})`
          : ""}
      </DetailRow>
      <DetailRow icon="user" label={t("host")}>
        {m.host}
      </DetailRow>
      {m.desc && (
        <DetailRow icon="doc" label={t("descr")}>
          {m.desc}
        </DetailRow>
      )}
      <MeetingDetailParticipants m={m} />
      {canManage && !past && (
        <MeetingDetailActions
          isSeries={isSeries}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      )}
    </div>
  )
}
