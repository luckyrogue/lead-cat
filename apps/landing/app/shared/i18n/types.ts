export type Locale = "en" | "ru" | "kk"
export type Dict = Record<string, unknown>
export const LOCALES: Locale[] = ["ru", "en", "kk"]
export const URL_LOCALES = ["en", "kk"] as const satisfies readonly Locale[]
export type UrlLocale = (typeof URL_LOCALES)[number]
export const DEFAULT_LOCALE: Locale = "ru"

export function isLocale(value: string): value is Locale {
  return LOCALES.includes(value as Locale)
}

export function isUrlLocale(value: string): value is UrlLocale {
  return (URL_LOCALES as readonly string[]).includes(value)
}
