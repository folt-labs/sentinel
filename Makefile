.PHONY: all build clean dev test help

# Variables
AGENT_BINARY = sentinel-agent
API_BINARY = sentinel-api
SCANNER_BINARY = sentinel-scanner

# Colors
GREEN = \033[0;32m
NC = \033[0m

help: ## Show this help
	@echo "Sentinel Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# Infrastructure
dev-up: ## Start development infrastructure (Postgres, Redis, Mailhog)
	docker compose -f deploy/docker-compose.dev.yml up -d

dev-down: ## Stop development infrastructure
	docker compose -f deploy/docker-compose.dev.yml down

dev-logs: ## View development infrastructure logs
	docker compose -f deploy/docker-compose.dev.yml logs -f

dev-clean: ## Stop and remove all development data
	docker compose -f deploy/docker-compose.dev.yml down -v

# Database
db-migrate: ## Run database migrations
	cd api && go run cmd/migrate/main.go

db-shell: ## Open PostgreSQL shell
	docker exec -it sentinel-postgres psql -U sentinel -d sentinel

db-reset: ## Reset database (drop and recreate)
	docker exec -it sentinel-postgres psql -U sentinel -c "DROP DATABASE IF EXISTS sentinel; CREATE DATABASE sentinel;"
	$(MAKE) db-migrate

# API
api-dev: ## Run API in development mode
	cd api && go run cmd/sentinel-api/main.go

api-build: ## Build API binary
	cd api && go build -o bin/$(API_BINARY) ./cmd/sentinel-api

api-test: ## Run API tests
	cd api && go test -v ./...

# Agent
agent-dev: ## Run agent in development mode (Linux only)
	cd agent && go run cmd/sentinel-agent/main.go

agent-build: ## Build agent binary for Linux
	cd agent && GOOS=linux GOARCH=amd64 go build -o bin/$(AGENT_BINARY) ./cmd/sentinel-agent

agent-build-windows: ## Build agent binary for Windows
	cd agent && go build -o bin/$(AGENT_BINARY).exe ./cmd/sentinel-agent

agent-test: ## Run agent tests
	cd agent && go test -v ./...

# Scanner
scanner-build: ## Build scanner binary
	cd scanner && go build -o bin/$(SCANNER_BINARY) ./cmd/sentinel-scanner

# Dashboard
dashboard-dev: ## Run dashboard in development mode
	cd dashboard && npm run dev

dashboard-build: ## Build dashboard for production
	cd dashboard && npm run build

dashboard-install: ## Install dashboard dependencies
	cd dashboard && npm install

# All builds
build-all: api-build agent-build scanner-build dashboard-build ## Build all components

# Testing
test-all: api-test agent-test ## Run all tests

# Development workflow
dev: dev-up ## Start full development environment
	@echo "Starting development environment..."
	@echo "Postgres: localhost:5432"
	@echo "Redis: localhost:6379"
	@echo "Mailhog UI: http://localhost:8025"
	@echo ""
	@echo "Run these commands in separate terminals:"
	@echo "  make api-dev       # Start API server"
	@echo "  make dashboard-dev # Start dashboard"

# Dependencies
deps: ## Download all dependencies
	cd agent && go mod download
	cd api && go mod download
	cd scanner && go mod download
	cd dashboard && npm install

# Go module management
tidy: ## Tidy all Go modules
	cd agent && go mod tidy
	cd api && go mod tidy
	cd scanner && go mod tidy

# Linting
lint: ## Run linters
	cd agent && golangci-lint run
	cd api && golangci-lint run
	cd scanner && golangci-lint run
	cd dashboard && npm run lint

# Clean
clean: ## Clean build artifacts
	rm -rf agent/bin api/bin scanner/bin
	rm -rf dashboard/.next dashboard/out

# Docker
docker-build: ## Build Docker images
	docker build -t sentinel-api:latest -f api/Dockerfile api/
	docker build -t sentinel-dashboard:latest -f dashboard/Dockerfile dashboard/

docker-up: ## Start full stack with Docker
	docker compose -f deploy/docker-compose.yml up -d

docker-down: ## Stop Docker stack
	docker compose -f deploy/docker-compose.yml down

# Install script
install-script: ## Generate agent install script
	@echo "#!/bin/bash"
	@echo "# Sentinel Agent Install Script"
	@echo "# Usage: curl -sSL https://your-server/install.sh | bash -s -- <API_KEY>"
	cat agent/scripts/install.sh
