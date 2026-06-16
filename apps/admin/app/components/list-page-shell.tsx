import type { ReactNode } from "react"

import { Card, CardContent } from "@leadcat/ui"

import { PageHeader } from "~/components/page-header"
import { PageLoading } from "~/components/page-loading"
import { useT } from "~/shared/i18n/context"

type ListPageShellProps = {
  title: string
  eyebrow?: string
  description?: string
  actions?: ReactNode
  isLoading: boolean
  loadingMessage: string
  error?: unknown
  isEmpty: boolean
  emptyState: ReactNode
  children: ReactNode
}

export function ListPageShell({
  title,
  eyebrow,
  description,
  actions,
  isLoading,
  loadingMessage,
  error,
  isEmpty,
  emptyState,
  children,
}: ListPageShellProps) {
  const t = useT()

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={title}
        eyebrow={eyebrow}
        description={description}
        actions={actions}
      />
      <Card>
        <CardContent>
          {isLoading ? (
            <PageLoading>{loadingMessage}</PageLoading>
          ) : error ? (
            <p
              className="rounded-[calc(var(--radius)*1.15)] border border-destructive/20 bg-destructive/5 px-4 py-5 text-sm text-destructive"
              role="alert"
            >
              {error instanceof Error
                ? error.message
                : t("common.failedToLoad")}
            </p>
          ) : isEmpty ? (
            emptyState
          ) : (
            children
          )}
        </CardContent>
      </Card>
    </div>
  )
}
