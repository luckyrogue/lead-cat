const KEY = "lc_checklist_dismissed"

export function isChecklistDismissed(): boolean {
  if (typeof window === "undefined") {
    return false
  }
  return window.localStorage.getItem(KEY) === "1"
}

export function dismissChecklist(): void {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(KEY, "1")
}
