.PHONY: help setup deps up down ps migrate migrate-down migrate-status \
	backend backend-watch frontend dev test lint fmt fmt-check typecheck build smoke docker-build clean

ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BACKEND := $(ROOT)backend
FRONTEND := $(ROOT)frontend
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
	@cd $(FRONTEND) && $(PNPM) install

deps: up
	@cd $(FRONTEND) && $(PNPM) install

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
	@set -a && . ./.env && set +a && cd $(FRONTEND) && $(PNPM) dev

dev:
	@$(MAKE) -j2 backend-watch frontend

test:
	@cd $(BACKEND) && $(GO) test -count=1 ./...

lint:
	@command -v golangci-lint >/dev/null || (echo "install: brew install golangci-lint" && exit 1)
	@cd $(BACKEND) && golangci-lint run --config $(ROOT)config/.golangci.yml ./...

fmt:
	@cd $(FRONTEND) && $(PNPM) run format
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt --config $(ROOT)config/.golangci.yml) || true

fmt-check:
	@cd $(FRONTEND) && $(PNPM) run format:check
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt --diff --config $(ROOT)config/.golangci.yml) || true

typecheck:
	@cd $(FRONTEND) && $(PNPM) typecheck

build:
	@cd $(BACKEND) && $(GO) build -o bin/lead-cat ./cmd/server && $(GO) build -o bin/migrate ./cmd/migrate
	@cd $(FRONTEND) && $(PNPM) build

smoke:
	@cd $(BACKEND) && $(GO) test -tags=smoke -count=1 ./test/smoke/...

coverage:
	@cd $(BACKEND) && $(GO) test -cover -count=1 \
		./internal/delivery/http/middleware/...

docker-build:
	docker build -t lead-cat:local -f deploy/Dockerfile --build-arg VITE_AUTH_DEV_MODE=false .

clean:
	rm -rf $(BACKEND)/bin $(FRONTEND)/dist
