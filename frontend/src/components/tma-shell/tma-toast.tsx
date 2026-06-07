import { cn } from "@/shared/lib/cn"
import { useMounted } from "./use-mounted"

export function TmaToast({
  data,
}: {
  data: { msg: string; emoji?: string } | null
}) {
  const { mounted, shown } = useMounted(!!data, 300)
  if (!mounted || !data) return null

  return (
    <div className="pointer-events-none absolute inset-x-0 top-[60px] z-[90] flex justify-center">
      <div
        className={cn(
          "font-display flex max-w-[84%] items-center gap-[9px] rounded-[14px] bg-[var(--tma-toast-bg)] px-4 py-[11px] text-sm font-bold text-white shadow-[0_10px_30px_rgba(0,0,0,0.3)] transition-all duration-300 ease-[cubic-bezier(.34,1.56,.64,1)]",
          shown
            ? "translate-y-0 scale-100 opacity-100"
            : "-translate-y-4 scale-[0.92] opacity-0"
        )}
      >
        <span className="text-[17px]">{data.emoji ?? "🐾"}</span>
        {data.msg}
      </div>
    </div>
  )
}
