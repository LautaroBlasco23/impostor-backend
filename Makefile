.PHONY: help db-up db-down db-logs db-clean run

help:
	@echo "Available commands:"
	@echo "  make db-up      - Start PostgreSQL and Redis"
	@echo "  make db-down    - Stop databases"
	@echo "  make db-logs    - View database logs"
	@echo "  make db-clean   - Stop and remove all data"
	@echo "  make run        - Run the application"

db-up:
	docker-compose up -d
	@echo "Waiting for databases to be ready..."
	@sleep 3
	@echo "Databases are running!"

db-down:
	docker-compose down

db-logs:
	docker-compose logs -f

db-clean:
	docker-compose down -v
	@echo "All data removed!"

run:
	go run cmd/server/main.go
