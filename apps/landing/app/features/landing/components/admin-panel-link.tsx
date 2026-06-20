import type { ReactNode } from "react"

import { getAdminLoginUrl } from "~/shared/urls"

export function AdminPanelLink({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <a href={getAdminLoginUrl()} className={className}>
      {children}
    </a>
  )
}
