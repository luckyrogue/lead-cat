#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

STAGED="$(git diff --cached --name-only --diff-filter=ACM)"
if [[ -z "$STAGED" ]]; then
  exit 0
fi

format_go() {
  local rel_files=()
  while IFS= read -r file; do
    [[ -n "$file" ]] && rel_files+=("${file#apps/backend/}")
  done < <(printf '%s\n' "$STAGED" | grep '^apps/backend/.*\.go$' || true)

  if ((${#rel_files[@]} == 0)); then
    return 0
  fi

  if command -v golangci-lint >/dev/null 2>&1; then
    (cd apps/backend && golangci-lint fmt "${rel_files[@]}")
  else
    for rel in "${rel_files[@]}"; do
      gofmt -w "apps/backend/${rel}"
    done
  fi

  printf '%s\n' "$STAGED" | grep '^apps/backend/.*\.go$' | xargs git add
}

format_app_prettier() {
  local app="$1"
  local files=()
  while IFS= read -r file; do
    [[ -n "$file" ]] && files+=("$file")
  done < <(printf '%s\n' "$STAGED" | grep "^apps/${app}/.*\.\(ts\|tsx\|mjs\)$" || true)

  if ((${#files[@]} == 0)); then
    return 0
  fi

  pnpm exec prettier --write "${files[@]}" --config "apps/${app}/config/prettier.config.mjs"
  git add "${files[@]}"
}

format_go
for app in landing admin mini-app; do
  format_app_prettier "$app"
done
