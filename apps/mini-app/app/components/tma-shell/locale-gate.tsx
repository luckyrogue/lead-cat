import { HtmlLangSync } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"

import { settingsQuery } from "~/entities/settings/queries"
import { LocaleProvider, resolveLocale } from "~/shared/i18n/context"

export function LocaleGate({ children }: { children: React.ReactNode }) {
  const settings = useQuery(settingsQuery())
  const locale = resolveLocale(settings.data?.language)
  return (
    <LocaleProvider locale={locale}>
      <HtmlLangSync lang={locale} />
      {children}
    </LocaleProvider>
  )
}
