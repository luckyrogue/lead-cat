import { describe, expect, it } from "vitest"
import { parseUrlLocale, localeCookieHeader } from "./locale-request"

describe("parseUrlLocale", () => {
  it("accepts url locales", () => {
    expect(parseUrlLocale("en")).toBe("en")
    expect(parseUrlLocale("kk")).toBe("kk")
  })
  it("rejects the default locale (ru is not a url locale) and garbage", () => {
    expect(parseUrlLocale("ru")).toBeNull()
    expect(parseUrlLocale("xx")).toBeNull()
    expect(parseUrlLocale(undefined)).toBeNull()
  })
})

describe("localeCookieHeader", () => {
  it("builds a year-long lax cookie", () => {
    const header = localeCookieHeader("en")
    expect(header).toContain("leadcat_locale=en")
    expect(header).toContain("Path=/")
    expect(header).toContain("Max-Age=31536000")
    expect(header).toContain("SameSite=Lax")
  })
})
