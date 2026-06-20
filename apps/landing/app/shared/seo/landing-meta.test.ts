import { describe, expect, it } from "vitest"
import {
  landingCanonicalPath,
  sitemapXml,
  robotsTxt,
  landingMeta,
} from "./landing-meta"

const SITE = "https://example.test"

describe("landingCanonicalPath", () => {
  it("maps locales to their canonical path", () => {
    expect(landingCanonicalPath("ru")).toBe("/")
    expect(landingCanonicalPath("en")).toBe("/en")
  })
})

describe("sitemapXml", () => {
  it("lists the localized roots with absolute URLs", () => {
    const xml = sitemapXml(SITE)
    expect(xml).toContain("<loc>https://example.test</loc>")
    expect(xml).toContain("<loc>https://example.test/en</loc>")
    expect(xml).toContain("<loc>https://example.test/kk</loc>")
    expect(xml.startsWith("<?xml")).toBe(true)
  })
})

describe("robotsTxt", () => {
  it("points at the absolute sitemap URL", () => {
    expect(robotsTxt(SITE)).toContain(
      "Sitemap: https://example.test/sitemap.xml"
    )
  })
})

describe("landingMeta", () => {
  it("emits a canonical link and html lang for the locale", () => {
    const tags = landingMeta("en", { siteUrl: SITE })
    expect(tags).toContainEqual({ html: { lang: "en" } })
    expect(tags).toContainEqual({
      tagName: "link",
      rel: "canonical",
      href: "https://example.test/en",
    })
  })
})
