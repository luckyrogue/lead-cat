import type { ReactNode } from "react"
import { buildTitle, fmtDate } from "@/entities/meeting/lib/format"
import { useTmaApp } from "@/shared/tma/context"
import type { MeetingDraft } from "@/shared/tma/types"
import { DetailRow } from "@/components/meetings/detail-row"
import { MeetingTitlePreview } from "@/components/meetings/meeting-title-preview"
import { ParticipantStack } from "@/components/meetings/meeting-ui"
import { CatCard, CatIcon } from "@/shared/ui/cat/primitives"
import { WizardStepTitle } from "./wizard-step-title"

function AlertBanner({
  title,
  children,
}: {
  title: string
  children: ReactNode
}) {
  return (
    <div className="mt-3.5 rounded-2xl border border-tma-danger/30 bg-tma-danger-soft px-[15px] py-[13px]">
      <div className="font-display mb-1.5 flex items-center gap-[7px] text-[14.5px] font-extrabold text-tma-danger">
        ⚠️ {title}
      </div>
      <div className="text-[13px] leading-snug text-tma-text/85">
        {children}
      </div>
    </div>
  )
}

export function WizardStepReview({
  draft,
  endTime,
  finalMeeting,
  conflictPeople,
  recurringBlocked,
}: {
  draft: MeetingDraft
  endTime: string
  finalMeeting: MeetingDraft & { end: string; organizer: string }
  conflictPeople: string[]
  recurringBlocked: boolean
}) {
  const { t, lang } = useTmaApp()

  return (
    <div>
      <WizardStepTitle>✨ {t("preview")}</WizardStepTitle>
      <MeetingTitlePreview
        label={t("autoName")}
        title={buildTitle({ ...finalMeeting, type: draft.type })}
        className="mb-3.5"
      />
      <CatCard>
        <DetailRow icon="calendar" label={t("dateT")}>
          {fmtDate(draft.date, lang)}
        </DetailRow>
        <DetailRow icon="clock" label={t("timeT")}>
          {draft.start} – {endTime} · UTC+5
        </DetailRow>
        <DetailRow icon="user" label={t("host")}>
          {draft.host}
        </DetailRow>
        <div className="flex items-center gap-3 pt-3">
          <div className="flex size-[34px] shrink-0 items-center justify-center rounded-[10px] bg-tma-accent-soft">
            <CatIcon
              name="users"
              size={18}
              className="text-tma-accent"
              sw={2}
            />
          </div>
          <div className="flex-1">
            <div className="mb-1 text-xs font-semibold text-tma-muted">
              {t("addPeople")}
            </div>
            {draft.participants.length ? (
              <ParticipantStack
                emails={draft.participants.map((x) => x.email)}
                max={6}
              />
            ) : (
              <span className="text-sm text-tma-faint">—</span>
            )}
          </div>
        </div>
      </CatCard>
      {recurringBlocked && (
        <AlertBanner title={t("recurringSoon")}>{t("recurringSoon")}</AlertBanner>
      )}
      {conflictPeople.length > 0 && (
        <AlertBanner title={t("conflict")}>
          {t("conflictSub")} {conflictPeople.join(", ")}
        </AlertBanner>
      )}
    </div>
  )
}
