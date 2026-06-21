#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
export FRONTEND_DEPS_IMAGE="${FRONTEND_DEPS_IMAGE:-leadcat-frontend-deps:local}"

compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-e2e"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

if [ "${E2E_SKIP_BUILD:-}" != "1" ]; then
  bash e2e/build-images.sh
fi

$compose up -d

ready=0
for _ in $(seq 1 40); do
  if curl -fsS http://localhost:8090/ >/dev/null 2>&1 \
    && curl -fsS http://localhost:8090/api/health >/dev/null 2>&1 \
    && curl -fsS http://localhost:8091/ >/dev/null 2>&1 \
    && curl -fsS http://localhost:8092/ >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done

if [ "${ready}" != "1" ]; then
  echo "stack did not become ready"
  $compose logs --tail=50
  exit 1
fi

export E2E_MAILPIT_URL="${E2E_MAILPIT_URL:-http://localhost:8125}"

suite="${E2E_SUITE:-smoke}"
case "$suite" in
  smoke)
    playwright_grep=(--grep @smoke)
    ;;
  full)
    playwright_grep=(--grep @a11y)
    ;;
  all)
    playwright_grep=()
    ;;
  *)
    echo "unknown E2E_SUITE=${suite} (expected smoke|full|all)"
    exit 1
    ;;
esac

( cd e2e && pnpm exec playwright test "${playwright_grep[@]}" "$@" )
