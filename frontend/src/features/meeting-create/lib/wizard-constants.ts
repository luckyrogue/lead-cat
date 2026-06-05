export const WIZARD_STEPS = ["what", "when", "who", "review"] as const
export type WizardStep = (typeof WIZARD_STEPS)[number]

export const TIME_SLOTS: string[] = (() => {
  const times: string[] = []
  for (let h = 8; h <= 19; h++) {
    for (const mn of [0, 30]) {
      times.push(`${String(h).padStart(2, "0")}:${String(mn).padStart(2, "0")}`)
    }
  }
  return times
})()

export const DURATION_OPTIONS = [15, 30, 45, 60, 90, 120] as const
