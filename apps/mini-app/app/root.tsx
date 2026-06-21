import { ACKEE_BASE_PATH, brandHeadLinks, brandMetaTags } from "@leadcat/brand"
import { Toaster } from "@leadcat/ui"
import { QueryClientProvider } from "@tanstack/react-query"
import { useEffect } from "react"
import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  type ErrorResponse,
} from "react-router"

import { RouteErrorPage } from "~/components/route-error-page"
import { getQueryClient } from "~/shared/api/query-client"
import { AuthProvider } from "~/shared/auth/auth-context"
import { initTelegramViewport } from "~/shared/tma/telegram-env"

import "./app.css"

const ACKEE_DOMAIN_ID = "fa8b7e8a-e583-47d9-98c0-1e36486e61c4"

export function links() {
  return [
    {
      rel: "preconnect",
      href: "https://telegram.org",
    },
    ...brandHeadLinks(),
  ]
}

export function meta() {
  return brandMetaTags({
    title: "Lead Cat",
    description: "Schedule meetings in Telegram with Google Calendar sync.",
  })
}

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, viewport-fit=cover"
        />
        <script src="https://telegram.org/js/telegram-web-app.js" />
        <Meta />
        <Links />
        {import.meta.env.PROD ? (
          <script
            async
            src={`${ACKEE_BASE_PATH}/tracker.js`}
            data-ackee-server={ACKEE_BASE_PATH}
            data-ackee-domain-id={ACKEE_DOMAIN_ID}
          />
        ) : null}
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

export function ErrorBoundary({ error }: { error: ErrorResponse | Error }) {
  return <RouteErrorPage error={error} />
}
