import { I18N } from "@/shared/miniapp/i18n"
import { useMiniApp } from "@/shared/miniapp/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function TgBar({
  onLang,
  native = false,
}: {
  onLang: () => void
  /** Inside Telegram: no fake ⋮/✕ — native header already provides them. */
  native?: boolean
}) {
  const { lang } = useMiniApp()
  return (
    <div className="border-miniapp-border bg-miniapp-tg-bar relative z-[6] flex h-[52px] shrink-0 items-center border-b py-0 pl-4 pr-3">
      <div className="flex min-w-0 flex-1 items-center gap-2.5">
        <div className="bg-miniapp-accent shadow-miniapp-sm flex size-[34px] shrink-0 items-center justify-center rounded-full">
          <span className="text-[19px]">🐱</span>
        </div>
        <div className="min-w-0">
          <div className="font-display text-miniapp-tg-bar-text text-base font-extrabold leading-[1.05]">
            Lead&nbsp;Cat
          </div>
          <div className="text-miniapp-muted flex items-center gap-1 text-[11px]">
            <span className="bg-miniapp-ok inline-block size-1.5 rounded-[3px]" />
            mini app
          </div>
        </div>
      </div>
      <button
        type="button"
        onClick={onLang}
        className="tg-icon-btn"
        aria-label="language"
      >
        <span className="text-base">{I18N[lang]._flag}</span>
        <CatIcon name="chevD" size={13} className="text-miniapp-muted" sw={2.4} />
      </button>
      {!native && (
        <>
          <button
            type="button"
            className="tg-icon-btn w-[34px] gap-0"
            aria-label="menu"
          >
            <span className="flex gap-[3px]">
              {[0, 1, 2].map((i) => (
                <span
                  key={i}
                  className="bg-miniapp-muted size-[3.4px] rounded-[2px]"
                />
              ))}
            </span>
          </button>
          <button
            type="button"
            className="tg-icon-btn w-[34px]"
            aria-label="close"
          >
            <CatIcon name="x" size={18} className="text-miniapp-muted" sw={2.2} />
          </button>
        </>
      )}
    </div>
  )
}
