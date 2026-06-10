import { useCallback, useSyncExternalStore } from "react"

import { getActiveOrgId, setActiveOrgId } from "~/shared/api/active-org"
import type { MeOrganization } from "~/shared/auth/types"

const listeners = new Set<() => void>()

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

function emit() {
  listeners.forEach((listener) => listener())
}

function useStoredOrgId() {
  return useSyncExternalStore(subscribe, getActiveOrgId, () => null)
}

export function useActiveOrg(organizations: MeOrganization[]) {
  const storedId = useStoredOrgId()
  const activeOrg =
    organizations.find((org) => org.id === storedId) ?? organizations[0] ?? null

  const selectOrg = useCallback((id: string) => {
    setActiveOrgId(id)
    emit()
  }, [])

  return { activeOrg, activeOrgId: activeOrg?.id ?? null, selectOrg }
}

export function ensureActiveOrg(organizations: MeOrganization[]) {
  if (organizations.length === 0) {
    return
  }
  const stored = getActiveOrgId()
  const valid = stored && organizations.some((org) => org.id === stored)
  if (!valid) {
    setActiveOrgId(organizations[0].id)
    emit()
  }
}
