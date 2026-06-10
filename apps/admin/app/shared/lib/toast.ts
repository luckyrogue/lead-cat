import { toast as sonner } from "@leadcat/ui"

import { ApiError } from "~/shared/api/client"

const ERROR_COPY: Record<string, string> = {
  last_owner: "An organization needs at least one owner.",
  invalid_email: "That email address looks off.",
  invalid_role: "That role isn't allowed.",
  validation_failed: "Please check the form and try again.",
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    if (error.code && ERROR_COPY[error.code]) {
      return ERROR_COPY[error.code]
    }
    return error.message || fallback
  }
  if (error instanceof Error) {
    return error.message
  }
  return fallback
}

export function toastSuccess(message: string) {
  sonner.success(message)
}

export function toastError(error: unknown, fallback: string) {
  sonner.error(getErrorMessage(error, fallback))
}
