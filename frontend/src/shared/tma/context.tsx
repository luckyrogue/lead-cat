import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  type ReactNode,
} from "react"
import { translate, type I18nKey } from "./i18n"
import type { CatPalette, Lang } from "./types"

export type TmaAppContextValue = CatPalette & {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: I18nKey) => string
  showToast: (msg: string, emoji?: string) => void
}

const TmaAppContext = createContext<TmaAppContextValue | null>(null)

export function TmaAppProvider({
  value,
  children,
}: {
  value: Omit<TmaAppContextValue, "t">
  children: ReactNode
}) {
  const t = useCallback(
    (key: I18nKey) => translate(value.lang, key),
    [value.lang]
  )
  const ctx = useMemo(() => ({ ...value, t }), [value, t])
  return <TmaAppContext.Provider value={ctx}>{children}</TmaAppContext.Provider>
}

export function useTmaApp(): TmaAppContextValue {
  const ctx = useContext(TmaAppContext)
  if (!ctx) throw new Error("useTmaApp must be used within TmaAppProvider")
  return ctx
}
