import { useEffect } from "react"

import { useLocale } from "~/shared/i18n/context"

export function DocumentLang() {
  const locale = useLocale()

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  return null
}
