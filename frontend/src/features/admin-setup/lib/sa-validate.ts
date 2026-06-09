import type { I18nKey } from "@/shared/miniapp/i18n"

export type SAValidation =
  | { ok: true; clientEmail: string; projectID: string }
  | { ok: false; errorKey: I18nKey }

export function validateSAJson(text: string): SAValidation {
  if (!text.trim()) {
    return { ok: false, errorKey: "saInvalidJson" as I18nKey }
  }
  let obj: unknown
  try {
    obj = JSON.parse(text)
  } catch {
    return { ok: false, errorKey: "saInvalidJson" as I18nKey }
  }
  if (typeof obj !== "object" || obj === null) {
    return { ok: false, errorKey: "saInvalidJson" as I18nKey }
  }
  const o = obj as Record<string, unknown>
  if (o.type !== "service_account") {
    return { ok: false, errorKey: "saNotServiceAccount" as I18nKey }
  }
  const required = ["project_id", "private_key", "client_email", "token_uri"]
  const missing = required.filter((k) => typeof o[k] !== "string" || !o[k])
  if (missing.length) {
    return { ok: false, errorKey: "saMissingFields" as I18nKey }
  }
  return {
    ok: true,
    clientEmail: o.client_email as string,
    projectID: o.project_id as string,
  }
}
