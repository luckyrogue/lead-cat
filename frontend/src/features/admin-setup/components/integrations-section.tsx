import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useIntegrations } from "@/entities/admin/queries"
import { useUpdateIntegrations, useVerifyIntegrations } from "@/entities/admin/mutations"
import type { GoogleVerifyResult } from "@/entities/admin/types"
import { validateSAJson } from "../lib/sa-validate"
import { SaPasteInput } from "./sa-paste-input"
import { VerifyResultCard } from "./verify-result-card"
import { SettingsGroup } from "@/features/profile/components/settings-group"

type VerifyState =
  | { state: "idle" }
  | { state: "loading" }
  | { state: "ok"; result: GoogleVerifyResult }
  | { state: "error"; errorCode: string }

export function IntegrationsSection() {
  const { t } = useTmaApp()
  const { data: int } = useIntegrations()
  const updateMut = useUpdateIntegrations()
  const verifyMut = useVerifyIntegrations()
  const [sa, setSa] = useState("")
  const [subject, setSubject] = useState(int?.googleSubject ?? "")
  const [calendarID, setCalendarID] = useState(int?.googleCalendarID || "primary")
  const [verify, setVerify] = useState<VerifyState>({ state: "idle" })

  const saValid = validateSAJson(sa)
  const canSave = (sa === "" || saValid.ok) && subject.trim() !== ""

  async function onSave() {
    try {
      await updateMut.mutateAsync({
        googleSAJson: sa || undefined,
        googleSubject: subject.trim(),
        googleCalendarID: calendarID.trim() || "primary",
      })
      toastSuccess(t("integrationsSaved" as never))
      setVerify({ state: "loading" })
      try {
        const res = await verifyMut.mutateAsync()
        setVerify({ state: "ok", result: res })
      } catch (err: unknown) {
        const code = typeof (err as { code?: string })?.code === "string"
          ? (err as { code: string }).code
          : "internal_error"
        setVerify({ state: "error", errorCode: code })
      }
      setSa("")
    } catch (err) {
      toastError(err, t("integrationsSaveFailed" as never))
    }
  }

  return (
    <SettingsGroup title={t("adminIntegrations" as never)}>
      <div className="flex flex-col gap-3 px-3 pb-3">
        <label className="flex flex-col gap-1">
          <span className="text-tma-muted text-xs font-bold">{t("googleSubject" as never)}</span>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
            placeholder="admin@yourdomain.com"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-tma-muted text-xs font-bold">{t("googleCalendarID" as never)}</span>
          <input
            value={calendarID}
            onChange={(e) => setCalendarID(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
          />
        </label>
        <SaPasteInput value={sa} onChange={setSa} />
        <button
          type="button"
          disabled={!canSave || updateMut.isPending}
          onClick={() => void onSave()}
          className="border-tma-accent bg-tma-accent self-start rounded-[12px] px-4 py-2 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          {updateMut.isPending ? t("saving" as never) : t("verifyButton" as never)}
        </button>
        <VerifyResultCard {...verify} />
      </div>
    </SettingsGroup>
  )
}
