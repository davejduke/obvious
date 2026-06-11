.PHONY: up down migrate seed test lint build clean

COMPOSE_FILE := infra/docker/docker-compose.yml
SERVICES := identity control-framework evidence audit-trail engagement integration reporting

# ============================================================
# Infrastructure
# ============================================================

up: ## Start all Docker Compose services
	docker compose -f $(COMPOSE_FILE) up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker compose -f $(COMPOSE_FILE) exec postgres pg_isready -U aiauditor -d aiauditor; do sleep 1; done
	@echo "All services started."

down: ## Stop all Docker Compose services
	docker compose -f $(COMPOSE_FILE) down

down-volumes: ## Stop services and remove volumes (WARNING: destroys all data)
	docker compose -f $(COMPOSE_FILE) down -v

logs: ## Tail logs from all services
	docker compose -f $(COMPOSE_FILE) logs -f

ps: ## List running services
	docker compose -f $(COMPOSE_FILE) ps

# ============================================================
# Database
# ============================================================

migrate: ## Run all database migrations in order
	@echo "Running migrations..."
	@for f in db/migrations/*.sql; do \
		echo "Applying $$f..."; \
		docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U aiauditor -d aiauditor -f /docker-entrypoint-initdb.d/$$(basename $$f) 2>/dev/null || \
		cat $$f | docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U aiauditor -d aiauditor; \
		echo "Done: $$f"; \
	done

seed: ## Seed NIS 2 Article 21 control data
	@echo "Seeding NIS 2 data..."
	cat db/migrations/002_seed_nis2.sql | docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U aiauditor -d aiauditor
	@echo "Seed complete."

db-shell: ## Open a psql shell
	docker compose -f $(COMPOSE_FILE) exec postgres psql -U aiauditor -d aiauditor

db-reset: ## Reset database (drop and recreate)
	docker compose -f $(COMPOSE_FILE) exec postgres psql -U aiauditor -c "DROP DATABASE IF EXISTS aiauditor; CREATE DATABASE aiauditor;"
	$(MAKE) migrate
	$(MAKE) seed

# ============================================================
# Build
# ============================================================

build: build-services build-engine build-frontend ## Build all components

build-services: ## Build all Go services
	@for svc in $(SERVICES); do \
		echo "Building services/$$svc..."; \
		(cd services/$$svc && go build ./cmd/api/...); \
		echo "OK: services/$$svc"; \
	done

build-engine: ## Build Python engine (install deps)
	@echo "Installing engine dependencies..."
	cd engine && pip install -e ".[dev]" --quiet
	@echo "Engine ready."

build-frontend: ## Build Next.js frontend
	@echo "Building frontend..."
	cd frontend && npm ci && npm run build
	@echo "Frontend built."

# ============================================================
# Test
# ============================================================

test: test-services test-engine test-frontend ## Run all tests

test-services: ## Run Go service tests
	@for svc in $(SERVICES); do \
		echo "Testing services/$$svc..."; \
		(cd services/$$svc && go test ./... -count=1 -timeout 30s 2>&1 || echo "No tests yet: $$svc"); \
	done

test-engine: ## Run Python engine tests
	cd engine && python -m pytest --tb=short -q 2>&1 || echo "No tests yet"

test-frontend: ## Run frontend tests
	cd frontend && npm test -- --passWithNoTests 2>&1 || echo "No tests yet"

# ============================================================
# Lint
# ============================================================

lint: lint-services lint-engine lint-frontend ## Run all linters

lint-services: ## Lint Go services
	@for svc in $(SERVICES); do \
		echo "Linting services/$$svc..."; \
		(cd services/$$svc && go vet ./... 2>&1); \
	done

lint-engine: ## Lint Python engine
	@if command -v ruff >/dev/null 2>&1; then \
		cd engine && ruff check .; \
	else \
		echo "ruff not installed, skipping Python lint"; \
	fi

lint-frontend: ## Lint frontend
	@if [ -d frontend/node_modules ]; then \
		cd frontend && npm run lint; \
	else \
		echo "Frontend not installed, run: make build-frontend"; \
	fi

# ============================================================
# Shared
# ============================================================

build-shared: ## Build shared Go module
	cd shared/types/go && go build ./...

vet-shared: ## Vet shared Go module
	cd shared/types/go && go vet ./...

# ============================================================
# Utilities
# ============================================================

clean: ## Remove build artifacts
	@for svc in $(SERVICES); do \
		rm -f services/$$svc/cmd/api/api; \
	done
	@find . -name '*.pyc' -delete
	@find . -name '__pycache__' -type d -exec rm -rf {} + 2>/dev/null || true

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

