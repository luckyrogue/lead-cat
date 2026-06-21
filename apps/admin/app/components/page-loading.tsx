import type { ReactNode } from "react"

type PageLoadingProps = {
  children: ReactNode
}

export function PageLoading({ children }: PageLoadingProps) {
  return (
    <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-6 text-sm text-muted-foreground">
      {children}
    </div>
  )
}
