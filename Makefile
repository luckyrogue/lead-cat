.PHONY: help setup deps up down ps migrate migrate-down migrate-status \
	backend backend-watch miniapp admin landing frontend dev lint fmt fmt-check typecheck openapi-generate brand-sync build docker-build clean \
	test ci

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

lint:
	@command -v golangci-lint >/dev/null || (echo "install: brew install golangci-lint" && exit 1)
	@cd $(BACKEND) && golangci-lint run --config $(ROOT)config/.golangci.yml ./...

test:
	@cd $(BACKEND) && $(GO) test ./...

fmt:
	@$(PNPM) --filter mini-app run format
	@$(PNPM) --filter admin run format
	@$(PNPM) --filter landing run format
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt --config $(ROOT)config/.golangci.yml) || true

fmt-check:
	@$(PNPM) --filter mini-app run format:check
	@$(PNPM) --filter admin run format:check
	@$(PNPM) --filter landing run format:check
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt --diff --config $(ROOT)config/.golangci.yml) || true

typecheck:
	@$(PNPM) --filter mini-app typecheck
	@$(PNPM) --filter admin typecheck
	@$(PNPM) --filter landing typecheck

build:
	@cd $(BACKEND) && $(GO) build -o bin/lead-cat ./cmd/server && $(GO) build -o bin/migrate ./cmd/migrate
	@$(MAKE) openapi-generate
	@$(PNPM) --filter mini-app build
	@$(PNPM) --filter admin build
	@$(PNPM) --filter landing build

ci: fmt-check lint test typecheck build
	@echo "ci: all gates passed"

docker-build:
	docker build -t lead-cat:local -f deploy/Dockerfile --build-arg VITE_AUTH_DEV_MODE=false .

clean:
	rm -rf $(BACKEND)/bin $(MINIAPP)/dist $(ADMIN)/dist $(LANDING)/dist
