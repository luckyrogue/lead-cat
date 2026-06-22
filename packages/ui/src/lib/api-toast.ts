import { mapApiErrorMessage } from "@leadcat/api-client"
import { toast as sonner } from "sonner"

type TFn = (key: string, params?: Record<string, string | number>) => string

let apiToastT: TFn | null = null

export function registerApiToastTranslator(t: TFn) {
  apiToastT = t
}

export function getErrorMessage(
  error: unknown,
  t: TFn,
  fallbackKey: string,
): string {
  return mapApiErrorMessage(t, error, fallbackKey)
}

export function toastSuccess(message: string) {
  sonner.success(message)
}

export function toastError(error: unknown, t: TFn, fallbackKey: string) {
  sonner.error(getErrorMessage(error, t, fallbackKey))
}

export function toastApiError(error: unknown, fallbackKey: string) {
  const t = apiToastT ?? ((key) => key)
  sonner.error(mapApiErrorMessage(t, error, fallbackKey))
}
