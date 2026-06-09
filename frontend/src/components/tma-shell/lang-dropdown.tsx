import { I18N } from "@/shared/tma/i18n"
import type { Lang } from "@/shared/tma/types"
import { useTmaApp } from "@/shared/tma/context"
import { cn } from "@/shared/lib/cn"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { BottomSheet } from "./bottom-sheet"

export function LangDropdown({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { lang, setLang, t } = useTmaApp()
  const langs: Lang[] = ["ru", "kk", "en"]

  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      label={t("language")}
      maxH="fit-content"
      zIndex={80}
      bodyClassName="px-3 pb-2 pt-1"
    >
      {langs.map((lng) => {
        const active = lng === lang
        return (
          <button
            key={lng}
            type="button"
            onClick={() => {
              setLang(lng)
              onClose()
            }}
            className={cn(
              "font-display flex w-full cursor-pointer items-center gap-[11px] rounded-[14px] border-none px-3 py-3.5 text-left text-base font-bold",
              active
                ? "bg-tma-accent-soft text-tma-accent"
                : "text-tma-text bg-transparent"
            )}
          >
            <span className="text-[22px]">{I18N[lng]._flag}</span>
            <span className="flex-1">{I18N[lng]._label}</span>
            {active && (
              <CatIcon
                name="check"
                size={18}
                className="text-tma-accent"
                sw={2.6}
              />
            )}
          </button>
        )
      })}
    </BottomSheet>
  )
}
