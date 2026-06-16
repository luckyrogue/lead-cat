import { useQuery } from "@tanstack/react-query"

import { settingsQuery } from "~/entities/settings/queries"
import { LocaleProvider, resolveLocale } from "~/shared/i18n/context"

export function LocaleGate({ children }: { children: React.ReactNode }) {
  const settings = useQuery(settingsQuery())
  return (
    <LocaleProvider locale={resolveLocale(settings.data?.language)}>
      {children}
    </LocaleProvider>
  )
}
