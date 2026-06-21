import { describe, expect, it } from "vitest"

import { assertAllowedOAuthUrl, OAuthUrlError } from "./oauth-url"

describe("assertAllowedOAuthUrl", () => {
  it("allows Google OAuth URLs", () => {
    const url =
      "https://accounts.google.com/o/oauth2/auth?client_id=abc&redirect_uri=https%3A%2F%2Fapp.example"
    expect(assertAllowedOAuthUrl(url)).toBe(url)
  })

  it("allows Microsoft OAuth URLs", () => {
    const url =
      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=abc"
    expect(assertAllowedOAuthUrl(url)).toBe(url)
  })

  it("rejects non-HTTPS URLs", () => {
    expect(() =>
      assertAllowedOAuthUrl("http://accounts.google.com/o/oauth2/auth")
    ).toThrow(OAuthUrlError)
  })

  it("rejects untrusted hosts", () => {
    expect(() => assertAllowedOAuthUrl("https://evil.example/oauth")).toThrow(
      OAuthUrlError
    )
  })

  it("rejects malformed URLs", () => {
    expect(() => assertAllowedOAuthUrl("not-a-url")).toThrow(OAuthUrlError)
  })
})
