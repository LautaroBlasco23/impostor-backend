.PHONY: help start stop install-tools code-check dev test full-docker-up full-docker-down local-docker-up local-docker-down docker-build docker-clean db-down
.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "  Main Commands:"
	@echo "    start                - Interactive environment selector (start)"
	@echo "    stop                 - Interactive environment selector (stop)"
	@echo ""
	@echo "  ── Local Dev  (native Go app + databases in Docker) ────────────────"
	@echo "    dev                  - Start app with hot-reload (requires DBs running)"
	@echo "    db-down              - Stop databases"
	@echo "    code-check           - Format and lint code"
	@echo "    test                 - Run tests"
	@echo "    install-tools        - Install Go tools (gofumpt, golangci-lint, air, gotestsum)"
	@echo ""
	@echo "  ── Local Docker  (full stack, no nginx, localhost ports) ───────────"
	@echo "    local-docker-up      - Start full stack without nginx"
	@echo "    local-docker-down    - Stop local full stack"
	@echo ""
	@echo "  ── Docker Production  (nginx + api + databases) ────────────────────"
	@echo "    full-docker-up       - Start all services"
	@echo "    full-docker-down     - Stop all services"
	@echo "    docker-build         - Build API image"
	@echo "    docker-clean         - Remove all containers and volumes"
	@echo ""

start:
	@bash scripts/start.sh

stop:
	@bash scripts/stop.sh

install-tools:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
	go install github.com/air-verse/air@latest
	go install gotest.tools/gotestsum@latest

code-check:
	gofumpt -l -w .
	golangci-lint run --fix ./...

dev:
	@until docker compose -f docker-compose.db.yml exec -T postgres pg_isready -U $${POSTGRES_USER:-postgres} > /dev/null 2>&1; do \
		sleep 1; \
	done
	ENV_FILE=.env air -c .air.toml

test:
	gotestsum --format=short-verbose

full-docker-up:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose --env-file .env up -d $(BUILD_FLAG)

full-docker-down:
	docker compose down

db-down:
	docker compose -f docker-compose.db.yml down

local-docker-up:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose -f docker-compose.local.yml --env-file .env up -d $(BUILD_FLAG)

local-docker-down:
	docker compose -f docker-compose.local.yml down

docker-build:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose build api

docker-clean:
	docker compose down -v
	docker compose -f docker-compose.local.yml down -v
	docker compose -f docker-compose.db.yml down -v
