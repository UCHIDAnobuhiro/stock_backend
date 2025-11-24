# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Local Development
```bash
# Start API server (with hot reload via Air)
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock up backend-dev

# Run batch data ingestion (fetches stock data from external API)
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock run --rm --no-deps ingest

# View logs
docker logs -f stock-backend-dev
```

### Testing & Linting
```bash
# Run all tests with race detection and coverage
go test ./... -v -race -cover

# Run tests for a specific package
go test ./internal/feature/candles/usecase/... -v

# Run a specific test function
go test ./internal/feature/auth/usecase/... -v -run TestAuthUsecase_Login

# Run linter (uses golangci-lint with depguard rules)
golangci-lint run --timeout=5m

# Build all packages
go build ./...
```

### Environment Setup
- Copy `example.env.docker` to `.docker.env` and configure:
  - `TWELVE_DATA_API_KEY`: Get from https://twelvedata.com/ (free tier: 8 req/min)
  - `JWT_SECRET`: Set a strong secret for production
  - DB and Redis configurations for local development

## Architecture Overview

This is a **Go/Gin REST API** using **feature-based Clean Architecture** with vertical slices.

### Directory Structure

```
internal/
├── app/
│   ├── di/           # Dependency injection factories
│   └── router/       # Main HTTP router configuration
├── feature/          # Feature modules (vertical slices)
│   ├── auth/
│   ├── candles/
│   └── symbollist/
├── platform/         # Infrastructure layer (renamed from "infrastructure")
│   ├── cache/        # Redis caching decorators
│   ├── db/           # Database initialization
│   ├── externalapi/  # External API clients (TwelveData)
│   ├── http/         # HTTP client configuration
│   ├── jwt/          # JWT generation and middleware
│   ├── mysql/        # (if present)
│   └── redis/        # Redis client setup
└── shared/           # Shared utilities (e.g., rate limiter)
```

### Feature Module Structure

Each feature follows a consistent layered structure:

```
feature/<name>/
├── domain/
│   ├── entity/       # Domain models (e.g., Candle, Symbol, User)
│   └── repository/   # Repository interfaces (no implementation)
├── usecase/          # Application logic (orchestrates repositories)
├── adapters/         # Repository implementations (MySQL, etc.)
└── transport/
    ├── handler/      # HTTP handlers (Gin)
    └── http/dto/     # Request/response DTOs
```

### Dependency Rules (Enforced by golangci-lint depguard)

**domain/** and **usecase/** layers MUST NOT import:
- `adapters/` (repository implementations)
- `transport/` (HTTP handlers, DTOs)

This ensures domain logic remains independent of infrastructure details.

### Key Architectural Patterns

1. **Repository Pattern**: All data access goes through repository interfaces defined in `domain/repository/`
2. **Decorator Pattern for Caching**: `platform/cache/CachingCandleRepository` wraps the base repository
   - Implements the same `CandleRepository` interface
   - Transparently adds Redis caching without changing usecase code
   - Gracefully degrades if Redis is unavailable (logs warning and runs without cache)
3. **Dependency Injection**: Manual DI in `cmd/server/main.go` and `internal/app/di/`
   - Repositories → Usecases → Handlers wired in main.go
   - `internal/app/di/` contains factory functions for complex dependencies
4. **Two Entry Points**:
   - `cmd/server/main.go`: REST API server (port 8080)
   - `cmd/ingest/main.go`: Batch job to fetch stock data from TwelveData API

### Data Flow Examples

#### Stock Price Request Flow
1. Client requests `/candles/:code?interval=1day&outputsize=200` with JWT auth
2. Router (`app/router`) validates JWT via `jwtmw.AuthRequired()` middleware
3. Routes to `candles/transport/handler/CandlesHandler.GetCandlesHandler`
4. Handler parses params (defaults: interval=1day, outputsize=200) and calls usecase
5. Usecase calls `CandleRepository.Find(ctx, symbol, interval, outputsize)`
6. `CachingCandleRepository` checks Redis with key format: `candles:{symbol}:{interval}:{outputsize}`
   - **Cache HIT**: Returns deserialized JSON from Redis
   - **Cache MISS**: Calls `candlesadapters.CandleRepository` (MySQL) → caches result with TTL → returns data
7. Handler transforms domain entities to DTOs and returns JSON

#### Batch Ingestion Flow
1. `cmd/ingest/main.go` starts with 5-minute context timeout
2. Loads active symbols from `symbollistadapters.SymbolRepository`
3. For each symbol × each interval (1day, 1week, 1month):
   - `RateLimiter.WaitIfNeeded()` enforces 8 req/min limit
   - Calls TwelveData API via `MarketRepository.GetTimeSeries()`
   - Upserts candles to MySQL via `CandleRepository.UpsertBatch()`
   - Cache invalidation: Deletes Redis keys matching `candles:{symbol}:{interval}:*`
4. Errors are logged but don't stop processing (continues to next symbol/interval)

### External Dependencies & Rate Limiting

- **TwelveData API**: Stock market data provider (rate limited: 8 req/min on free tier)
  - Rate limiting is handled by `shared/ratelimiter` package
  - Ingest batch fetches 200 data points per request for 3 intervals (1day, 1week, 1month)
  - Rate limiter automatically sleeps when limit is reached
- **MySQL/Cloud SQL**: Primary data store (GORM ORM)
- **Redis**: Caching layer with dynamic TTL
  - Cache TTL is set to next 8 AM Japan time (market open) using `cache.TimeUntilNext8AM()`
  - Cache keys include symbol, interval, and outputsize
  - Cache invalidation happens on `UpsertBatch` operations

### Authentication

- JWT-based authentication using `Authorization: Bearer <token>` header
- Middleware: `platform/jwt/AuthRequired()`
- Public endpoints: `/healthz`, `/signup`, `/login`
- Protected endpoints: `/candles/:code`, `/symbols`

### Testing Notes

- Test files follow `*_test.go` naming convention
- Tests exist at handler and usecase layers
- Use table-driven tests and mocks for repository interfaces

## Adding New Features

When adding a new feature, follow the established pattern:

1. **Create feature directory** under `internal/feature/<feature-name>/`
2. **Define domain layer first**:
   - `domain/entity/` - Create domain models (pure Go structs)
   - `domain/repository/` - Define repository interfaces (no implementations)
3. **Implement usecase layer**: `usecase/` - Business logic that orchestrates repositories
4. **Implement adapters**: `adapters/` - Repository implementations (MySQL, etc.)
5. **Add transport layer**:
   - `transport/handler/` - HTTP handlers
   - `transport/http/dto/` - Request/response DTOs
6. **Wire dependencies** in `cmd/server/main.go` or `cmd/ingest/main.go`
7. **Register routes** in `internal/app/router/router.go`

**Important**: Respect the dependency rules - domain/usecase layers cannot import adapters or transport layers. This is enforced by golangci-lint depguard.

## Creating Pull Requests

When creating pull requests, **ALWAYS follow the template in** [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md).

The PR description must include these sections:

1. **Description**: Brief explanation of what the PR does
2. **Changes**: Bulleted list of main changes
3. **Testing**: How the changes were tested
   - Check applicable items: Unit tests, Integration tests, Manual testing
4. **Review Points**: What reviewers should focus on

Use the template format to ensure consistency and completeness in all pull request descriptions.
