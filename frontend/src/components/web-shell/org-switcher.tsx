import { useWebAuth } from "@/shared/web-auth/context"

export function OrgSwitcher() {
  const { organizations, activeOrgId, setActiveOrgId } = useWebAuth()

  if (organizations.length === 0) return null

  return (
    <select
      value={activeOrgId ?? ""}
      onChange={(e) => setActiveOrgId(e.target.value)}
      className="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
    >
      {organizations.map((org) => (
        <option key={org.id} value={org.id}>
          {org.name}
        </option>
      ))}
    </select>
  )
}
