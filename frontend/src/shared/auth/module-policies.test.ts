import { describe, expect, it } from "vitest"
import {
  canAccessTmaAdmin,
  getVisibleTabBarModules,
  tmaModulePolicies,
  visibleTmaTabs,
} from "@/shared/auth/module-policies"

describe("module-policies", () => {
  it("exposes all module routes", () => {
    expect(tmaModulePolicies.map((t) => t.href)).toEqual([
      "/",
      "/meetings",
      "/checker",
      "/auto",
      "/profile",
      "/profile/admin",
    ])
  })

  it("grants admin panel only to admin role", () => {
    expect(canAccessTmaAdmin(null)).toBe(false)
    expect(
      canAccessTmaAdmin({
        telegramId: 1,
        name: "U",
        email: "u@x.kz",
        role: "user",
      })
    ).toBe(false)
    expect(
      canAccessTmaAdmin({
        telegramId: 2,
        name: "A",
        email: "a@x.kz",
        role: "admin",
      })
    ).toBe(true)
  })

  it("returns tab bar modules without auto", () => {
    expect(getVisibleTabBarModules(null).map((m) => m.key)).toEqual([
      "home",
      "meetings",
      "checker",
      "profile",
    ])
  })

  it("returns all tabs for any authed user", () => {
    expect(visibleTmaTabs(null)).toHaveLength(5)
  })
})
