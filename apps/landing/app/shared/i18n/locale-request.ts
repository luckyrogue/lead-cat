import {
  DEFAULT_LOCALE,
  isUrlLocale,
  type UrlLocale,
} from "~/shared/i18n/types"

const LOCALE_COOKIE = "leadcat_locale"

export function parseUrlLocale(value: string | undefined): UrlLocale | null {
  return value && isUrlLocale(value) ? value : null
}

export function localeCookieHeader(locale: string): string {
  return `${LOCALE_COOKIE}=${locale}; Path=/; Max-Age=31536000; SameSite=Lax`
}

export { DEFAULT_LOCALE }
