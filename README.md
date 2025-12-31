o Game Server

A multiplayer word-guessing game server built with Go, Fiber, Redis, and PostgreSQL.

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
    │   ├── model/
    │   ├── repository/
    │   ├── service/
    │   ├── controller/
    │   └── routes/
    ├── word/                   # Word entity (stored in PostgreSQL)
    │   ├── model/
    │   ├── repository/
    │   ├── service/
    │   ├── controller/
    │   └── routes/
    └── game/                   # Game logic
        ├── model/
        ├── repository/
        ├── service/
        ├── controller/
        └── routes/
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

## Docker Commands

```bash
make db-up      # Start PostgreSQL and Redis
make db-down    # Stop databases
make db-logs    # View database logs
make db-clean   # Stop and remove all data
make run        # Run the application
```

## API Endpoints

### Rooms
- `POST /api/v1/rooms` - Create room
- `GET /api/v1/rooms` - Get all rooms
- `GET /api/v1/rooms/:id` - Get room by ID
- `DELETE /api/v1/rooms/:id` - Delete room

### Users
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/:id` - Get user by ID
- `GET /api/v1/users/room/:roomId` - Get users in room
- `DELETE /api/v1/users/:id` - Delete user

### Words
- `POST /api/v1/words` - Create word
- `GET /api/v1/words` - Get all words
- `GET /api/v1/words/:id` - Get word by ID
- `GET /api/v1/words/category/:category` - Get words by category
- `GET /api/v1/words/category/:category/random?limit=10` - Get random words
- `DELETE /api/v1/words/:id` - Delete word

### Games
- `POST /api/v1/games/start` - Start new game
- `GET /api/v1/games/:id` - Get game by ID
- `GET /api/v1/games/room/:roomId` - Get game by room
- `POST /api/v1/games/guess` - Submit guess
- `POST /api/v1/games/:id/next-round` - Advance to next round
- `POST /api/v1/games/:id/end` - End game

## Data Flow

1. **Room Creation**: Rooms are stored in Redis with 24-hour TTL
2. **User Join**: Users join rooms and are stored in Redis, linked to their room
3. **Game Start**: Game pulls random words from PostgreSQL and manages state in Redis
4. **Gameplay**: Game logic validates guesses, updates scores, and manages rounds
5. **Cleanup**: Redis handles automatic cleanup via TTL for temporary data

## Key Design Decisions

- **Redis for volatile data**: Rooms, users, and games are temporary and stored in Redis with TTL
- **PostgreSQL for persistent data**: Words are permanent and stored in PostgreSQL
- **Layered architecture**: Clear separation between model, repository, service, and controller
- **Interface-based design**: All repositories and services use interfaces for testability
