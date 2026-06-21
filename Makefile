.PHONY: help setup deps up down ps migrate migrate-down migrate-status \
	backend backend-watch miniapp admin landing frontend dev lint fmt fmt-check hooks typecheck openapi-generate brand-sync build clean \
	test ci ci-full govulncheck dast

ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BACKEND := $(ROOT)apps/backend
MINIAPP := $(ROOT)apps/mini-app
ADMIN := $(ROOT)apps/admin
LANDING := $(ROOT)apps/landing
COMPOSE := docker compose -f deploy/docker-compose.yml
GO := env -u GOROOT go
PNPM := pnpm
.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z0-9_.-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

setup:
	@test -f .env || cp deploy/.env.example .env
	@echo "Edit .env (BOT_TOKEN, MASTER_ENCRYPTION_KEY) then: make migrate && make dev"
	@$(MAKE) up
	@$(PNPM) install
	@$(MAKE) brand-sync
	@$(MAKE) openapi-generate
	@$(MAKE) hooks

deps: up
	@$(PNPM) install
	@$(MAKE) brand-sync
	@$(MAKE) openapi-generate

openapi-generate:
	@$(PNPM) openapi:generate

brand-sync:
	@cd packages/brand && $(PNPM) generate && $(PNPM) sync

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

ps:
	$(COMPOSE) ps

migrate:
	@set -a && . ./.env && set +a && cd $(BACKEND) && $(GO) run ./cmd/migrate up

migrate-down:
	@set -a && . ./.env && set +a && cd $(BACKEND) && $(GO) run ./cmd/migrate down

migrate-status:
	@set -a && . ./.env && set +a && cd $(BACKEND) && $(GO) run ./cmd/migrate status

backend:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(BACKEND) && $(GO) run ./cmd/server

backend-watch:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && \
		GOBIN=$$($(GO) env GOPATH)/bin && \
		test -x "$$GOBIN/air" || (echo "install: env -u GOROOT go install github.com/air-verse/air@latest" && exit 1) && \
		cd $(BACKEND) && env -u GOROOT PATH="$$GOBIN:$$PATH" air

frontend:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(MINIAPP) && $(PNPM) dev

miniapp:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(MINIAPP) && $(PNPM) dev

admin:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(ADMIN) && $(PNPM) dev

landing:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(LANDING) && $(PNPM) dev

dev:
	@$(MAKE) -j2 backend-watch frontend

lint: ## golangci-lint + eslint (apps)
	@command -v golangci-lint >/dev/null || (echo "install: brew install golangci-lint" && exit 1)
	@cd $(BACKEND) && golangci-lint run ./...
	@$(PNPM) turbo run lint --filter=./apps/*

test: ## go test -race + vitest (apps + ui)
	@cd $(BACKEND) && $(GO) test -race ./...
	@$(PNPM) turbo run test --filter=./apps/* --filter=@leadcat/ui

fmt: ## format frontend + go (golangci fmt)
	@$(PNPM) turbo run format --filter=./apps/*
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt) || true

fmt-check: ## check formatting (prettier + golangci fmt diff)
	@$(PNPM) turbo run format:check --filter=./apps/*
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt --diff) || (echo "install: brew install golangci-lint" && exit 1)

hooks: ## install git pre-commit hook (auto-format staged files)
	@chmod +x scripts/format-staged.sh scripts/install-git-hooks.sh .githooks/pre-commit
	@bash scripts/install-git-hooks.sh

typecheck: ## tsc + react-router typegen (apps)
	@$(PNPM) turbo run typecheck --filter=./apps/*

build: ## go binaries + openapi + frontend production build
	@cd $(BACKEND) && $(GO) build -o bin/lead-cat ./cmd/server && $(GO) build -o bin/migrate ./cmd/migrate
	@$(MAKE) openapi-generate
	@$(PNPM) turbo run build --filter=./apps/*

ci: fmt-check lint test typecheck build ## local CI gate (core _build.yml checks)
	@cd $(BACKEND) && $(GO) vet ./...
	@echo "ci: all gates passed"

ci-full: ci ## full gate incl. OpenAPI drift + govulncheck + e2e
	@$(PNPM) openapi:generate
	@git diff --exit-code -- apps/backend/openapi/openapi.json packages/api-client/src/generated/schema.ts
	@cd $(BACKEND) && go install golang.org/x/vuln/cmd/govulncheck@v1.1.4 && govulncheck ./...
	@bash e2e/run.sh

load: ## run the k6 load harness (capacity + shedding) — on-demand, not a CI gate
	bash load/run.sh all

dast: ## run the OWASP ZAP baseline scan over the app stack
	bash security/run.sh

govulncheck:
	@cd $(BACKEND) && go install golang.org/x/vuln/cmd/govulncheck@v1.1.4 && govulncheck ./...

clean:
	rm -rf $(BACKEND)/bin $(MINIAPP)/build $(MINIAPP)/dist $(ADMIN)/build $(ADMIN)/dist $(LANDING)/build $(LANDING)/dist .turbo
