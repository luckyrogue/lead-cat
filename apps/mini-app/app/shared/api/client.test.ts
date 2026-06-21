import { describe, expect, it, vi } from "vitest"

import { setReauthHandler, setSessionInvalidatedHandler } from "./client"

describe("mini-app api session handlers", () => {
  it("registers and clears session invalidation handler", () => {
    const handler = vi.fn()
    setSessionInvalidatedHandler(handler)
    setSessionInvalidatedHandler(null)
    expect(handler).not.toHaveBeenCalled()
  })

  it("registers and clears reauth handler", () => {
    const handler = vi.fn(async () => "token")
    setReauthHandler(handler)
    setReauthHandler(null)
    expect(handler).not.toHaveBeenCalled()
  })
})
