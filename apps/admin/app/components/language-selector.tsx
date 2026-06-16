import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import { useT } from "~/shared/i18n/context"
import { LOCALES, type Locale } from "~/shared/i18n/types"

type LanguageSelectorProps = {
  value: Locale
  onChange: (locale: Locale) => void
  className?: string
}

export function LanguageSelector({
  value,
  onChange,
  className,
}: LanguageSelectorProps) {
  const t = useT()

  return (
    <Select value={value} onValueChange={(next) => onChange(next as Locale)}>
      <SelectTrigger
        aria-label={t("locale.label")}
        className={className ?? "h-9 w-[9.5rem] border-border/70 bg-background/80 backdrop-blur-sm"}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="end">
        {LOCALES.map((locale) => (
          <SelectItem key={locale} value={locale}>
            {t(`locale.${locale}`)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
