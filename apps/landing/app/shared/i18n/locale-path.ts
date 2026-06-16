import { DEFAULT_LOCALE, type Locale } from "~/shared/i18n/types"

export function localePath(locale: Locale, hash = ""): string {
  if (locale === DEFAULT_LOCALE) {
    return `/${hash}`
  }
  return `/${locale}${hash}`
}
