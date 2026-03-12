# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A multiplayer word-guessing game server built with Go, Fiber, Redis, and PostgreSQL. Features WebSocket support for real-time multiplayer gameplay.

## Tech Stack

- **Framework**: Fiber v2 (HTTP) + Fiber WebSocket
- **Databases**:
  - Redis (volatile data: rooms, users, games with TTL)
  - PostgreSQL (persistent data: word lists)
- **Language**: Go 1.25.5
- **Testing**: testify (assertions), gotestsum (test runner)
- **Development**: air (hot reload), gofumpt (formatting), golangci-lint v2.7.2 (linting)

## Architecture

**Layered architecture** with clear separation of concerns:

```
cmd/server/main.go              # Entry point
internal/
├── config/                     # Configuration management
├── database/                   # Database connection setup
│   ├── redis.go
│   └── postgres.go
├── core/                       # Business logic (layered per entity)
│   ├── room/                   # Room entity
│   │   ├── model/              # Data structures
│   │   ├── repository/         # Data access layer (interface-based)
│   │   ├── service/            # Business logic layer
│   │   ├── controller/         # HTTP handlers
│   │   └── routes/             # Route registration
│   ├── game/                   # Game logic (same structure)
│   ├── user/                   # User entity (same structure)
│   └── word/                   # Word entity (same structure)
├── middleware/                 # HTTP middleware
└── websocket/                  # WebSocket handling
    └── controller/
```

**Design patterns**:
- Interface-based repositories and services for testability
- Dependency injection through constructor functions
- Entity-driven organization (each entity has model → repository → service → controller hierarchy)

## Common Commands

### Development Setup
```bash
make install-tools              # Install Go tooling (gofumpt, golangci-lint, air, gotestsum)
make db-up                      # Start PostgreSQL and Redis in Docker
make db-down                    # Stop databases
make db-remove                  # Remove databases and volumes
```

### Development
```bash
make dev                        # Start server with hot-reload (requires databases running)
make code-check                 # Format and lint code
```

### Testing
```bash
make test                       # Run all tests with gotestsum
go test ./internal/core/room/service -v     # Run specific package tests
go test -run TestName ./...     # Run tests matching pattern
```

### Docker
```bash
make docker-up                  # Start all services (requires .env file)
make docker-down                # Stop all services
make docker-build               # Build API Docker image
```

### Database Management
```bash
make db-wait                    # Wait for databases to be ready (used in CI/scripts)
```

## Environment Setup

1. Copy environment template:
```bash
cp .env.example .env
```

2. Environment variables control database connections. Defaults in `.env.example` work with docker-compose setup.

## Key Design Decisions

- **Redis for volatile data**: Rooms, users, and games are temporary; TTLs prevent stale data
- **PostgreSQL for persistent data**: Word lists are immutable application data
- **Interface-based repositories**: All data access goes through interfaces (`RoomRepository`, `GameRepository`, etc.) enabling mock-based testing
- **Entity-driven structure**: New features follow the established model → repository → service → controller pattern

## Testing Patterns

- Unit tests: Mock repositories, test business logic in isolation
- Integration tests: Use real databases (see `*_integration_test.go` files)
- Assertions: Use `testify/assert` package for readable test failures
- Test discovery: Tests use standard Go naming (`*_test.go`)

## Linting & Formatting

- **Formatter**: gofumpt (enforces idiomatic Go formatting)
- **Linter**: golangci-lint v2.7.2 (see `.golangci.yml` for enabled rules)
- Run `make code-check` before committing to auto-fix issues

## Hot Reload Development

The `make dev` command uses Air for instant reloads on file changes. Air is configured in `.air.toml` to:
- Exclude test files and vendor directory from watch
- Rebuild binary in `./tmp/main`
- Set `ENV_FILE=.env` when running

## WebSocket Implementation

Real-time game updates use Fiber's WebSocket contrib. See `WEBSOCKET.md` for protocol details and `internal/websocket/` for handler implementation.

## Database Migrations

Migrations are in the `migrations/` directory. PostgreSQL schema is initialized through these migration files.
