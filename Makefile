.PHONY: help dev dev-backend dev-frontend build build-backend build-frontend \
        up down logs migrate-up migrate-down migrate-create seed \
        test test-backend test-frontend lint clean

# Default target
.DEFAULT_GOAL := help

# ────────────────────────────────────────
# Variables
# ────────────────────────────────────────
COMPOSE         := docker compose
COMPOSE_DEV     := $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
MIGRATE         := migrate
MIGRATE_DB_URL  ?= postgres://verdox:verdoxpass@localhost:5432/verdox?sslmode=disable
MIGRATION_DIR   := backend/migrations

# ============================================================
# Help
# ============================================================

help: ## Show this help message
	@echo "Verdox Makefile Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ============================================================
# Development
# ============================================================

dev: ## Start full stack with hot reload (docker-compose.dev.yml)
	@echo "Starting postgres and redis..."
	@$(COMPOSE_DEV) up -d postgres redis
	@echo "Waiting for PostgreSQL to be ready..."
	@until $(COMPOSE_DEV) exec postgres pg_isready -U verdox -d verdox > /dev/null 2>&1; do sleep 1; done
	@echo "Running migrations..."
	@$(MIGRATE) -path $(MIGRATION_DIR) -database "$(MIGRATE_DB_URL)" up || true
	@echo "Starting all services..."
	$(COMPOSE_DEV) up -d --build

dev-backend: ## Start backend only with air (Go hot reload)
	cd backend && go install github.com/air-verse/air@latest && air -c .air.toml

dev-frontend: ## Start frontend only with Next.js dev server
	cd frontend && npm run dev

# ============================================================
# Build
# ============================================================

build: build-backend build-frontend ## Build all Docker images for production

build-backend: ## Build backend Docker image
	$(COMPOSE) build backend

build-frontend: ## Build frontend Docker image
	$(COMPOSE) build frontend

# ============================================================
# Docker Compose
# ============================================================

up: ## Start the production stack (detached)
	$(COMPOSE) up -d --build

down: ## Stop and remove all containers and volumes
	$(COMPOSE) down -v
# 	@docker volume prune -f > /dev/null 2>&1 || true

logs: ## Tail logs for all services (Ctrl+C to stop)
	$(COMPOSE) logs -f

# ============================================================
# Database
# ============================================================

migrate-up: ## Run all pending database migrations
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(MIGRATE_DB_URL)" up

migrate-down: ## Rollback the last applied migration
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(MIGRATE_DB_URL)" down 1

migrate-create: ## Create a new migration pair (usage: make migrate-create NAME=add_index)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=add_index"; \
		exit 1; \
	fi
	$(MIGRATE) create -ext sql -dir $(MIGRATION_DIR) -seq $(NAME)

seed: ## Bootstrap root user from ROOT_EMAIL and ROOT_PASSWORD env vars
	cd backend && go run ./scripts/seed/main.go

# ============================================================
# Testing
# ============================================================

test: test-backend test-frontend ## Run all tests (backend + frontend)

test-backend: ## Run Go tests with race detector
	cd backend && go test -race -count=1 -timeout 120s ./...

test-frontend: ## Run frontend tests (Jest / Vitest)
	cd frontend && npm test -- --watchAll=false

# ============================================================
# Linting
# ============================================================

lint: ## Run all linters (golangci-lint + ESLint)
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint

# ============================================================
# Cleanup
# ============================================================

clean: ## Remove all containers, volumes, and build artifacts
	$(COMPOSE) down -v --remove-orphans
	docker image prune -f --filter "label=project=verdox"
	rm -rf backend/tmp frontend/.next frontend/node_modules
