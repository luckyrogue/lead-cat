export const SITE_NAME = "Lead Cat"
export const THEME_COLOR = "#F2603F"
export const DEFAULT_SITE_URL = "https://lead-cat.space"

export function resolveSiteUrl(
  value: string | undefined = import.meta.env?.VITE_SITE_URL,
): string {
  const trimmed = value?.trim().replace(/\/$/, "")
  return trimmed || DEFAULT_SITE_URL
}
