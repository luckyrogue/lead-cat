import { describe, expect, it } from "vitest"
import {
  canAccessMiniAppAdmin,
  getVisibleTabBarModules,
  miniappModulePolicies,
} from "@/shared/auth/module-policies"

describe("module-policies", () => {
  it("exposes all module routes", () => {
    expect(miniappModulePolicies.map((t) => t.href)).toEqual([
      "/",
      "/meetings",
      "/checker",
      "/profile",
      "/profile/admin",
    ])
  })

  it("grants admin panel only to admin role", () => {
    expect(canAccessMiniAppAdmin(null)).toBe(false)
    expect(
      canAccessMiniAppAdmin({
        telegramId: 1,
        name: "U",
        email: "u@x.kz",
        role: "user",
      })
    ).toBe(false)
    expect(
      canAccessMiniAppAdmin({
        telegramId: 2,
        name: "A",
        email: "a@x.kz",
        role: "admin",
      })
    ).toBe(true)
  })

  it("returns tab bar modules", () => {
    expect(getVisibleTabBarModules(null).map((m) => m.key)).toEqual([
      "home",
      "meetings",
      "checker",
      "profile",
    ])
  })
})
