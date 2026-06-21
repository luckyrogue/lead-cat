import { describe, expect, it } from "vitest"

import {
  getTimezoneOptions,
  getTimezoneOptionsWithEmpty,
} from "./timezone-options"

const t = (key: string) => `t:${key}`

describe("getTimezoneOptions", () => {
  it("maps every timezone to its value and a translated label", () => {
    const opts = getTimezoneOptions(t)
    expect(opts.length).toBeGreaterThan(0)
    expect(opts[0]).toEqual({
      value: "Asia/Almaty",
      label: "t:timezones.almaty",
    })
    expect(opts.map((o) => o.value)).toContain("UTC")
    expect(opts.every((o) => o.label.startsWith("t:timezones."))).toBe(true)
  })

  it("has no duplicate timezone values", () => {
    const values = getTimezoneOptions(t).map((o) => o.value)
    expect(new Set(values).size).toBe(values.length)
  })
})

describe("getTimezoneOptionsWithEmpty", () => {
  it("prepends an empty option using the given label key", () => {
    const opts = getTimezoneOptionsWithEmpty(t, "common.any")
    expect(opts).toHaveLength(getTimezoneOptions(t).length + 1)
    expect(opts[0]).toEqual({ value: "", label: "t:common.any" })
    expect(opts[1].value).toBe("Asia/Almaty")
  })
})
