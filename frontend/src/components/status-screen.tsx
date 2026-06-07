import type { ReactNode } from "react"
import { cn } from "@/shared/lib/cn"

type StatusScreenProps = {
  emoji: string
  title: string
  children?: ReactNode
  action?: ReactNode
  className?: string
}

export function StatusScreen({
  emoji,
  title,
  children,
  action,
  className,
}: StatusScreenProps) {
  return (
    <main
      className={cn(
        "flex min-h-screen items-center justify-center bg-cat-bg p-6 font-[family-name:var(--font-body)]",
        className
      )}
    >
      <div className="mx-auto flex w-full max-w-[420px] flex-col gap-4 text-center">
        <div className="text-5xl">{emoji}</div>
        <h1 className="font-display m-0 text-2xl font-extrabold text-cat-secondary">
          {title}
        </h1>
        {children}
        {action}
      </div>
    </main>
  )
}
