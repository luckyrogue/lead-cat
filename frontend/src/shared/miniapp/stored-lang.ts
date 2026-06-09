import type { Lang } from "@/shared/miniapp/types"

export function readStoredLang(): Lang {
  const stored = localStorage.getItem("lc-lang")
  if (stored === "ru" || stored === "kk") {
    return stored
  }
  if (stored === "en") {
    return "ru"
  }
  return "ru"
}

export function writeStoredLang(lang: Lang) {
  localStorage.setItem("lc-lang", lang)
}
