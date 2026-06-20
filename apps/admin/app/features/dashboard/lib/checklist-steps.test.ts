import { describe, expect, it } from "vitest"
import { computeSteps, allDone, doneCount } from "./checklist-steps"

const base = {
  connections: [] as { connected: boolean }[],
  membersCount: 1,
  invitesCount: 0,
  meetingsCount: 0,
}

describe("computeSteps", () => {
  it("marks calendar done when any connection is connected", () => {
    const steps = computeSteps({
      ...base,
      connections: [{ connected: false }, { connected: true }],
    })
    expect(steps.find((s) => s.key === "calendar")!.done).toBe(true)
  })
  it("marks invite done when there are invites OR more than one member", () => {
    expect(
      computeSteps({ ...base, invitesCount: 1 }).find((s) => s.key === "invite")!.done,
    ).toBe(true)
    expect(
      computeSteps({ ...base, membersCount: 2 }).find((s) => s.key === "invite")!.done,
    ).toBe(true)
    expect(
      computeSteps(base).find((s) => s.key === "invite")!.done,
    ).toBe(false)
  })
  it("marks meeting done when there is at least one meeting", () => {
    expect(
      computeSteps({ ...base, meetingsCount: 3 }).find((s) => s.key === "meeting")!.done,
    ).toBe(true)
  })
})

describe("allDone / doneCount", () => {
  it("counts and reports completion", () => {
    const steps = computeSteps({
      connections: [{ connected: true }],
      membersCount: 2,
      invitesCount: 0,
      meetingsCount: 1,
    })
    expect(doneCount(steps)).toBe(3)
    expect(allDone(steps)).toBe(true)
  })
  it("is not all done when a step is incomplete", () => {
    expect(allDone(computeSteps(base))).toBe(false)
  })
})
