import { buildTitle, fmtDate } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { useMiniApp } from "@/shared/miniapp/context"
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
  const { t, lang } = useMiniApp()
  return (
    <div className="px-1.5 pb-1.5 pt-2.5 text-center">
      <div className="animate-lc-pop mb-2 text-[60px]">🐱</div>
      <h2 className="miniapp-heading mb-1.5 text-2xl">{t("created")}</h2>
      <p className="text-miniapp-muted mb-[18px] text-[14.5px]">
        {t("createdSub")}
      </p>
      <div className="border-miniapp-border bg-miniapp-card-alt mb-[18px] rounded-2xl border px-[15px] py-[13px] text-left">
        <div className="miniapp-heading text-[15.5px] leading-snug">
          {buildTitle(m)}
        </div>
        <div className="text-miniapp-muted mt-2 flex items-center gap-2.5 text-[13px] font-semibold">
          <span className="inline-flex items-center gap-1">
            <CatIcon
              name="calendar"
              size={14}
              className="text-miniapp-muted"
              sw={2}
            />
            {fmtDate(m.date, lang)}
          </span>
          <span className="inline-flex items-center gap-1">
            <CatIcon name="clock" size={14} className="text-miniapp-muted" sw={2} />
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
