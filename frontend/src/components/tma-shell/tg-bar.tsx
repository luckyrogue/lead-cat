import { I18N } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function TgBar({
  onLang,
  native = false,
}: {
  onLang: () => void
  /** Inside Telegram: no fake ⋮/✕ — native header already provides them. */
  native?: boolean
}) {
  const { lang } = useTmaApp()
  return (
    <div className="relative z-[6] flex h-[52px] shrink-0 items-center border-b border-tma-border bg-tma-tg-bar py-0 pl-4 pr-3">
      <div className="flex min-w-0 flex-1 items-center gap-2.5">
        <div className="flex size-[34px] shrink-0 items-center justify-center rounded-full bg-tma-accent shadow-tma-sm">
          <span className="text-[19px]">🐱</span>
        </div>
        <div className="min-w-0">
          <div className="font-display text-base font-extrabold leading-[1.05] text-tma-tg-bar-text">
            Lead&nbsp;Cat
          </div>
          <div className="flex items-center gap-1 text-[11px] text-tma-muted">
            <span className="inline-block size-1.5 rounded-[3px] bg-tma-ok" />
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
        <CatIcon name="chevD" size={13} className="text-tma-muted" sw={2.4} />
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
                  className="size-[3.4px] rounded-[2px] bg-tma-muted"
                />
              ))}
            </span>
          </button>
          <button
            type="button"
            className="tg-icon-btn w-[34px]"
            aria-label="close"
          >
            <CatIcon name="x" size={18} className="text-tma-muted" sw={2.2} />
          </button>
        </>
      )}
    </div>
  )
}
