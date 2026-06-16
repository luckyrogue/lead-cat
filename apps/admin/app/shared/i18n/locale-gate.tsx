import { useMe } from "~/shared/auth/use-me"
import { LocaleProvider, resolveLocale } from "~/shared/i18n/context"

export function LocaleGate({ children }: { children: React.ReactNode }) {
  const { data: me } = useMe()
  return (
    <LocaleProvider locale={resolveLocale(me?.user?.language)}>
      {children}
    </LocaleProvider>
  )
}
