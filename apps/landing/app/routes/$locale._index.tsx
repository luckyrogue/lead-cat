import { data, redirect } from "react-router"

import {
  localeCookieHeader,
  parseUrlLocale,
} from "~/shared/i18n/locale-request"
import { LocaleLandingPage } from "~/shared/i18n/locale-landing-page"
import { landingMeta } from "~/shared/seo/landing-meta"
import type { UrlLocale } from "~/shared/i18n/types"

export function loader({
  params,
  request,
}: {
  params: { locale?: string }
  request: Request
}) {
  const locale = parseUrlLocale(params.locale)
  if (!locale) {
    const url = new URL(request.url)
    return redirect(`/${url.search}${url.hash}`)
  }

  return data(
    { locale },
    { headers: { "Set-Cookie": localeCookieHeader(locale) } }
  )
}

export function meta({
  params,
  location,
}: {
  params: { locale?: string }
  location: { pathname: string }
}) {
  const locale = parseUrlLocale(params.locale) ?? "en"
  return landingMeta(locale, { pathname: location.pathname })
}

export default function LocaleIndex({
  loaderData,
}: {
  loaderData: { locale: UrlLocale }
}) {
  return <LocaleLandingPage locale={loaderData.locale} />
}
