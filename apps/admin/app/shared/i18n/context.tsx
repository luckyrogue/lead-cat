import { createContext, useContext, useMemo } from "react"

import { dictionaries } from "~/shared/i18n/dictionaries"
import { translate } from "~/shared/i18n/translate"
import { DEFAULT_LOCALE, type Locale } from "~/shared/i18n/types"

type TFn = (key: string, params?: Record<string, string | number>) => string

const LocaleContext = createContext<{ locale: Locale; t: TFn }>({
  locale: DEFAULT_LOCALE,
  t: (key) => key,
})

export function LocaleProvider({
  locale,
  children,
}: {
  locale: Locale
  children: React.ReactNode
}) {
  const value = useMemo(() => {
    const active = dictionaries[locale] ?? dictionaries[DEFAULT_LOCALE]
    const t: TFn = (key, params) =>
      translate(active, dictionaries.en, key, params)
    return { locale, t }
  }, [locale])
  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  )
}

export function useT(): TFn {
  return useContext(LocaleContext).t
}

export function useLocale(): Locale {
  return useContext(LocaleContext).locale
}

export function resolveLocale(language: string | undefined | null): Locale {
  return language === "en" || language === "kk" ? language : DEFAULT_LOCALE
}
