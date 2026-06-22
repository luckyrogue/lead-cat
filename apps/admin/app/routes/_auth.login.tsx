import { useEffect } from "react"
import { useSearchParams } from "react-router"

import { LoginForm } from "~/features/auth/components/login-form"
import { useT } from "~/shared/i18n/context"
import { toastError } from "@leadcat/ui"

export default function LoginPage() {
  const t = useT()
  const [params] = useSearchParams()
  const error = params.get("error")

  useEffect(() => {
    if (error === "invalid_link") {
      toastError(undefined, t, "auth.login.invalidLinkMessage")
    }
  }, [error, t])

  return <LoginForm />
}
