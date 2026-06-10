const STORAGE_KEY = "lc.admin.activeOrg"

export function getActiveOrgId(): string | null {
  if (typeof window === "undefined") {
    return null
  }
  return window.localStorage.getItem(STORAGE_KEY)
}

export function setActiveOrgId(id: string | null) {
  if (typeof window === "undefined") {
    return
  }
  if (id) {
    window.localStorage.setItem(STORAGE_KEY, id)
  } else {
    window.localStorage.removeItem(STORAGE_KEY)
  }
}
