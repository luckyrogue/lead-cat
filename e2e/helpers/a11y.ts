import AxeBuilder from "@axe-core/playwright"
import { expect, type Page } from "@playwright/test"

type Impact = "critical" | "serious" | "moderate" | "minor"

// expectNoA11yViolations scans the current page with axe-core, logs every
// violation grouped by impact, and fails the test if any violation's impact is
// in blockImpacts (default: critical + serious).
export async function expectNoA11yViolations(
  page: Page,
  label: string,
  blockImpacts: Impact[] = ["critical", "serious"],
): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze()
  const violations = results.violations
  if (violations.length > 0) {
    const summary = violations
      .map(
        (v) =>
          `  [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} node(s))\n    ${v.helpUrl}`,
      )
      .join("\n")
    console.log(`a11y violations on ${label}:\n${summary}`)
  }
  const blocking = violations.filter(
    (v) => v.impact && (blockImpacts as string[]).includes(v.impact),
  )
  expect(
    blocking,
    `${label}: ${blocking.length} blocking a11y violation(s) — ${blocking
      .map((v) => `${v.impact}:${v.id}`)
      .join(", ")}`,
  ).toEqual([])
}
