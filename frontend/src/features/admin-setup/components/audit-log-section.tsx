import { useState } from "react"
import { useMiniApp } from "@/shared/miniapp/context"
import { useAuditLog } from "@/entities/admin/queries"
import { AUDIT_ACTIONS } from "@/entities/admin/constants"
import { SettingsGroup } from "@/features/profile/components/settings-group"

export function AuditLogSection() {
  const { t } = useMiniApp()
  const [action, setAction] = useState("")
  const { data: entries = [] } = useAuditLog(action ? { action } : {})

  return (
    <SettingsGroup title={t("adminAuditLog" as never)}>
      <div className="flex flex-col gap-3 px-3 pb-3">
        <label className="flex flex-col gap-1">
          <span className="text-miniapp-muted text-xs font-bold">{t("auditActionFilter" as never)}</span>
          <select
            value={action}
            onChange={(e) => setAction(e.target.value)}
            className="border-miniapp-border bg-miniapp-card text-miniapp-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
          >
            <option value="">{t("auditAllActions" as never)}</option>
            {AUDIT_ACTIONS.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </label>
        {entries.length === 0 && (
          <div className="text-miniapp-muted text-sm">{t("auditNoEntries" as never)}</div>
        )}
        <div className="flex flex-col gap-2">
          {entries.map((e) => (
            <div
              key={e.id}
              className="border-miniapp-border bg-miniapp-card-alt rounded-[12px] border p-3"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-miniapp-accent text-[13px] font-bold">{e.action}</span>
                <span className="text-miniapp-muted text-[11px]">
                  {new Date(e.createdAt).toLocaleString()}
                </span>
              </div>
              <div className="text-miniapp-muted mt-0.5 text-xs">{e.actorEmail}</div>
              {Object.keys(e.details).length > 0 && (
                <pre className="text-miniapp-muted mt-1 overflow-auto text-[11px] whitespace-pre-wrap">
                  {JSON.stringify(e.details, null, 2)}
                </pre>
              )}
            </div>
          ))}
        </div>
      </div>
    </SettingsGroup>
  )
}
