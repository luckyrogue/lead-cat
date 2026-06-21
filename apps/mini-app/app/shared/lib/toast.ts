import { mapApiErrorMessage } from "@leadcat/api-client"
import { toast as sonner } from "@leadcat/ui"

type TFn = (key: string, params?: Record<string, string | number>) => string

let apiToastT: TFn | null = null

export function registerApiToastTranslator(t: TFn) {
  apiToastT = t
}

export function toastApiError(error: unknown, fallbackKey: string) {
  const t = apiToastT ?? ((key) => key)
  sonner.error(mapApiErrorMessage(t, error, fallbackKey))
}

export function toastError(error: unknown, t: TFn, fallbackKey: string) {
  sonner.error(mapApiErrorMessage(t, error, fallbackKey))
}
