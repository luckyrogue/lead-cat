import { describe, expect, it } from "vitest"

import type { Me } from "~/shared/auth/types"

import { patchMeSettings, patchMeUser } from "./queries"

const me: Me = {
  user: {
    id: "u1",
    email: "a@b.c",
    avatar_url: "",
    auth_method: "web",
    timezone: "Asia/Almaty",
    language: "ru",
  },
  organizations: [{ id: "o1", name: "Org", slug: "org" }],
}

describe("patchMeSettings", () => {
  it("overrides only the provided fields, keeping the rest from current", () => {
    expect(
      patchMeSettings({ timezone: "UTC", language: "en" }, { language: "kk" })
    ).toEqual({ timezone: "UTC", language: "kk" })
  })

  it("falls back to empty strings when neither prefs nor current have a value", () => {
    expect(patchMeSettings(undefined, {})).toEqual({
      timezone: "",
      language: "",
    })
  })
})

describe("patchMeUser", () => {
  it("returns the cache value unchanged when empty (no entry to seed)", () => {
    expect(patchMeUser(null, { language: "kk" })).toBeNull()
    expect(patchMeUser(undefined, { language: "kk" })).toBeUndefined()
  })

  it("patches only timezone when only timezone is provided", () => {
    const next = patchMeUser(me, { timezone: "UTC" })
    expect(next?.user.timezone).toBe("UTC")
    expect(next?.user.language).toBe("ru")
  })

  it("patches only language when only language is provided", () => {
    const next = patchMeUser(me, { language: "kk" })
    expect(next?.user.language).toBe("kk")
    expect(next?.user.timezone).toBe("Asia/Almaty")
  })

  it("preserves all other user fields and organizations", () => {
    const next = patchMeUser(me, { language: "en", timezone: "UTC" })
    expect(next?.user.id).toBe("u1")
    expect(next?.user.email).toBe("a@b.c")
    expect(next?.organizations).toEqual(me.organizations)
  })

  it("writes the explicit empty-string 'default language' choice (not skipped)", () => {
    const next = patchMeUser(me, { language: "" })
    expect(next?.user.language).toBe("")
  })

  it("does not mutate the input object", () => {
    patchMeUser(me, { language: "kk" })
    expect(me.user.language).toBe("ru")
  })
})
