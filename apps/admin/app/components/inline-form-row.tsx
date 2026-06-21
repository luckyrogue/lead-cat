import type { ComponentProps } from "react"

import { Label } from "@leadcat/ui"

import { cn } from "~/shared/lib/cn"

type InlineFormRowProps = ComponentProps<"div">

export function InlineFormRow({
  children,
  className,
  middleCol = "9rem",
  ...props
}: InlineFormRowProps & { middleCol?: string }) {
  return (
    <div
      className={cn(
        "grid grid-cols-1 gap-y-4 sm:grid-cols-[minmax(0,1fr)_var(--inline-form-middle-col)_auto] sm:gap-x-3 sm:gap-y-2",
        className
      )}
      style={
        {
          "--inline-form-middle-col": middleCol,
        } as React.CSSProperties
      }
      {...props}
    >
      {children}
    </div>
  )
}

const fieldColClass = {
  1: {
    label: "sm:col-start-1 sm:row-start-1",
    control: "sm:col-start-1 sm:row-start-2",
    hint: "sm:col-start-1 sm:row-start-3",
  },
  2: {
    label: "sm:col-start-2 sm:row-start-1",
    control: "sm:col-start-2 sm:row-start-2",
    hint: "sm:col-start-2 sm:row-start-3",
  },
} as const

type InlineFormFieldProps = {
  label: React.ReactNode
  htmlFor: string
  col: 1 | 2
  error?: string
  children: React.ReactNode
}

export function InlineFormField({
  label,
  htmlFor,
  col,
  error,
  children,
}: InlineFormFieldProps) {
  const slots = fieldColClass[col]
  return (
    <div className="grid grid-cols-1 gap-y-2 sm:contents">
      <Label htmlFor={htmlFor} className={slots.label}>
        {label}
      </Label>
      <div className={slots.control}>{children}</div>
      {col === 1 ? (
        <p
          className={cn("min-h-5 text-sm text-destructive", slots.hint)}
          role={error ? "alert" : undefined}
          aria-live={error ? "polite" : undefined}
        >
          {error ?? ""}
        </p>
      ) : null}
    </div>
  )
}

type InlineFormActionProps = {
  children: React.ReactNode
  className?: string
}

export function InlineFormAction({
  children,
  className,
}: InlineFormActionProps) {
  return (
    <div className={cn("sm:col-start-3 sm:row-start-2 sm:self-end", className)}>
      {children}
    </div>
  )
}
