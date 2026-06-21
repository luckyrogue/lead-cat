#!/usr/bin/env bash
# load/run.sh — k6 load harness: capacity (limits off) + shedding (limits on)
# Usage: bash load/run.sh [capacity|shedding|all]
set -euo pipefail
cd "$(dirname "$0")/.."

BASE_COMPOSE="docker compose -f deploy/docker-compose.e2e.yml"
# Capacity: disables rate limits + exposes backend on :8081
CAP_COMPOSE="${BASE_COMPOSE} -f deploy/docker-compose.load.yml"
# Shedding: keeps rate limits ON + exposes backend on :8081
SHED_COMPOSE="${BASE_COMPOSE} -f deploy/docker-compose.load-shedding.yml"
PROJECT_FLAGS="-p leadcat-load"

# k6 talks directly to backend on 8081 to avoid nginx ephemeral-port exhaustion
# on macOS Docker Desktop (nginx -> backend through bridge exhausts local ports
# at high RPS; backend direct connection is also more accurate for latency).
# On Linux --network host also works; this path is portable to both.
if [ "$(uname)" = "Darwin" ]; then
  K6_BACKEND_URL="http://host.docker.internal:8081"
  K6_NETWORK_FLAG=""
else
  K6_BACKEND_URL="http://localhost:8081"
  K6_NETWORK_FLAG="--network host"
fi

scenario="${1:-all}"

cleanup() {
  echo "[run.sh] tearing down stack..."
  ${BASE_COMPOSE} ${PROJECT_FLAGS} down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_healthy() {
  local port="${1:-8081}"
  echo "[run.sh] waiting for backend on :${port} (up to 3 min)..."
  for i in $(seq 1 60); do
    if curl -fsS "http://localhost:${port}/api/health" >/dev/null 2>&1; then
      echo "[run.sh] backend healthy after $((i * 3))s"
      return 0
    fi
    sleep 3
  done
  echo "[run.sh] ERROR: backend not healthy after 180s"
  ${BASE_COMPOSE} ${PROJECT_FLAGS} logs --tail=80
  exit 1
}

seed() {
  echo "[run.sh] seeding database..."
  ${BASE_COMPOSE} ${PROJECT_FLAGS} exec -T postgres \
    psql -U notify -d notify -v ON_ERROR_STOP=1 < load/seed.sql
  echo "[run.sh] verifying seed: GET /api/book/loadtest-intro"
  if ! curl -fsS "http://localhost:8081/api/book/loadtest-intro" >/dev/null; then
    echo "[run.sh] ERROR: seed verify failed — /api/book/loadtest-intro not 200"
    exit 1
  fi
  echo "[run.sh] seed OK"
}

run_k6() {
  local script="$1"
  echo "[run.sh] running k6: ${script} → ${K6_BACKEND_URL}"
  # shellcheck disable=SC2086
  docker run --rm -i \
    ${K6_NETWORK_FLAG} \
    -e LOAD_BASE_URL="${K6_BACKEND_URL}" \
    -v "${PWD}/load:/load" \
    grafana/k6 run /load/"${script}"
}

run_capacity() {
  echo "=== CAPACITY (rate limits disabled, backend direct :8081) ==="
  cleanup
  ${CAP_COMPOSE} ${PROJECT_FLAGS} up -d --build
  wait_healthy 8081
  seed
  run_k6 capacity.js
  echo "[run.sh] capacity run complete."
}

run_shedding() {
  echo "=== SHEDDING (rate limits enabled, backend direct :8081) ==="
  cleanup
  ${SHED_COMPOSE} ${PROJECT_FLAGS} up -d --build
  wait_healthy 8081
  seed
  # Capture k6 output; parse the shedding_summary line for pass/fail.
  k6_out=$(run_k6 shedding.js 2>&1 || true)
  echo "${k6_out}"

  # Extract counts from the summary line emitted by handleSummary.
  shed_line=$(echo "${k6_out}" | grep "shedding_summary:" || echo "shedding_summary: total=0 failed_non2xx=0 check_fails_5xx=0")
  total=$(echo "${shed_line}"     | grep -oE 'total=[0-9]+' | cut -d= -f2)
  failed=$(echo "${shed_line}"    | grep -oE 'failed_non2xx=[0-9]+' | cut -d= -f2)
  fivexx=$(echo "${shed_line}"    | grep -oE 'check_fails_5xx=[0-9]+' | cut -d= -f2)
  total=${total:-0}; failed=${failed:-0}; fivexx=${fivexx:-0}

  echo "[run.sh] confirming health after shedding burst..."
  if curl -fsS "http://localhost:8081/api/health" >/dev/null 2>&1; then
    health_ok=true
    echo "[run.sh] health after burst: OK (200)"
  else
    health_ok=false
    echo "[run.sh] ERROR: health after burst: FAILED"
  fi

  echo ""
  # 429s present = failed_non2xx > 0 (429 is non-2xx so k6 counts it as failed).
  got_429=false; [ "${failed:-0}" -gt 0 ] && got_429=true
  no_5xx=true;   [ "${fivexx:-0}" -gt 0 ] && no_5xx=false

  if ${got_429} && ${no_5xx} && ${health_ok}; then
    echo "[run.sh] SHEDDING PASS: total=${total} 429s=${failed} 5xx=0 health=200"
  else
    echo "[run.sh] SHEDDING FAIL: total=${total} 429s=${failed} 5xx_check_fails=${fivexx} health_ok=${health_ok}"
    exit 1
  fi
}

case "${scenario}" in
  capacity) run_capacity ;;
  shedding) run_shedding ;;
  all)      run_capacity; run_shedding ;;
  *) echo "usage: run.sh [capacity|shedding|all]"; exit 2 ;;
esac

echo "[run.sh] done."
