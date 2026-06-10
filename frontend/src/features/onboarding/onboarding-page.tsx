import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { useCreateOrg } from "@/shared/web-auth/queries"

export function OnboardingPage() {
  const [orgName, setOrgName] = useState("")
  const [touched, setTouched] = useState(false)
  const navigate = useNavigate()
  const createOrg = useCreateOrg()

  const isEmpty = orgName.trim().length === 0
  const showError = touched && isEmpty

  function handleSubmit() {
    setTouched(true)
    if (isEmpty) return
    createOrg.mutate(orgName.trim(), {
      onSuccess: () => {
        void navigate({ to: "/dashboard" })
      },
    })
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm space-y-6 rounded-2xl bg-white p-8 shadow-sm">
        <div className="space-y-1 text-center">
          <h1 className="text-2xl font-bold tracking-tight text-gray-900">
            Create your organization
          </h1>
          <p className="text-sm text-gray-500">
            Give your workspace a name to get started.
          </p>
        </div>

        <div className="space-y-3">
          <div className="space-y-1">
            <input
              type="text"
              placeholder="Acme Inc."
              value={orgName}
              onChange={(e) => {
                setOrgName(e.target.value)
                setTouched(true)
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSubmit()
              }}
              disabled={createOrg.isPending}
              className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:opacity-50"
            />
            {showError && (
              <p className="text-xs text-red-500">Organization name is required.</p>
            )}
          </div>

          {createOrg.isError && (
            <p className="text-xs text-red-500">
              Failed to create organization. Please try again.
            </p>
          )}

          <button
            type="button"
            onClick={handleSubmit}
            disabled={createOrg.isPending}
            className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700 active:bg-blue-800 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {createOrg.isPending ? "Creating…" : "Create organization"}
          </button>
        </div>
      </div>
    </div>
  )
}
