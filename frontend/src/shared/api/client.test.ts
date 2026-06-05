import { describe, expect, it } from "vitest"
import { toApiError } from "@/shared/api/client"
import { ApiError } from "@/shared/api/types"

describe("api client errors", () => {
  it("maps TMA auth code to ApiError", () => {
    const err = toApiError({
      isAxiosError: true,
      response: { status: 401, data: { code: "not_registered" } },
      message: "Request failed",
      config: {},
    })

    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(401)
    expect(err.code).toBe("not_registered")
    expect(err.message).toBe("not_registered")
  })

  it("maps backend message field", () => {
    const err = toApiError({
      isAxiosError: true,
      response: {
        status: 400,
        data: { error: "error", message: "bad request" },
      },
      message: "Request failed",
      config: {},
    })

    expect(err.message).toBe("bad request")
    expect(err.status).toBe(400)
  })
})
