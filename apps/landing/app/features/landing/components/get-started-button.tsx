import { Button } from "@leadcat/ui"
import type { ComponentProps, ReactNode } from "react"

import { getStartedLinkProps } from "~/shared/urls"

export function GetStartedButton({
  children,
  ...props
}: ComponentProps<typeof Button> & { children: ReactNode }) {
  const link = getStartedLinkProps()
  return (
    <Button asChild {...props}>
      <a href={link.href} target={link.target} rel={link.rel}>
        {children}
      </a>
    </Button>
  )
}
