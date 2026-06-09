import { CatCard } from "@/shared/ui/cat/primitives"

export function SettingsGroup({
  title,
  children,
}: {
  title?: string
  children: React.ReactNode
}) {
  return (
    <div className="mb-5">
      {title && <div className="tma-section-title mx-1 mb-[9px]">{title}</div>}
      <CatCard className="overflow-hidden p-0">{children}</CatCard>
    </div>
  )
}
