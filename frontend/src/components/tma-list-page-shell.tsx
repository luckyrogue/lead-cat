import type { ReactNode } from "react"
import { Spinner } from "@/components/ui/spinner"

type TmaListPageShellProps = {
  title: string
  actions?: ReactNode
  isLoading?: boolean
  filters?: ReactNode
  empty?: boolean
  emptyState: ReactNode
  children: ReactNode
  detail?: ReactNode
}

export function TmaListPageShell({
  title,
  actions,
  isLoading,
  filters,
  empty,
  emptyState,
  children,
  detail,
}: TmaListPageShellProps) {
  return (
    <div className="px-4 pb-7">
      <div className="mx-1 mb-3.5 flex items-center justify-between gap-3">
        <h2 className="tma-heading m-0 text-[26px]">{title}</h2>
        {actions}
      </div>
      {filters}
      {isLoading ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : empty ? (
        <div className="overflow-hidden rounded-[20px] border border-tma-border bg-tma-card">
          {emptyState}
        </div>
      ) : (
        children
      )}
      {detail}
    </div>
  )
}
