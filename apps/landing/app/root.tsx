import { ACKEE_BASE_PATH, brandHeadLinks } from "@leadcat/brand"
import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  type ErrorResponse,
} from "react-router"

import { RouteErrorPage } from "~/components/route-error-page"

import "./app.css"

const ACKEE_DOMAIN_ID = "9cc5108c-dac6-4a3f-9c87-ad9570eb39d4"

export function links() {
  return brandHeadLinks()
}

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" suppressHydrationWarning>
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
  return <Outlet />
}

export function ErrorBoundary({ error }: { error: ErrorResponse | Error }) {
  return <RouteErrorPage error={error} />
}
