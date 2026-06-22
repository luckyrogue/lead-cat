import * as React from "react"

import { cn } from "../../lib/cn"
import { Label } from "./label"

type FieldProps = {
  label: React.ReactNode
  hint?: React.ReactNode
  error?: string
  className?: string
  children: React.ReactNode
}

function Field({ label, hint, error, className, children }: FieldProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <Label>
        {label}
        {hint ? (
          <span className="ml-2 text-xs font-normal text-muted-foreground">
            {hint}
          </span>
        ) : null}
      </Label>
      {children}
      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}

export { Field }
