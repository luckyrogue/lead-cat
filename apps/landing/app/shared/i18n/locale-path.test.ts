import { describe, expect, it } from "vitest"
import { localePath } from "./locale-path"

describe("localePath", () => {
  it("returns root for the default locale (ru)", () => {
    expect(localePath("ru")).toBe("/")
    expect(localePath("ru", "#features")).toBe("/#features")
  })
  it("prefixes non-default locales", () => {
    expect(localePath("en")).toBe("/en")
    expect(localePath("kk", "#pricing")).toBe("/kk#pricing")
  })
})
