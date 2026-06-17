import { data } from "react-router"

import { localeCookieHeader } from "~/shared/i18n/locale-request"
import { LocaleLandingPage } from "~/features/landing/locale-landing-page"
import { heroAssetLinks } from "~/shared/seo/hero-assets"
import { landingMeta } from "~/shared/seo/landing-meta"
import { DEFAULT_LOCALE } from "~/shared/i18n/types"

export function links() {
  return heroAssetLinks()
}

export function loader() {
  return data(
    { locale: DEFAULT_LOCALE },
    { headers: { "Set-Cookie": localeCookieHeader(DEFAULT_LOCALE) } }
  )
}

export function meta() {
  return landingMeta(DEFAULT_LOCALE)
}

export default function Index() {
  return <LocaleLandingPage locale={DEFAULT_LOCALE} />
}
