#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/backend"

read_cov() {
  env -u GOROOT go test -count=1 -cover "$1" 2>&1 | awk '/coverage:/ { gsub(/%/, "", $5); print $5; exit }'
}

middleware_pct="$(read_cov ./internal/delivery/http/middleware/...)"
scenario_pct="$(read_cov ./internal/domain/scenario/...)"

min=50

echo "middleware coverage: ${middleware_pct}%"
echo "scenario coverage: ${scenario_pct}%"

fail=0
if awk -v c="${middleware_pct:-0}" -v m="$min" 'BEGIN { exit (c+0 >= m+0) ? 0 : 1 }'; then
  :
else
  echo "FAIL: middleware coverage ${middleware_pct}% < ${min}%"
  fail=1
fi
if awk -v c="${scenario_pct:-0}" -v m="$min" 'BEGIN { exit (c+0 >= m+0) ? 0 : 1 }'; then
  :
else
  echo "FAIL: scenario coverage ${scenario_pct}% < ${min}%"
  fail=1
fi
exit "$fail"
