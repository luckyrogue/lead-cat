import { resolveSiteUrl } from "@leadcat/brand"

import { robotsTxt } from "~/shared/seo/landing-meta"

export function loader() {
  return new Response(robotsTxt(resolveSiteUrl()), {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public, max-age=86400",
    },
  })
}
