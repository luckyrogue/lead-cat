import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  type ReactNode,
} from "react"
import { translate, type I18nKey } from "./i18n"
import type { CatPalette, Lang } from "./types"

export type MiniAppContextValue = CatPalette & {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: I18nKey) => string
  openLangPicker: () => void
}

const MiniAppContext = createContext<MiniAppContextValue | null>(null)

export function MiniAppProvider({
  value,
  children,
}: {
  value: Omit<MiniAppContextValue, "t">
  children: ReactNode
}) {
  const t = useCallback(
    (key: I18nKey) => translate(value.lang, key),
    [value.lang]
  )
  const ctx = useMemo(() => ({ ...value, t }), [value, t])
  return <MiniAppContext.Provider value={ctx}>{children}</MiniAppContext.Provider>
}

export function useMiniApp(): MiniAppContextValue {
  const ctx = useContext(MiniAppContext)
  if (!ctx) throw new Error("useMiniApp must be used within MiniAppProvider")
  return ctx
}
