import { ACKEE_BASE_PATH, brandHeadLinks, brandMetaTags } from "@leadcat/brand"
import { Toaster } from "@leadcat/ui"
import { QueryClientProvider } from "@tanstack/react-query"
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

import "./app.css"

const ACKEE_DOMAIN_ID = "4f95b6c1-5a45-42bf-bbad-dd3e2f4d373a"

export function links() {
  return brandHeadLinks()
}

export function meta() {
  return brandMetaTags({
    title: "Lead Cat Admin",
    description: "Admin dashboard for Lead Cat meeting scheduling.",
  })
}

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
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

  return (
    <QueryClientProvider client={queryClient}>
      <Outlet />
      <Toaster position="top-right" />
    </QueryClientProvider>
  )
}

export function ErrorBoundary({ error }: { error: ErrorResponse | Error }) {
  return <RouteErrorPage error={error} />
}
