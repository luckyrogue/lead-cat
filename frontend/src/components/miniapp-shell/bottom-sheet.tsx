import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"
import { translate } from "@/shared/miniapp/i18n"
import { readStoredLang } from "@/shared/miniapp/stored-lang"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { useMounted } from "./use-mounted"
import { useSwipeDismiss } from "./use-swipe-dismiss"

const MAX_H_CLASS = {
  "88%": "max-h-[88%]",
  "94%": "max-h-[94%]",
  "70%": "max-h-[70%]",
  "fit-content": "",
} as const

export type BottomSheetProps = {
  open: boolean
  onClose: () => void
  children: ReactNode
  title?: string
  label?: string
  footer?: ReactNode
  onBack?: () => void
  maxH?: keyof typeof MAX_H_CLASS
  zIndex?: 70 | 75 | 80
  dismissible?: boolean
  showHandle?: boolean
  bodyClassName?: string
}

export function BottomSheet({
  open,
  onClose,
  children,
  title,
  label,
  footer,
  onBack,
  maxH = "88%",
  zIndex = 70,
  dismissible = true,
  showHandle = true,
  bodyClassName,
}: BottomSheetProps) {
  const { mounted, shown } = useMounted(open, 320)
  const swipe = useSwipeDismiss(onClose, dismissible, open)

  if (!mounted) return null

  const closeLabel = translate(readStoredLang(), "close")
  const fitContent = maxH === "fit-content"

  return (
    <div
      className={cn(
        "absolute inset-0 flex flex-col justify-end",
        zIndex === 80 ? "z-[80]" : zIndex === 75 ? "z-[75]" : "z-[70]"
      )}
    >
      <div
        onClick={dismissible ? onClose : undefined}
        className={cn(
          "absolute inset-0 transition-[background,backdrop-filter] duration-300",
          shown
            ? "bg-[rgba(20,12,4,0.42)] backdrop-blur-[2px]"
            : "bg-transparent backdrop-blur-none"
        )}
      />
      <div
        onClick={(e) => e.stopPropagation()}
        {...swipe.pointerHandlers}
        className={cn(
          "bg-miniapp-bg relative flex flex-col rounded-t-[26px] shadow-[0_-10px_40px_rgba(0,0,0,0.25)]",
          MAX_H_CLASS[maxH],
          !fitContent && "min-h-0",
          !footer && "pb-[max(20px,env(safe-area-inset-bottom,0px))]"
        )}
        style={{
          transform: swipe.getTransform(shown),
          transition: swipe.panelMotionStyle.transition,
        }}
      >
        {showHandle && (
          <div className="flex shrink-0 touch-none justify-center pt-2.5">
            <div className="bg-miniapp-border-strong h-[5px] w-10 rounded-[3px]" />
          </div>
        )}

        {title && (
          <div
            className={cn(
              "border-miniapp-border flex shrink-0 touch-none items-center gap-2 border-b",
              showHandle ? "px-2.5 pb-2.5 pt-1" : "px-2.5 py-3"
            )}
          >
            <button
              type="button"
              onClick={onBack ?? onClose}
              aria-label={onBack ? undefined : closeLabel}
              className="tg-icon-btn text-miniapp-text ml-0 w-[38px]"
            >
              <CatIcon name={onBack ? "chevL" : "x"} size={20} sw={2.2} />
            </button>
            <div className="miniapp-heading mr-[38px] flex-1 text-center text-[17px]">
              {title}
            </div>
          </div>
        )}

        {label && (
          <div className="miniapp-label shrink-0 touch-none px-5 pb-1.5 pt-2.5">
            {label}
          </div>
        )}

        <div
          ref={swipe.scrollRef}
          className={cn(
            "lc-scroll",
            fitContent ? "overflow-visible" : "min-h-0 flex-1 overflow-auto",
            bodyClassName ?? "px-4 pb-[26px] pt-2"
          )}
        >
          {children}
        </div>

        {footer && (
          <div className="border-miniapp-border bg-miniapp-tg-bar shrink-0 border-t px-4 py-3 pb-[max(12px,env(safe-area-inset-bottom,0px))]">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}
