import { resolveSiteUrl } from "@leadcat/brand"

import { sitemapXml } from "~/shared/seo/landing-meta"

export function loader() {
  return new Response(sitemapXml(resolveSiteUrl()), {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "public, max-age=86400",
    },
  })
}
