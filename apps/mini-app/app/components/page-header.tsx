import { cn } from "@leadcat/ui"

type PageHeaderProps = {
  title: string
  subtitle?: string
  action?: React.ReactNode
  className?: string
}

export function PageHeader({
  title,
  subtitle,
  action,
  className,
}: PageHeaderProps) {
  return (
    <header
      className={cn("mb-5 flex items-start justify-between gap-3", className)}
    >
      <div className="min-w-0">
        {subtitle ? (
          <p className="text-sm font-semibold text-muted-foreground">
            {subtitle}
          </p>
        ) : null}
        <h1 className="text-xl font-bold text-foreground">{title}</h1>
      </div>
      {action}
    </header>
  )
}
