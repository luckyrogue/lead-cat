import type { Lang } from "@/shared/tma/types"

export function readStoredLang(): Lang {
  const stored = localStorage.getItem("lc-lang")
  if (stored === "ru" || stored === "kk" || stored === "en") {
    return stored
  }
  return "ru"
}

export function writeStoredLang(lang: Lang) {
  localStorage.setItem("lc-lang", lang)
}
