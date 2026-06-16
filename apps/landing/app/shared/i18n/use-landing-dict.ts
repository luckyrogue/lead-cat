import { dictionaries } from "~/shared/i18n/dictionaries"
import type { LandingDict } from "~/shared/i18n/dictionaries/en"
import { useLocale } from "~/shared/i18n/context"
import { DEFAULT_LOCALE } from "~/shared/i18n/types"

export function useLandingDict(): LandingDict {
  const locale = useLocale()
  return dictionaries[locale] ?? dictionaries[DEFAULT_LOCALE]
}
