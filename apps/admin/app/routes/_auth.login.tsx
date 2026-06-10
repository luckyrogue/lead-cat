import { useEffect } from "react"
import { useSearchParams } from "react-router"

import { LoginForm } from "~/features/auth/components/login-form"
import { toastError } from "~/shared/lib/toast"

export default function LoginPage() {
  const [params] = useSearchParams()
  const error = params.get("error")

  useEffect(() => {
    if (error === "invalid_link") {
      toastError(
        new Error("That sign-in link is invalid or has expired."),
        "Sign-in failed."
      )
    }
  }, [error])

  return <LoginForm />
}
