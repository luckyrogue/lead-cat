export type ChecklistStepKey = "calendar" | "invite" | "meeting"

export type ChecklistStep = { key: ChecklistStepKey; done: boolean }

export function computeSteps(input: {
  connections: { connected: boolean }[]
  membersCount: number
  invitesCount: number
  meetingsCount: number
}): ChecklistStep[] {
  return [
    { key: "calendar", done: input.connections.some((c) => c.connected) },
    { key: "invite", done: input.invitesCount > 0 || input.membersCount > 1 },
    { key: "meeting", done: input.meetingsCount > 0 },
  ]
}

export function allDone(steps: ChecklistStep[]): boolean {
  return steps.every((s) => s.done)
}

export function doneCount(steps: ChecklistStep[]): number {
  return steps.filter((s) => s.done).length
}
