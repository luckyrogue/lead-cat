import { useMemo } from "react"
import { cn } from "@/shared/lib/cn"
import { useMiniApp } from "@/shared/miniapp/context"
import { validateSAJson, type SAValidation } from "../lib/sa-validate"

type Props = { value: string; onChange: (v: string) => void; disabled?: boolean }

export function SaPasteInput({ value, onChange, disabled }: Props) {
  const { t } = useMiniApp()
  const v: SAValidation = useMemo(() => validateSAJson(value), [value])
  return (
    <label className="flex flex-col gap-1">
      <span className="text-miniapp-muted text-xs font-bold">{t("googleSAJson" as never)}</span>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        spellCheck={false}
        autoCapitalize="off"
        autoComplete="off"
        rows={8}
        disabled={disabled}
        className={cn(
          "border-miniapp-border bg-miniapp-card text-miniapp-text w-full rounded-[12px] border px-3 py-2.5 font-mono text-[13px] whitespace-pre overflow-auto",
          disabled && "cursor-not-allowed opacity-60"
        )}
        placeholder={'{\n  "type": "service_account",\n  "project_id": "...",\n  "private_key": "-----BEGIN PRIVATE KEY-----\\n...",\n  ...\n}'}
      />
      {!v.ok && value.trim() !== "" && (
        <p className="text-miniapp-danger text-xs">{t(v.errorKey as never)}</p>
      )}
      <p className="text-miniapp-muted text-xs">{t("saPasteHint" as never)}</p>
    </label>
  )
}
