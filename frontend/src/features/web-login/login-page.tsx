import { useState } from "react"
import { requestMagicLink, ssoStartUrl } from "@/shared/web-auth/api"

type MagicLinkState = "idle" | "loading" | "sent" | "error"

export function LoginPage() {
  const [email, setEmail] = useState("")
  const [magicState, setMagicState] = useState<MagicLinkState>("idle")

  async function handleMagicLink() {
    if (!email.trim()) return
    setMagicState("loading")
    try {
      await requestMagicLink(email.trim())
      setMagicState("sent")
    } catch {
      setMagicState("error")
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm space-y-6 rounded-2xl bg-white p-8 shadow-sm">
        <div className="space-y-1 text-center">
          <h1 className="text-2xl font-bold tracking-tight text-gray-900">
            Sign in to Lead Cat
          </h1>
          <p className="text-sm text-gray-500">Choose how you want to continue</p>
        </div>

        <div className="space-y-3">
          <button
            type="button"
            onClick={() => {
              window.location.href = ssoStartUrl("google")
            }}
            className="flex w-full items-center justify-center gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 active:bg-gray-100"
          >
            <GoogleIcon />
            Continue with Google
          </button>

          <button
            type="button"
            onClick={() => {
              window.location.href = ssoStartUrl("microsoft")
            }}
            className="flex w-full items-center justify-center gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 active:bg-gray-100"
          >
            <MicrosoftIcon />
            Continue with Microsoft
          </button>
        </div>

        <div className="flex items-center gap-3">
          <div className="h-px flex-1 bg-gray-200" />
          <span className="text-xs text-gray-400">or</span>
          <div className="h-px flex-1 bg-gray-200" />
        </div>

        {magicState === "sent" ? (
          <div className="rounded-xl bg-green-50 px-4 py-3 text-center text-sm text-green-700">
            Check your email — we sent a sign-in link to <strong>{email}</strong>.
          </div>
        ) : (
          <div className="space-y-3">
            <input
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleMagicLink()
              }}
              className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm text-gray-900 placeholder-gray-400 outline-none ring-0 transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            />
            {magicState === "error" && (
              <p className="text-xs text-red-500">
                Failed to send magic link. Please try again.
              </p>
            )}
            <button
              type="button"
              onClick={() => void handleMagicLink()}
              disabled={magicState === "loading" || !email.trim()}
              className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700 active:bg-blue-800 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {magicState === "loading" ? "Sending…" : "Send magic link"}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
      <path
        d="M17.64 9.205c0-.638-.057-1.252-.164-1.841H9v3.481h4.844c-.209 1.125-.843 2.078-1.796 2.717v2.258h2.908c1.702-1.567 2.684-3.875 2.684-6.615z"
        fill="#4285F4"
      />
      <path
        d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.259c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332A8.997 8.997 0 009 18z"
        fill="#34A853"
      />
      <path
        d="M3.964 10.71A5.41 5.41 0 013.682 9c0-.593.102-1.17.282-1.71V4.958H.957A8.996 8.996 0 000 9c0 1.452.348 2.827.957 4.042l3.007-2.332z"
        fill="#FBBC05"
      />
      <path
        d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0A8.997 8.997 0 00.957 4.958L3.964 7.29C4.672 5.163 6.656 3.58 9 3.58z"
        fill="#EA4335"
      />
    </svg>
  )
}

function MicrosoftIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
      <rect x="1" y="1" width="7.5" height="7.5" fill="#F25022" />
      <rect x="9.5" y="1" width="7.5" height="7.5" fill="#7FBA00" />
      <rect x="1" y="9.5" width="7.5" height="7.5" fill="#00A4EF" />
      <rect x="9.5" y="9.5" width="7.5" height="7.5" fill="#FFB900" />
    </svg>
  )
}
