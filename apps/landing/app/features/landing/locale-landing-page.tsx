import { LocaleProvider } from "~/shared/i18n/context"
import { DocumentLang } from "~/shared/i18n/document-lang"
import type { Locale } from "~/shared/i18n/types"
import { LandingPage } from "~/features/landing/pages/landing-page"

export function LocaleLandingPage({ locale }: { locale: Locale }) {
  return (
    <LocaleProvider locale={locale}>
      <DocumentLang />
      <LandingPage />
    </LocaleProvider>
  )
}
