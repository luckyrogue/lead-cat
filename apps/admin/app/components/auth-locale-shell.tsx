import { useState, type ReactNode } from "react"

import { LanguageSelector } from "~/components/language-selector"
import { LocaleProvider } from "~/shared/i18n/context"
import {
  readLocalePreference,
  resolveGuestLocale,
  writeLocalePreference,
} from "~/shared/i18n/locale-preference"
import type { Locale } from "~/shared/i18n/types"

type AuthLocaleShellProps = {
  children: ReactNode
}

export function AuthLocaleShell({ children }: AuthLocaleShellProps) {
  const [locale, setLocale] = useState<Locale>(() => {
    if (typeof window === "undefined") {
      return resolveGuestLocale()
    }
    return readLocalePreference() ?? resolveGuestLocale()
  })

  function handleLocaleChange(next: Locale) {
    writeLocalePreference(next)
    setLocale(next)
  }

  return (
    <LocaleProvider locale={locale}>
      <div className="relative min-h-svh">
        <div className="absolute top-4 right-4 z-20 sm:top-6 sm:right-6">
          <LanguageSelector value={locale} onChange={handleLocaleChange} />
        </div>
        {children}
      </div>
    </LocaleProvider>
  )
}
