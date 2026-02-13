# Game Server

A multiplayer word-guessing game server built with Go, Fiber, Redis, and PostgreSQL.

This project includes a `Makefile` that provides convenient `make` commands to simplify common development tasks.

## Architecture

```
cmd/server/main.go              # Application entry point
internal/
├── config/                     # Configuration management
├── database/                   # Database connections
│   ├── redis.go
│   └── postgres.go
└── core/                       # Business logic
    ├── room/                   # Room entity (stored in Redis)
    │   ├── model/
    │   ├── repository/
    │   ├── service/
    │   ├── controller/
    │   └── routes/
    ├── user/                   # User entity (stored in Redis)
    ├── word/                   # Word entity (stored in PostgreSQL)
    └── game/                   # Game logic
```

## Prerequisites

- Go 1.23+
- Docker & Docker Compose (for local databases)

## Quick Start

1. Install dependencies:
```bash
go mod download
```

2. Start databases with Docker:
```bash
make db-up
```

Or manually:
```bash
docker-compose up -d
```

3. Set up environment variables (optional, defaults work with docker-compose):
```bash
cp .env.example .env
```

4. Run the server:
```bash
make run
```

Or manually:
```bash
go run cmd/server/main.go
```

## Key Design Decisions

- **Redis for volatile data**: Rooms, users, and games are temporary and stored in Redis with TTL
- **PostgreSQL for persistent data**: Words are permanent and stored in PostgreSQL
- **Layered architecture**: Clear separation between model, repository, service, and controller
- **Interface-based design**: All repositories and services use interfaces for testability
