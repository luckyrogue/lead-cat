import { HtmlLangSync } from "@leadcat/ui"

import { useMe } from "~/shared/auth/use-me"
import { LocaleProvider, resolveLocale } from "~/shared/i18n/context"
import { readLocalePreference } from "~/shared/i18n/locale-preference"

export function LocaleGate({ children }: { children: React.ReactNode }) {
  const { data: me } = useMe()
  const locale = resolveLocale(me?.user?.language || readLocalePreference())
  return (
    <LocaleProvider locale={locale}>
      <HtmlLangSync lang={locale} />
      {children}
    </LocaleProvider>
  )
}
