import { CatFace, Loader2 } from "@leadcat/ui"

import { useT } from "~/shared/i18n/context"

export function LoadingState({ label }: { label?: string }) {
  const t = useT()
  return (
    <div className="flex flex-col items-center gap-3 py-12 text-muted-foreground">
      <Loader2 className="size-6 animate-spin" />
      <span className="text-sm">{label ?? t("common.loading")}</span>
    </div>
  )
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="flex flex-col items-center gap-3 py-12 text-center">
      <CatFace className="size-14 opacity-70" />
      <p className="font-semibold text-foreground">{title}</p>
      {hint ? <p className="text-sm text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

export function ErrorState({
  title,
  onRetry,
}: {
  title: string
  onRetry?: () => void
}) {
  const t = useT()
  return (
    <div className="flex flex-col items-center gap-3 py-12 text-center">
      <p className="font-semibold text-foreground">{title}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          {t("states.tryAgain")}
        </button>
      ) : null}
    </div>
  )
}
