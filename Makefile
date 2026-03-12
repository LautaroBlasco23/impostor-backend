.PHONY: help install-tools code-check dev docker-up docker-down docker-build db-up db-down db-remove db-wait test start stop
.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "  🚀 Quick Start:"
	@echo "    start              - Start environment (choose local or docker)"
	@echo "    stop               - Stop environment (choose local or docker)"
	@echo ""
	@echo "  🛠️  Development:"
	@echo "    install-tools      - Install Go tools (gofumpt, golangci-lint, air, gotestsum)"
	@echo "    code-check         - Format and lint code"
	@echo "    dev                - Start application with databases"
	@echo "    test               - Run tests"
	@echo ""
	@echo "  🐳 Docker:"
	@echo "    docker-up          - Start all services"
	@echo "    docker-down        - Stop services"
	@echo "    docker-build       - Build API image"
	@echo ""
	@echo "  🗄️  Database:"
	@echo "    db-up              - Start databases"
	@echo "    db-down            - Stop databases"
	@echo "    db-remove          - Remove databases and volumes"
	@echo "    db-wait            - Wait for databases to be ready"

install-tools:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
	go install github.com/air-verse/air@latest
	go install gotest.tools/gotestsum@latest

code-check:
	gofumpt -l -w .
	golangci-lint run --fix ./...

db-wait:
	@echo "Waiting for databases to be ready..."
	@until docker compose -f docker-compose.db.yml exec -T postgres pg_isready -U $${POSTGRES_USER:-postgres} > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "PostgreSQL is ready"

dev: db-up db-wait
	ENV_FILE=.env air -c .air.toml

docker-up:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose --env-file .env up -d

docker-down:
	docker compose down

docker-build:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose build api

db-up:
	docker compose -f docker-compose.db.yml up -d

db-down:
	docker compose -f docker-compose.db.yml stop

db-remove:
	docker compose -f docker-compose.db.yml down -v

test:
	gotestsum --format=short-verbose

start:
	@bash scripts/start.sh

stop:
	@bash scripts/stop.sh
