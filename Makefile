.PHONY: help setup deps up down ps migrate migrate-down migrate-status \
	backend backend-watch miniapp admin landing frontend dev \
	openapi-generate brand-sync fmt clean

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
	@echo "setup deps up down ps migrate migrate-down migrate-status backend backend-watch miniapp admin landing frontend dev openapi-generate brand-sync fmt clean"

setup:
	@test -f .env || cp deploy/.env.example .env
	@echo "Edit .env (BOT_TOKEN, MASTER_ENCRYPTION_KEY) then: make migrate && make dev"
	@$(MAKE) up
	@$(PNPM) install
	@$(MAKE) brand-sync
	@$(MAKE) openapi-generate

deps:
	@$(MAKE) up
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
	@$(MAKE) frontend

admin:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(ADMIN) && $(PNPM) dev

landing:
	@test -f .env || (echo "missing .env — run: make setup" && exit 1)
	@set -a && . ./.env && set +a && cd $(LANDING) && $(PNPM) dev

dev:
	@$(MAKE) -j2 backend-watch frontend

fmt:
	@$(PNPM) turbo run format --filter=./apps/*
	@command -v golangci-lint >/dev/null && (cd $(BACKEND) && golangci-lint fmt) || true

clean:
	rm -rf $(BACKEND)/bin $(MINIAPP)/build $(MINIAPP)/dist $(ADMIN)/build $(ADMIN)/dist $(LANDING)/build $(LANDING)/dist .turbo
