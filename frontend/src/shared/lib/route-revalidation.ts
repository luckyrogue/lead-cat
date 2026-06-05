import type { ParsedLocation } from "@tanstack/react-router"

/** Skip route loader refetch when only search params changed (sadu shouldRevalidateExceptSearch). */
export function isSearchOnlyNavigation(
  current: ParsedLocation,
  previous: ParsedLocation | undefined
): boolean {
  if (!previous) return false
  return (
    current.pathname === previous.pathname &&
    current.searchStr !== previous.searchStr
  )
}

export function shouldReloadExceptSearch({
  location,
  previousLocation,
}: {
  location: ParsedLocation
  previousLocation?: ParsedLocation
}): boolean {
  return !isSearchOnlyNavigation(location, previousLocation)
}
