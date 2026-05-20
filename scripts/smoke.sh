#!/usr/bin/env bash
set -euo pipefail

BASE="${SMOKE_BASE_URL:-http://localhost:8080}"
TOKEN="${SMOKE_TOKEN:-Bearer smoke-owner}"
TOKEN_B="${SMOKE_TOKEN_B:-Bearer smoke-stranger}"
TIMEOUT="${SMOKE_RUN_TIMEOUT:-30}"

echo "==> health"
curl -fsS "$BASE/api/health" | grep -q '"postgres":"ok"'

echo "==> me"
curl -fsS -H "Authorization: $TOKEN" "$BASE/api/me" | grep -q '"id"'

echo "==> create workspace"
WS=$(curl -fsS -X POST -H "Authorization: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Smoke Lair","slug":"smoke-'$(date +%s)'"}' "$BASE/api/workspaces")
WID=$(echo "$WS" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
test -n "$WID"

echo "==> workspace get (ACL owner)"
curl -fsS -H "Authorization: $TOKEN" "$BASE/api/workspaces/$WID" | grep -q '"slug"'

echo "==> IDOR (stranger must get 403)"
code=$(curl -sS -o /dev/null -w "%{http_code}" -H "Authorization: $TOKEN_B" "$BASE/api/workspaces/$WID")
if [ "$code" != "403" ]; then
  echo "expected 403 for stranger, got $code"
  exit 1
fi

echo "==> create scenario (manual trigger only)"
DEF='{"nodes":[{"id":"t1","type":"trigger.manual","parameters":{}}],"edges":[]}'
SC=$(curl -fsS -X POST -H "Authorization: $TOKEN" -H "Content-Type: application/json" \
  -d "{\"name\":\"Smoke run\",\"definition\":$DEF}" "$BASE/api/workspaces/$WID/scenarios")
SID=$(echo "$SC" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
test -n "$SID"

echo "==> run scenario"
curl -fsS -X POST -H "Authorization: $TOKEN" "$BASE/api/workspaces/$WID/scenarios/$SID/run" | grep -q '"run_id"'

echo "==> poll runs until success (${TIMEOUT}s)"
deadline=$((SECONDS + TIMEOUT))
status=""
while [ "$SECONDS" -lt "$deadline" ]; do
  RUNS=$(curl -fsS -H "Authorization: $TOKEN" "$BASE/api/workspaces/$WID/scenarios/$SID/runs")
  status=$(echo "$RUNS" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p' | head -1)
  if [ "$status" = "success" ]; then
    break
  fi
  if [ "$status" = "failed" ]; then
    echo "run failed: $RUNS"
    exit 1
  fi
  sleep 1
done
if [ "$status" != "success" ]; then
  echo "timeout waiting for success (last status: ${status:-pending})"
  exit 1
fi

if [ "${SMOKE_SKIP_VCS:-0}" != "1" ]; then
  echo "==> integrations verify (optional)"
  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST -H "Authorization: $TOKEN" \
    "$BASE/api/workspaces/$WID/integrations/verify" || true)
  if [ "$code" != "200" ] && [ "$code" != "400" ]; then
    echo "integrations verify unexpected status $code"
    exit 1
  fi
else
  echo "==> integrations verify skipped (SMOKE_SKIP_VCS=1)"
fi

echo "==> Lead Cat smoke OK (workspace $WID, scenario $SID)"
