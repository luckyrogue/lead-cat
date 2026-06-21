import { verifyMagicLink } from "~/shared/auth/api"
import { PageLoading } from "~/components/page-loading"
import { useT } from "~/shared/i18n/context"
import { useEffect, useRef } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { toastError } from "~/shared/lib/toast"

export default function MagicLinkVerifyPage() {
  const t = useT()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const started = useRef(false)

  useEffect(() => {
    if (started.current) {
      return
    }
    started.current = true
    const token = params.get("token")?.trim() ?? ""
    if (!token) {
      navigate("/login?error=invalid_link", { replace: true })
      return
    }
    void (async () => {
      try {
        const dest = await verifyMagicLink(token)
        window.history.replaceState({}, "", "/auth/magic")
        navigate(dest, { replace: true })
      } catch (error) {
        toastError(error, t("auth.login.invalidLinkMessage"))
        navigate("/login?error=invalid_link", { replace: true })
      }
    })()
  }, [navigate, params, t])

  return (
    <div className="flex min-h-40 items-center justify-center">
      <PageLoading>{t("auth.loading")}</PageLoading>
    </div>
  )
}
