import { useMemo } from "react"
import { cn } from "@/shared/lib/cn"
import { useTmaApp } from "@/shared/tma/context"
import { validateSAJson, type SAValidation } from "../lib/sa-validate"

type Props = { value: string; onChange: (v: string) => void; disabled?: boolean }

export function SaPasteInput({ value, onChange, disabled }: Props) {
  const { t } = useTmaApp()
  const v: SAValidation = useMemo(() => validateSAJson(value), [value])
  return (
    <label className="flex flex-col gap-1">
      <span className="text-tma-muted text-xs font-bold">{t("googleSAJson" as never)}</span>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        spellCheck={false}
        autoCapitalize="off"
        autoComplete="off"
        rows={8}
        disabled={disabled}
        className={cn(
          "border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 font-mono text-[13px] whitespace-pre overflow-auto",
          disabled && "cursor-not-allowed opacity-60"
        )}
        placeholder={'{\n  "type": "service_account",\n  "project_id": "...",\n  "private_key": "-----BEGIN PRIVATE KEY-----\\n...",\n  ...\n}'}
      />
      {!v.ok && value.trim() !== "" && (
        <p className="text-tma-danger text-xs">{t(v.errorKey as never)}</p>
      )}
      <p className="text-tma-muted text-xs">{t("saPasteHint" as never)}</p>
    </label>
  )
}
