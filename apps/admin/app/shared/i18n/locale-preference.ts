import { DEFAULT_LOCALE, LOCALES, type Locale } from "~/shared/i18n/types"

const STORAGE_KEY = "leadcat_admin_locale"
const COOKIE_KEY = "leadcat_locale"

export function readLocalePreference(): Locale | null {
  if (typeof window === "undefined") {
    return null
  }
  const stored = window.localStorage.getItem(STORAGE_KEY)
  if (stored && LOCALES.includes(stored as Locale)) {
    return stored as Locale
  }
  return null
}

export function writeLocalePreference(locale: Locale): void {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(STORAGE_KEY, locale)
  document.cookie = `${COOKIE_KEY}=${locale}; Path=/; Max-Age=31536000; SameSite=Lax`
}

export function resolveGuestLocale(): Locale {
  const stored = readLocalePreference()
  if (stored) {
    return stored
  }
  if (typeof navigator === "undefined") {
    return DEFAULT_LOCALE
  }
  const lang = navigator.language.toLowerCase()
  if (lang.startsWith("kk")) {
    return "kk"
  }
  if (lang.startsWith("en")) {
    return "en"
  }
  return DEFAULT_LOCALE
}
