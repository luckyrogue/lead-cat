import http from "k6/http"
import { check } from "k6"

const BASE = __ENV.LOAD_BASE_URL || "http://localhost:8090"
const SLUG = __ENV.LOAD_SLUG || "loadtest-intro"

export const options = {
  scenarios: {
    health: {
      executor: "ramping-vus",
      exec: "health",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 20 },
        { duration: "20s", target: 20 },
        { duration: "5s", target: 0 },
      ],
    },
    book: {
      executor: "ramping-vus",
      exec: "book",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 20 },
        { duration: "20s", target: 20 },
        { duration: "5s", target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    "http_req_duration{scenario:health}": ["p(95)<200"],
    "http_req_duration{scenario:book}": ["p(95)<500"],
  },
}

export function health() {
  const r = http.get(`${BASE}/api/health`)
  check(r, { "health 200": (res) => res.status === 200 })
}

export function book() {
  const r = http.get(`${BASE}/api/book/${SLUG}`)
  check(r, { "book 200": (res) => res.status === 200 })
}
