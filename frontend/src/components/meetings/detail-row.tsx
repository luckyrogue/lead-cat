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
    <div className="border-miniapp-border flex gap-3 border-b py-3">
      <div className="bg-miniapp-accent-soft flex size-[34px] shrink-0 items-center justify-center rounded-[10px]">
        <CatIcon name={icon} size={18} className="text-miniapp-accent" sw={2} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-miniapp-muted mb-0.5 text-xs font-semibold">
          {label}
        </div>
        <div className="text-miniapp-text text-[15px] font-semibold">
          {children}
        </div>
      </div>
    </div>
  )
}
