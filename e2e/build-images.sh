#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
export FRONTEND_DEPS_IMAGE="${FRONTEND_DEPS_IMAGE:-leadcat-frontend-deps:local}"

require_client_builds() {
  for app in admin mini-app; do
    if [ ! -d "apps/${app}/build/client" ]; then
      echo "missing apps/${app}/build/client — run: pnpm turbo run build --filter=${app} --filter=@leadcat/brand"
      exit 1
    fi
  done
}

echo "building leadcat-frontend-deps..."
docker build \
  -f deploy/docker/Dockerfile.frontend-deps \
  -t "${FRONTEND_DEPS_IMAGE}" \
  .

echo "building leadcat-e2e-backend..."
docker build \
  -f apps/backend/Dockerfile \
  -t leadcat-e2e-backend:latest \
  .

require_client_builds

echo "building leadcat-e2e-admin..."
docker build \
  -f deploy/docker/Dockerfile.e2e-spa \
  --build-arg APP=admin \
  -t leadcat-e2e-admin:latest \
  .

echo "building leadcat-e2e-mini-app..."
docker build \
  -f deploy/docker/Dockerfile.e2e-spa \
  --build-arg APP=mini-app \
  -t leadcat-e2e-mini-app:latest \
  .

echo "building leadcat-e2e-landing..."
docker build \
  -f deploy/docker/Dockerfile.e2e-landing \
  --build-arg "FRONTEND_DEPS_IMAGE=${FRONTEND_DEPS_IMAGE}" \
  -t leadcat-e2e-landing:latest \
  .
