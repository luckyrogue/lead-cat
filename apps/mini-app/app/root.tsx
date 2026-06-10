import { Toaster } from "@leadcat/ui"
import { QueryClientProvider } from "@tanstack/react-query"
import { useEffect } from "react"
import { Links, Meta, Outlet, Scripts, ScrollRestoration } from "react-router"

import { getQueryClient } from "~/shared/api/query-client"
import { AuthProvider } from "~/shared/auth/auth-context"
import { initTelegramViewport } from "~/shared/tma/telegram-env"

import "./app.css"

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, viewport-fit=cover"
        />
        <Meta />
        <Links />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  )
}

export default function App() {
  const queryClient = getQueryClient()

  useEffect(() => {
    initTelegramViewport()
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Outlet />
        <Toaster richColors position="top-center" />
      </AuthProvider>
    </QueryClientProvider>
  )
}
