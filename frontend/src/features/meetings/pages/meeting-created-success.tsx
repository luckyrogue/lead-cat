import { buildTitle, fmtDate } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { useTmaApp } from "@/shared/tma/context"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingCreatedSuccess({
  m,
  onDone,
  onView,
}: {
  m: Meeting
  onDone: () => void
  onView: () => void
}) {
  const { t, lang } = useTmaApp()
  return (
    <div className="px-1.5 pb-1.5 pt-2.5 text-center">
      <div className="animate-lc-pop mb-2 text-[60px]">🐱</div>
      <h2 className="tma-heading mb-1.5 text-2xl">{t("created")}</h2>
      <p className="mb-[18px] text-[14.5px] text-tma-muted">{t("createdSub")}</p>
      <div className="mb-[18px] rounded-2xl border border-tma-border bg-tma-card-alt px-[15px] py-[13px] text-left">
        <div className="tma-heading text-[15.5px] leading-snug">
          {buildTitle(m)}
        </div>
        <div className="mt-2 flex items-center gap-2.5 text-[13px] font-semibold text-tma-muted">
          <span className="inline-flex items-center gap-1">
            <CatIcon name="calendar" size={14} className="text-tma-muted" sw={2} />
            {fmtDate(m.date, lang)}
          </span>
          <span className="inline-flex items-center gap-1">
            <CatIcon name="clock" size={14} className="text-tma-muted" sw={2} />
            {m.start}–{m.end}
          </span>
        </div>
      </div>
      <div className="flex gap-2.5">
        <CatBtn variant="outline" full onClick={onView}>
          {t("preview")}
        </CatBtn>
        <CatBtn variant="primary" full onClick={onDone}>
          OK 🐾
        </CatBtn>
      </div>
    </div>
  )
}
