import type { ReactNode } from "react"
import { buildTitle, fmtDate } from "@/entities/meeting/lib/format"
import type { OccurrenceConflicts } from "@/entities/meeting/scheduling-api"
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
    <div className="border-tma-danger/30 bg-tma-danger-soft mt-3.5 rounded-2xl border px-[15px] py-[13px]">
      <div className="font-display text-tma-danger mb-1.5 flex items-center gap-[7px] text-[14.5px] font-extrabold">
        ⚠️ {title}
      </div>
      <div className="text-tma-text/85 text-[13px] leading-snug">
        {children}
      </div>
    </div>
  )
}

export function WizardStepReview({
  draft,
  endTime,
  finalMeeting,
  conflictOccurrences,
}: {
  draft: MeetingDraft
  endTime: string
  finalMeeting: MeetingDraft & { end: string; organizer: string }
  conflictOccurrences: OccurrenceConflicts[]
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
          <div className="bg-tma-accent-soft flex size-[34px] shrink-0 items-center justify-center rounded-[10px]">
            <CatIcon
              name="users"
              size={18}
              className="text-tma-accent"
              sw={2}
            />
          </div>
          <div className="flex-1">
            <div className="text-tma-muted mb-1 text-xs font-semibold">
              {t("addPeople")}
            </div>
            {draft.participants.length ? (
              <ParticipantStack
                emails={draft.participants.map((x) => x.email)}
                max={6}
              />
            ) : (
              <span className="text-tma-faint text-sm">—</span>
            )}
          </div>
        </div>
      </CatCard>
      {conflictOccurrences.length > 0 &&
        draft.rec === "once" &&
        (() => {
          const names = Array.from(
            new Set(
              (conflictOccurrences[0]?.conflicts ?? []).map((c) => {
                const parts = c.name.split(" ")
                return parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : "")
              })
            )
          )
          if (!names.length) return null
          return (
            <AlertBanner title={t("conflict")}>
              {t("conflictSub")} {names.join(", ")}
            </AlertBanner>
          )
        })()}
      {conflictOccurrences.length > 0 && draft.rec !== "once" && (
        <AlertBanner title={t("seriesConflicts")}>
          <div className="space-y-1">
            {conflictOccurrences.slice(0, 5).map((oc) => {
              const names = Array.from(
                new Set(
                  oc.conflicts.map((c) => {
                    const parts = c.name.split(" ")
                    return parts[0] + " " + (parts[1] ? `${parts[1][0]}.` : "")
                  })
                )
              )
              return (
                <div key={`${oc.date}-${oc.start}`}>
                  <strong>{oc.date}</strong> {oc.start}–{oc.end}:{" "}
                  {names.join(", ")}
                </div>
              )
            })}
            {conflictOccurrences.length > 5 && (
              <div className="text-tma-text/70 mt-1.5">
                {t("seriesConflictsMore").replace(
                  "{count}",
                  String(conflictOccurrences.length - 5)
                )}
              </div>
            )}
          </div>
        </AlertBanner>
      )}
    </div>
  )
}
