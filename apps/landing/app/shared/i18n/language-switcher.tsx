import {
  Check,
  ChevronDown,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@leadcat/ui"
import { Link, useLocation } from "react-router"

import { useLocale, useT } from "~/shared/i18n/context"
import { localePath } from "~/shared/i18n/locale-path"
import { LOCALES } from "~/shared/i18n/types"

const localeLabels = {
  ru: "RU",
  en: "EN",
  kk: "KK",
} as const

export function LanguageSwitcher() {
  const t = useT()
  const locale = useLocale()
  const { hash } = useLocation()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="text-kitty-700 inline-flex items-center gap-1.5 rounded-2xl border border-border/60 bg-cream-50/80 px-3 py-1.5 text-xs font-bold transition-colors outline-none hover:border-coral-200 hover:text-coral-500 focus-visible:ring-2 focus-visible:ring-coral-300/60"
        aria-label={t("lang.switchLabel")}
      >
        <span>{localeLabels[locale]}</span>
        <ChevronDown className="size-3.5 opacity-70" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-[10rem]">
        {LOCALES.map((code) => (
          <DropdownMenuItem key={code} asChild>
            <Link
              to={localePath(code, hash)}
              className="text-kitty-700 font-medium"
            >
              <span className="flex-1">{t(`lang.${code}`)}</span>
              <Check
                className={`size-4 text-coral-500 ${
                  locale === code ? "opacity-100" : "opacity-0"
                }`}
              />
            </Link>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
