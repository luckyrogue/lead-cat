import { describe, expect, it } from "vitest"

import { meetingDetailQuery } from "./queries"

describe("meetingDetailQuery", () => {
  it("uses detail query key and is disabled for empty id", () => {
    const q = meetingDetailQuery("")
    expect(q.queryKey).toEqual(["meetings", "detail", ""])
    expect(q.enabled).toBe(false)
  })

  it("enables fetch when id is set", () => {
    const q = meetingDetailQuery("abc-123")
    expect(q.queryKey).toEqual(["meetings", "detail", "abc-123"])
    expect(q.enabled).toBe(true)
  })
})
