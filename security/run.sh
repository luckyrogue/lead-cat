#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-dast"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
$compose up -d --build
for i in $(seq 1 60); do
  curl -fsS http://localhost:8090/api/health >/dev/null 2>&1 && break
  sleep 3
  [ "$i" = 60 ] && { echo "stack not healthy"; $compose logs --tail=50; exit 1; }
done

mkdir -p security/report
net="--network host"
host="localhost"
if [ "$(uname)" = "Darwin" ]; then net=""; host="host.docker.internal"; fi

status=0
for app in "admin:8090" "landing:8091" "mini-app:8092"; do
  name="${app%%:*}"; port="${app##*:}"
  echo "[dast] scanning $name (http://$host:$port)"
  docker run --rm $net -v "$PWD/security:/zap/wrk:rw" zaproxy/zap-stable \
    zap-baseline.py -t "http://$host:$port" \
    -c zap-baseline.conf -r "report/zap-$name.html" -w "report/zap-$name.md" \
    || status=$?   # ZAP exits non-zero on WARN/FAIL; report, don't gate
done
echo "[dast] done (combined ZAP exit signal: $status). Reports in security/report/."
