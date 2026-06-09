import { useTmaApp } from "@/shared/tma/context"
import type { GoogleVerifyResult } from "@/entities/admin/types"

type Props =
  | { state: "idle" }
  | { state: "loading" }
  | { state: "ok"; result: GoogleVerifyResult }
  | { state: "error"; errorCode: string }

export function VerifyResultCard(props: Props) {
  const { t } = useTmaApp()
  if (props.state === "idle") return null
  if (props.state === "loading") {
    return <div className="text-tma-muted text-sm">{t("verifying" as never)}</div>
  }
  if (props.state === "ok") {
    return (
      <div className="border-tma-accent bg-tma-accent-soft text-tma-accent rounded-[12px] border p-3 text-sm">
        <div className="font-bold">{t("verifyOK" as never)}</div>
        {props.result.calendarSummary && (
          <div className="opacity-80 mt-0.5">
            {props.result.calendarSummary} ({props.result.timeZone})
          </div>
        )}
      </div>
    )
  }
  const key =
    props.errorCode === "google_sa_invalid" ? "googleSaInvalidErr" :
    props.errorCode === "google_subject_invalid" ? "googleSubjectInvalidErr" :
    props.errorCode === "google_calendar_not_accessible" ? "googleCalendarNotAccessibleErr" :
    props.errorCode === "google_api_disabled" ? "googleApiDisabledErr" :
    "verifyUnknownErr"
  return (
    <div className="border-tma-danger bg-tma-danger-soft text-tma-danger rounded-[12px] border p-3 text-sm">
      {t(key as never)}
    </div>
  )
}
