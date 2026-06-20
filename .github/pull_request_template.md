## Summary

<!-- What changed and why (1–3 bullets). -->

-

## Test plan

- [ ] `make ci` passes locally (fmt-check, lint, test, typecheck, build)
- [ ] Manual check described below (if UI or auth flow changed)

<!-- Steps you ran, or "N/A — docs-only". -->

## CI (GitHub)

PRs run: **go** (vet, test -race, golangci), **pnpm** (format, vitest, typecheck, lint, build, OpenAPI drift), **security** (govulncheck, pnpm audit), **e2e**, **docker validate**. Merges to `main` also run e2e before image push.

## Checklist

- [ ] Docs updated if API, auth, env, or deploy changed (`docs/*`)
- [ ] No secrets or `.env` values in the diff
- [ ] Scope matches the linked issue (no unrelated refactors)
