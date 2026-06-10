import { createFileRoute } from "@tanstack/react-router"
import { LoginPage } from "@/features/web-login/login-page"

export const Route = createFileRoute("/login")({
  component: LoginPage,
})
