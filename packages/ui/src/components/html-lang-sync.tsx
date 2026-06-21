import { useEffect } from "react"

// Keeps the document's <html lang> attribute in sync with the active locale.
// Side-effect only — renders nothing. Pass the resolved locale as `lang`.
export function HtmlLangSync({ lang }: { lang: string }) {
  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])
  return null
}
