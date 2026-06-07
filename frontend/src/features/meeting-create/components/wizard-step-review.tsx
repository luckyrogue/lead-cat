import { buildTitle, fmtDate } from "@/shared/tma/meeting-utils"
import { useTmaApp } from "@/shared/tma/context"
import type { MeetingDraft } from "@/shared/tma/types"
import { hexToRgba } from "@/shared/tma/palette"
import { DetailRow } from "@/components/meetings/detail-row"
import { MeetingTitlePreview } from "@/components/meetings/meeting-title-preview"
import { ParticipantStack } from "@/features/meetings/components/meeting-ui"
import { CatCard, CatIcon } from "@/shared/ui/cat/primitives"
import { WizardStepTitle } from "./wizard-step-title"

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
  const p = useTmaApp()
  const t = p.t

  return (
    <div>
      <WizardStepTitle>✨ {t("preview")}</WizardStepTitle>
      <MeetingTitlePreview
        label={t("autoName")}
        title={buildTitle({ ...finalMeeting, type: draft.type })}
        marginBottom={14}
      />
      <CatCard>
        <DetailRow icon="calendar" label={t("dateT")}>
          {fmtDate(draft.date, p.lang)}
        </DetailRow>
        <DetailRow icon="clock" label={t("timeT")}>
          {draft.start} – {endTime} · UTC+5
        </DetailRow>
        <DetailRow icon="user" label={t("host")}>
          {draft.host}
        </DetailRow>
        <div
          style={{
            display: "flex",
            gap: 12,
            padding: "12px 0 0",
            alignItems: "center",
          }}
        >
          <div
            style={{
              width: 34,
              height: 34,
              borderRadius: 10,
              background: p.accentSoft,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            <CatIcon name="users" size={18} color={p.accent} sw={2} />
          </div>
          <div style={{ flex: 1 }}>
            <div
              style={{
                fontSize: 12,
                color: p.muted,
                fontWeight: 600,
                marginBottom: 4,
              }}
            >
              {t("addPeople")}
            </div>
            {draft.participants.length ? (
              <ParticipantStack
                emails={draft.participants.map((x) => x.email)}
                max={6}
              />
            ) : (
              <span style={{ color: p.faint, fontSize: 14 }}>—</span>
            )}
          </div>
        </div>
      </CatCard>
      {recurringBlocked && (
        <div
          style={{
            marginTop: 14,
            background: p.dangerSoft,
            borderRadius: 16,
            padding: "13px 15px",
            border: `1px solid ${hexToRgba(p.danger, 0.3)}`,
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 7,
              color: p.danger,
              fontWeight: 800,
              fontFamily: "var(--font-display)",
              fontSize: 14.5,
              marginBottom: 6,
            }}
          >
            ⚠️ {t("recurringSoon")}
          </div>
          <div
            style={{
              fontSize: 13,
              color: p.text,
              opacity: 0.85,
              lineHeight: 1.4,
            }}
          >
            {t("recurringSoon")}
          </div>
        </div>
      )}
      {conflictPeople.length > 0 && (
        <div
          style={{
            marginTop: 14,
            background: p.dangerSoft,
            borderRadius: 16,
            padding: "13px 15px",
            border: `1px solid ${hexToRgba(p.danger, 0.3)}`,
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 7,
              color: p.danger,
              fontWeight: 800,
              fontFamily: "var(--font-display)",
              fontSize: 14.5,
              marginBottom: 6,
            }}
          >
            ⚠️ {t("conflict")}
          </div>
          <div
            style={{
              fontSize: 13,
              color: p.text,
              opacity: 0.85,
              lineHeight: 1.4,
            }}
          >
            {t("conflictSub")} {conflictPeople.join(", ")}
          </div>
        </div>
      )}
    </div>
  )
}
