import { resolveSiteUrl, SITE_NAME } from "@leadcat/brand"
import { dictionaries } from "~/shared/i18n/dictionaries"
import { localePath } from "~/shared/i18n/locale-path"
import { LOCALES, type Locale } from "~/shared/i18n/types"

function absoluteUrl(pathname: string, siteUrl = resolveSiteUrl()): string {
  if (!pathname.startsWith("/")) {
    return `${siteUrl}/${pathname}`
  }
  return `${siteUrl}${pathname === "/" ? "" : pathname}`
}

export function landingCanonicalPath(locale: Locale): string {
  return localePath(locale)
}

export function landingJsonLd(locale: Locale, siteUrl = resolveSiteUrl()) {
  const dict = dictionaries[locale]
  const pageUrl = absoluteUrl(landingCanonicalPath(locale), siteUrl)

  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "Organization",
        "@id": `${siteUrl}/#organization`,
        name: SITE_NAME,
        url: siteUrl,
        logo: `${siteUrl}/logo.svg`,
      },
      {
        "@type": "WebSite",
        "@id": `${siteUrl}/#website`,
        url: siteUrl,
        name: SITE_NAME,
        description: dict.meta.description,
        inLanguage: LOCALES,
        publisher: { "@id": `${siteUrl}/#organization` },
      },
      {
        "@type": "WebPage",
        "@id": `${pageUrl}#webpage`,
        url: pageUrl,
        name: dict.meta.title,
        description: dict.meta.description,
        isPartOf: { "@id": `${siteUrl}/#website` },
        inLanguage: locale,
      },
      {
        "@type": "SoftwareApplication",
        name: SITE_NAME,
        applicationCategory: "BusinessApplication",
        operatingSystem: "Web",
        description: dict.meta.description,
        offers: {
          "@type": "Offer",
          price: "0",
          priceCurrency: "USD",
        },
        featureList: [
          "Google Calendar sync",
          "Telegram Mini App",
          "Team scheduling",
          "Meeting reminders",
        ],
      },
    ],
  }
}

export function landingMeta(
  locale: Locale,
  options?: { pathname?: string; siteUrl?: string }
) {
  const dict = dictionaries[locale]
  const siteUrl = options?.siteUrl ?? resolveSiteUrl()
  const canonicalPath = options?.pathname ?? landingCanonicalPath(locale)
  const canonical = absoluteUrl(canonicalPath, siteUrl)
  const image = `${siteUrl}/og-image.png`
  const jsonLd = landingJsonLd(locale, siteUrl)

  return [
    { html: { lang: locale } },
    { title: dict.meta.title },
    { name: "description", content: dict.meta.description },
    { name: "keywords", content: dict.meta.keywords },
    { name: "robots", content: "index,follow,max-image-preview:large" },
    { name: "author", content: SITE_NAME },
    { tagName: "link", rel: "canonical", href: canonical },
    { property: "og:type", content: "website" },
    { property: "og:site_name", content: SITE_NAME },
    { property: "og:title", content: dict.meta.title },
    { property: "og:description", content: dict.meta.description },
    { property: "og:url", content: canonical },
    { property: "og:image", content: image },
    { property: "og:image:width", content: "1200" },
    { property: "og:image:height", content: "630" },
    { property: "og:image:alt", content: dict.meta.title },
    { property: "og:locale", content: dict.meta.ogLocale },
    ...LOCALES.filter((code) => code !== locale).map((code) => ({
      property: "og:locale:alternate",
      content: dictionaries[code].meta.ogLocale,
    })),
    { name: "twitter:card", content: "summary_large_image" },
    { name: "twitter:title", content: dict.meta.title },
    { name: "twitter:description", content: dict.meta.description },
    { name: "twitter:image", content: image },
    { name: "twitter:image:alt", content: dict.meta.title },
    {
      tagName: "link",
      rel: "alternate",
      hrefLang: "ru",
      href: absoluteUrl("/", siteUrl),
    },
    {
      tagName: "link",
      rel: "alternate",
      hrefLang: "en",
      href: absoluteUrl("/en", siteUrl),
    },
    {
      tagName: "link",
      rel: "alternate",
      hrefLang: "kk",
      href: absoluteUrl("/kk", siteUrl),
    },
    {
      tagName: "link",
      rel: "alternate",
      hrefLang: "x-default",
      href: absoluteUrl("/", siteUrl),
    },
    {
      "script:ld+json": jsonLd,
    },
  ]
}

export function sitemapXml(siteUrl = resolveSiteUrl()): string {
  const pages = [
    { loc: "/", changefreq: "weekly", priority: "1.0" },
    { loc: "/en", changefreq: "weekly", priority: "0.9" },
    { loc: "/kk", changefreq: "weekly", priority: "0.9" },
  ]

  const urls = pages
    .map(
      (page) => `  <url>
    <loc>${absoluteUrl(page.loc, siteUrl)}</loc>
    <changefreq>${page.changefreq}</changefreq>
    <priority>${page.priority}</priority>
  </url>`
    )
    .join("\n")

  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls}
</urlset>
`
}

export function robotsTxt(siteUrl = resolveSiteUrl()): string {
  return `User-agent: *
Allow: /

Sitemap: ${absoluteUrl("/sitemap.xml", siteUrl)}
`
}
