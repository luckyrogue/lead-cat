import type { ReactNode } from "react"
import { Spinner } from "@/components/ui/spinner"

type MiniAppListPageShellProps = {
  title: string
  actions?: ReactNode
  isLoading?: boolean
  filters?: ReactNode
  empty?: boolean
  emptyState: ReactNode
  children: ReactNode
  detail?: ReactNode
}

export function MiniAppListPageShell({
  title,
  actions,
  isLoading,
  filters,
  empty,
  emptyState,
  children,
  detail,
}: MiniAppListPageShellProps) {
  return (
    <div className="px-4 pb-7">
      <div className="mx-1 mb-3.5 flex items-center justify-between gap-3">
        <h2 className="miniapp-heading m-0 text-[26px]">{title}</h2>
        {actions}
      </div>
      {filters}
      {isLoading ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : empty ? (
        <div className="border-miniapp-border bg-miniapp-card overflow-hidden rounded-[20px] border">
          {emptyState}
        </div>
      ) : (
        children
      )}
      {detail}
    </div>
  )
}
