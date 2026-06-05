import { toast as sonner } from "sonner"

import { ApiError } from "@/shared/api/types"

export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.message
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
