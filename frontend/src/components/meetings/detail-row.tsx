import { CatIcon } from "@/shared/ui/cat/primitives"

export function DetailRow({
  icon,
  label,
  children,
}: {
  icon: Parameters<typeof CatIcon>[0]["name"]
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex gap-3 border-b border-tma-border py-3">
      <div className="flex size-[34px] shrink-0 items-center justify-center rounded-[10px] bg-tma-accent-soft">
        <CatIcon name={icon} size={18} className="text-tma-accent" sw={2} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="mb-0.5 text-xs font-semibold text-tma-muted">
          {label}
        </div>
        <div className="text-[15px] font-semibold text-tma-text">{children}</div>
      </div>
    </div>
  )
}
