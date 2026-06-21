import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { getInitData } from "./telegram-env"

describe("getInitData DEV guard", () => {
  beforeEach(() => {
    vi.stubGlobal("window", {
      Telegram: undefined,
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  it("uses VITE_TMA_DEV_TG_ID only in DEV mode", () => {
    vi.stubEnv("DEV", true)
    vi.stubEnv("VITE_TMA_DEV_TG_ID", "dev-telegram-id")
    expect(getInitData()).toBe("dev-telegram-id")
  })

  it("ignores VITE_TMA_DEV_TG_ID outside DEV mode", () => {
    vi.stubEnv("DEV", false)
    vi.stubEnv("VITE_TMA_DEV_TG_ID", "dev-telegram-id")
    expect(getInitData()).toBe("")
  })

  it("prefers Telegram initData over dev fallback", () => {
    vi.stubEnv("DEV", true)
    vi.stubEnv("VITE_TMA_DEV_TG_ID", "dev-telegram-id")
    vi.stubGlobal("window", {
      Telegram: {
        WebApp: {
          initData: "telegram-init-data",
        },
      },
    })
    expect(getInitData()).toBe("telegram-init-data")
  })
})
