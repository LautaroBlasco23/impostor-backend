.PHONY: help install-tools code-check dev prod-up prod-down prod-build db-up db-down db-remove test

.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "  🛠️  Development:"
	@echo "    install-tools      - Install Go tools (gofumpt, golangci-lint, air, gotestsum)"
	@echo "    code-check         - Format and lint code"
	@echo "    dev                - Start application"
	@echo "    test               - Run tests"
	@echo ""
	@echo "  🐳 Production:"
	@echo "    prod-up            - Start all services"
	@echo "    prod-down          - Stop services"
	@echo "    prod-build         - Build API image"
	@echo ""
	@echo "  🗄️  Database:"
	@echo "    db-up              - Start databases"
	@echo "    db-down            - Stop databases"
	@echo "    db-remove          - Remove databases and volumes"

install-tools:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
	go install github.com/air-verse/air@latest
	go install gotest.tools/gotestsum@latest

code-check:
	gofumpt -l -w .
	golangci-lint run --fix ./...

dev:
	ENV_FILE=.env air -c .air.toml

prod-up:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker-compose --env-file .env up -d

prod-down:
	docker-compose down

prod-build:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker-compose build api

db-up:
	docker-compose -f docker-compose.db.yml up -d

db-down:
	docker-compose -f docker-compose.db.yml stop

db-remove:
	docker-compose -f docker-compose.db.yml down -v

test:
	gotestsum --format=short-verbose
