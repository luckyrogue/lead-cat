import http from "k6/http"
import { check } from "k6"

const BASE = __ENV.LOAD_BASE_URL || "http://localhost:8090"
const SLUG = __ENV.LOAD_SLUG || "loadtest-intro"

// One burst well past the 60/min book_get limit — expect 429s, never 5xx.
export const options = {
  scenarios: {
    burst: {
      executor: "constant-arrival-rate",
      rate: 30,
      timeUnit: "1s",
      duration: "20s",
      preAllocatedVUs: 20,
      maxVUs: 50,
    },
  },
  thresholds: {
    // No 5xx is hard requirement; 429s are expected.
    "checks{check:no 5xx}": ["rate==1"],
  },
}

export default function () {
  const r = http.get(`${BASE}/api/book/${SLUG}`)
  check(r, { "no 5xx": (res) => res.status < 500 })
}

// handleSummary: print counts from k6 summary data.
// The http_req_failed metric counts all non-2xx/3xx (includes 429s).
// Zero "no 5xx" check failures => zero 5xx responses.
// Pass/fail decision (429s present + health after) is asserted in run.sh.
export function handleSummary(data) {
  const total = data.metrics["http_reqs"] ? data.metrics["http_reqs"].values["count"] : 0
  const failed = data.metrics["http_req_failed"] ? data.metrics["http_req_failed"].values["passes"] : 0
  const fivexx = data.metrics["checks"] ? (data.metrics["checks"].values["fails"] || 0) : 0

  return {
    stdout: `\nshedding_summary: total=${total} failed_non2xx=${failed} check_fails_5xx=${fivexx}\n`,
  }
}
