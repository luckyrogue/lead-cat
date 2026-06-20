#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-e2e"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
$compose up -d --build
ready=0
for i in $(seq 1 60); do
  if curl -fsS http://localhost:8090/ >/dev/null 2>&1 && curl -fsS http://localhost:8090/api/health >/dev/null 2>&1; then ready=1; break; fi
  sleep 3
done
[ "${ready}" = 1 ] || { echo "stack did not become ready"; $compose logs --tail=50; exit 1; }
( cd e2e && pnpm exec playwright test "$@" )
