import { ApiError } from "@/shared/api/types"
import type { I18nKey } from "@/shared/tma/i18n"

export function writeErrorKey(error: unknown): I18nKey {
  if (!(error instanceof ApiError)) return "errGeneric"
  switch (error.code) {
    case "meetings_not_configured":
      return "errNotConfigured"
    case "meetings_recurring_unsupported":
      return "recurringSoon"
    case "forbidden":
      return "errNotYours"
  }
  if (error.status === 403 || error.status === 404) return "errNotYours"
  return "errGeneric"
}
