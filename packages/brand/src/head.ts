import type { LinkDescriptor } from "react-router"

import { SITE_NAME, THEME_COLOR } from "./site"

export function brandHeadLinks(): LinkDescriptor[] {
  return [
    { rel: "icon", href: "/favicon.svg", type: "image/svg+xml" },
    { rel: "icon", href: "/favicon-32.png", sizes: "32x32", type: "image/png" },
    { rel: "icon", href: "/favicon-16.png", sizes: "16x16", type: "image/png" },
    { rel: "apple-touch-icon", href: "/apple-touch-icon.png", sizes: "180x180" },
    { rel: "manifest", href: "/site.webmanifest" },
  ]
}

export function brandMetaTags(options?: { title?: string; description?: string }) {
  const title = options?.title ?? SITE_NAME
  const description =
    options?.description ??
    "Meetings your team will actually love. Google Calendar sync and Telegram Mini App."

  return [
    { name: "application-name", content: SITE_NAME },
    { name: "theme-color", content: THEME_COLOR },
    { name: "msapplication-TileColor", content: THEME_COLOR },
    { title },
    { name: "description", content: description },
  ]
}
