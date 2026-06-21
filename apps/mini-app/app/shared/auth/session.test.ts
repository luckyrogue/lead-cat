import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  clearSession,
  getSession,
  getStoredUser,
  getToken,
  setSession,
  type AuthUser,
} from "./session"

const user: AuthUser = {
  telegram_id: 42,
  name: "Test User",
  email: "test@example.com",
  role: "member",
}

describe("session memory JWT", () => {
  const store = new Map<string, string>()

  beforeEach(() => {
    store.clear()
    vi.stubGlobal("window", {})
    vi.stubGlobal("sessionStorage", {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => {
        store.set(key, value)
      },
      removeItem: (key: string) => {
        store.delete(key)
      },
    })
    clearSession()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    clearSession()
  })

  it("keeps JWT in memory only", () => {
    setSession({ token: "secret-jwt", user })
    expect(getToken()).toBe("secret-jwt")
    expect(store.get("leadcat.tma.session")).not.toContain("secret-jwt")
    expect(getStoredUser()).toEqual(user)
  })

  it("returns null session when token is cleared from memory", () => {
    setSession({ token: "secret-jwt", user })
    clearSession()
    expect(getToken()).toBeNull()
    expect(getStoredUser()).toBeNull()
    expect(getSession()).toBeNull()
  })

  it("does not restore token from sessionStorage alone", () => {
    store.set("leadcat.tma.session", JSON.stringify({ user }))
    expect(getStoredUser()).toEqual(user)
    expect(getToken()).toBeNull()
    expect(getSession()).toBeNull()
  })
})
