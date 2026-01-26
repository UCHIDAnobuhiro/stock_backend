# Candles Feature

## Overview

The Candles feature provides stock market candlestick (OHLCV) data management. It handles both real-time data retrieval via REST API and batch data ingestion from external market data providers.

### Key Features

- **Candlestick Data Retrieval**: Query OHLCV data by symbol, interval, and output size
- **Multiple Time Intervals**: Support for daily, weekly, and monthly intervals
- **Batch Data Ingestion**: Automated fetching from TwelveData API with rate limiting
- **Redis Caching**: Transparent caching layer with automatic cache invalidation
- **Upsert Operations**: Efficient batch insert/update using composite unique keys

## Sequence Diagrams

### Get Candles Flow (API Request)

```mermaid
sequenceDiagram
    participant Client
    participant Handler as CandlesHandler
    participant Usecase as CandlesUsecase
    participant Cache as CachingCandleRepository
    participant Repository as CandleMySQL
    participant Redis
    participant DB as MySQL

    Client->>Handler: GET /candles/:code?interval=1day&outputsize=200
    Handler->>Handler: Parse params (defaults: interval=1day, outputsize=200)
    Handler->>Usecase: GetCandles(symbol, interval, outputsize)
    Usecase->>Usecase: Apply defaults if needed
    Usecase->>Cache: Find(symbol, interval, outputsize)

    alt Redis Available
        Cache->>Redis: GET candles:AAPL:1day:200
        alt Cache HIT
            Redis-->>Cache: JSON data
            Cache->>Cache: Unmarshal JSON
            Cache-->>Usecase: []Candle
        else Cache MISS
            Redis-->>Cache: nil
            Cache->>Repository: Find(symbol, interval, outputsize)
            Repository->>DB: SELECT * FROM candles WHERE symbol=? AND interval=? ORDER BY time DESC LIMIT ?
            DB-->>Repository: Rows
            Repository-->>Cache: []Candle
            Cache->>Redis: SET candles:AAPL:1day:200 (TTL)
            Cache-->>Usecase: []Candle
        end
    else Redis Unavailable
        Cache->>Repository: Find(symbol, interval, outputsize)
        Repository->>DB: SELECT ...
        DB-->>Repository: Rows
        Repository-->>Cache: []Candle
        Cache-->>Usecase: []Candle
    end

    Usecase-->>Handler: []Candle
    Handler->>Handler: Convert to DTOs
    Handler-->>Client: 200 OK<br/>[{time, open, high, low, close, volume}, ...]
```

### Batch Ingestion Flow

```mermaid
sequenceDiagram
    participant Main as cmd/ingest/main.go
    participant Usecase as IngestUsecase
    participant RateLimiter
    participant Market as MarketRepository<br/>(TwelveData API)
    participant Cache as CachingCandleRepository
    participant Repository as CandleMySQL
    participant Redis
    participant DB as MySQL

    Main->>Usecase: IngestAll(symbols)

    loop For each symbol
        loop For each interval (1day, 1week, 1month)
            Usecase->>RateLimiter: WaitIfNeeded()
            RateLimiter-->>Usecase: (waits if rate limit reached)

            Usecase->>Market: GetTimeSeries(symbol, interval, 200)
            Market-->>Usecase: []Candle

            Usecase->>Usecase: Set symbol & interval on candles
            Usecase->>Cache: UpsertBatch(candles)
            Cache->>Repository: UpsertBatch(candles)
            Repository->>DB: INSERT ... ON DUPLICATE KEY UPDATE
            DB-->>Repository: Success
            Repository-->>Cache: nil

            alt Redis Available
                Cache->>Redis: SCAN & DEL candles:AAPL:1day:*
                Note over Cache,Redis: Cache invalidation
            end

            Cache-->>Usecase: nil

            alt Error Occurred
                Usecase->>Usecase: Log error, continue to next
            end
        end
    end

    Usecase-->>Main: nil
```

## API Specification

### GET /candles/:code

Retrieves candlestick data for a specified stock symbol. Requires JWT authentication.

**Path Parameters**
| Parameter | Description | Example |
|-----------|-------------|---------|
| `code` | Stock ticker symbol | `7203.T`, `AAPL` |

**Query Parameters**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `interval` | `1day` | Time interval (`1day`, `1week`, `1month`) |
| `outputsize` | `200` | Number of data points to return (max: 5000) |

**Request Example**
```
GET /candles/7203.T?interval=1day&outputsize=100
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response**

- **200 OK** - Success
  ```json
  [
    {
      "time": "2024-01-15",
      "open": 2500.0,
      "high": 2550.0,
      "low": 2480.0,
      "close": 2530.0,
      "volume": 1500000
    },
    {
      "time": "2024-01-14",
      "open": 2480.0,
      "high": 2510.0,
      "low": 2470.0,
      "close": 2500.0,
      "volume": 1200000
    }
  ]
  ```
  Note: Results are ordered by time descending (newest first).

- **401 Unauthorized** - Missing or invalid JWT token
  ```json
  {
    "error": "authorization header required"
  }
  ```

- **502 Bad Gateway** - Database or upstream error
  ```json
  {
    "error": "database connection failed"
  }
  ```

## Dependency Diagram

```mermaid
graph TB
    subgraph "Transport Layer"
        Handler[CandlesHandler<br/>transport/handler]
        DTO[CandleResponse DTO<br/>transport/http/dto]
    end

    subgraph "Usecase Layer"
        CandlesUC[CandlesUsecase<br/>usecase]
        IngestUC[IngestUsecase<br/>usecase]
        RepoInterface[CandleRepository Interface<br/>usecase/candles_usecase.go]
        MarketInterface[MarketRepository Interface<br/>usecase/ingest_usecase.go]
    end

    subgraph "Domain Layer"
        Entity[Candle Entity<br/>domain/entity]
    end

    subgraph "Adapters Layer"
        RepoImpl[CandleMySQL<br/>adapters]
    end

    subgraph "Platform Layer"
        Cache[CachingCandleRepository<br/>platform/cache]
        TwelveData[TwelveDataRepository<br/>platform/externalapi]
        RateLimiter[RateLimiter<br/>shared/ratelimiter]
    end

    subgraph "External Dependencies"
        DB[(MySQL)]
        Redis[(Redis)]
        API[TwelveData API]
    end

    Handler -->|depends on| CandlesUC
    Handler -->|uses| DTO
    CandlesUC -->|defines| RepoInterface
    CandlesUC -->|uses| Entity
    IngestUC -->|defines| MarketInterface
    IngestUC -->|uses| RepoInterface
    IngestUC -->|uses| RateLimiter
    Cache -.->|implements| RepoInterface
    Cache -->|decorates| RepoImpl
    Cache -->|uses| Redis
    RepoImpl -.->|implements| RepoInterface
    RepoImpl -->|uses| Entity
    RepoImpl -->|accesses| DB
    TwelveData -.->|implements| MarketInterface
    TwelveData -->|calls| API

    style Handler fill:#e1f5ff
    style CandlesUC fill:#fff4e1
    style IngestUC fill:#fff4e1
    style Entity fill:#e8f5e9
    style RepoInterface fill:#fff4e1
    style MarketInterface fill:#fff4e1
    style RepoImpl fill:#f3e5f5
    style Cache fill:#f3e5f5
    style TwelveData fill:#f3e5f5
    style DB fill:#ffebee
    style Redis fill:#ffebee
    style API fill:#ffebee
```

### Dependency Explanation

#### Transport Layer ([transport/handler/candle_handler.go](transport/handler/candle_handler.go))
- **CandlesHandler**: Processes HTTP requests and calls CandlesUsecase
- **CandleResponse DTO** ([transport/http/dto/candle_response.go](transport/http/dto/candle_response.go)): Response data structure with JSON serialization

#### Usecase Layer
- **CandlesUsecase** ([usecase/candles_usecase.go](usecase/candles_usecase.go)): Retrieves candlestick data with parameter validation
  - Applies default values for interval and outputsize
  - Enforces maximum outputsize limit (5000)
  - Defines `CandleRepository` interface (following Go's "consumer defines interface" convention)
- **IngestUsecase** ([usecase/ingest_usecase.go](usecase/ingest_usecase.go)): Batch data ingestion from external APIs
  - Iterates over symbols and intervals
  - Respects rate limits via RateLimiter
  - Defines `MarketRepository` interface

#### Domain Layer
- **Candle Entity** ([domain/entity/candle.go](domain/entity/candle.go)): OHLCV candlestick data model
  - `Symbol`: Stock ticker symbol (e.g., "AAPL", "7203.T")
  - `Interval`: Time interval (e.g., "1day", "1week", "1month")
  - `Time`: Timestamp for the candle period
  - `Open`, `High`, `Low`, `Close`: Price data
  - `Volume`: Trading volume

#### Adapters Layer ([adapters/candle_mysql.go](adapters/candle_mysql.go))
- **CandleMySQL**: MySQL implementation of CandleRepository (using GORM)
  - `Find`: Retrieves candles ordered by time descending
  - `UpsertBatch`: Batch insert/update using `ON DUPLICATE KEY UPDATE`
  - Composite unique index on (symbol, interval, time)

#### Platform Layer
- **CachingCandleRepository** ([platform/cache/caching_candle_repository.go](../../platform/cache/caching_candle_repository.go)): Redis caching decorator
  - Implements decorator pattern wrapping CandleMySQL
  - Cache key format: `candles:{symbol}:{interval}:{outputsize}`
  - Automatic cache invalidation on UpsertBatch
  - Graceful degradation when Redis is unavailable

### Architectural Characteristics

1. **Clean Architecture**: Domain layer is independent of infrastructure
2. **Dependency Inversion**: Usecase defines CandleRepository interface; adapters implement it
3. **Decorator Pattern**: CachingCandleRepository transparently adds caching
4. **Interface Ownership**: Interfaces defined where they are consumed (Go best practice)
5. **Graceful Degradation**: System continues operating when Redis is unavailable

## Directory Structure

```
candles/
├── README.md                          # This file
├── domain/
│   └── entity/
│       └── candle.go                  # Candle entity (OHLCV data)
├── usecase/
│   ├── candles_usecase.go             # Query logic + CandleRepository interface
│   ├── candles_usecase_test.go        # Usecase tests
│   ├── ingest_usecase.go              # Batch ingestion + MarketRepository interface
│   └── ingest_usecase_test.go         # Ingest tests
├── adapters/
│   ├── candle_mysql.go                # MySQL repository implementation
│   └── candle_mysql_test.go           # Repository tests
└── transport/
    ├── handler/
    │   ├── candle_handler.go          # HTTP handler
    │   └── candle_handler_test.go     # Handler tests
    └── http/dto/
        └── candle_response.go         # Response DTO
```

## Testing

All tests in the candles feature follow a **table-driven testing pattern** for consistency and maintainability.

### Test Structure and Patterns

#### Common Patterns Across All Tests

1. **Table-Driven Tests**: All test functions use a `tests` slice with struct fields:
   - `name`: Test case description (e.g., `"success: all parameters specified"`, `"error: repository returns error"`)
   - `wantErr`: Boolean flag indicating if an error is expected
   - Additional fields specific to each test type

2. **Parallel Execution**: Repository and handler tests use `t.Parallel()`:
   ```go
   func TestCandleMySQL_Find(t *testing.T) {
       t.Parallel()
       // ...
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               // ...
           })
       }
   }
   ```

3. **Helper Functions**: Each test file includes helper functions:
   - Repository: `setupTestDB()`, `seedCandle()`
   - Handler: Uses `httptest.NewRecorder()` for HTTP testing

#### Usecase Tests ([usecase/candles_usecase_test.go](usecase/candles_usecase_test.go))

Uses **mock repositories** to test business logic in isolation.

**Test Case Structure:**
```go
tests := []struct {
    name               string
    inputSymbol        string
    inputInterval      string
    inputOutputsize    int
    mockFindFunc       func(...) ([]entity.Candle, error)
    expectedCandles    []entity.Candle
    expectedErr        error
    expectedInterval   string  // Verified value passed to mock
    expectedOutputsize int     // Verified value passed to mock
}{/* ... */}
```

**Key Features:**
- Mock implementations with customizable behavior
- Parameter validation testing (defaults, max limits)
- Call count verification

**Run Command:**
```bash
go test ./internal/feature/candles/usecase/... -v
```

#### Handler Tests ([transport/handler/candle_handler_test.go](transport/handler/candle_handler_test.go))

Uses **mock usecases** to test HTTP request/response handling.

**Test Case Structure:**
```go
tests := []struct {
    name           string
    url            string
    mockGetCandles func(...) ([]entity.Candle, error)
    expectedStatus int
    expectedBody   string  // JSON string comparison
}{/* ... */}
```

**Key Features:**
- HTTP status code verification
- JSON response body matching with `assert.JSONEq`
- Query parameter parsing validation
- Default value handling

**Run Command:**
```bash
go test ./internal/feature/candles/transport/handler/... -v
```

#### Repository Tests ([adapters/candle_mysql_test.go](adapters/candle_mysql_test.go))

Uses **in-memory SQLite database** for integration testing.

**Test Case Structure:**
```go
tests := []struct {
    name         string
    symbol       string
    interval     string
    outputsize   int
    wantErr      bool
    setupFunc    func(t *testing.T, db *gorm.DB)
    validateFunc func(t *testing.T, candles []entity.Candle)
}{/* ... */}
```

**Key Features:**
- Each test gets a fresh in-memory SQLite database
- `setupFunc`: Prepares test data before execution
- `validateFunc`: Custom validation logic for success cases
- Tests upsert behavior (insert vs update)
- Tests ordering and limit functionality

**Run Command:**
```bash
go test ./internal/feature/candles/adapters/... -v
```

### Run All Tests

```bash
go test ./internal/feature/candles/... -v -race -cover
```

### Example Test Output

```
=== RUN   TestCandlesUsecase_GetCandles
=== RUN   TestCandlesUsecase_GetCandles/success:_all_parameters_specified
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_interval_is_empty
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_outputsize_is_0
=== RUN   TestCandlesUsecase_GetCandles/success:_default_value_used_when_outputsize_exceeds_max
=== RUN   TestCandlesUsecase_GetCandles/error:_repository_returns_error
--- PASS: TestCandlesUsecase_GetCandles (0.00s)
    --- PASS: TestCandlesUsecase_GetCandles/success:_all_parameters_specified (0.00s)
    --- PASS: TestCandlesUsecase_GetCandles/success:_default_value_used_when_interval_is_empty (0.00s)
    ...
```

## Caching Strategy

### Cache Configuration

| Setting | Value | Description |
|---------|-------|-------------|
| Key Format | `candles:{symbol}:{interval}:{outputsize}` | Unique key per query |
| Default TTL | 5 minutes | Configurable via constructor |
| Namespace | `candles` | Key prefix for isolation |

### Cache Behavior

1. **Read Path (Find)**
   - Check Redis for cached data
   - On HIT: Return deserialized data
   - On MISS: Query MySQL, cache result, return data
   - On Redis error: Bypass cache, query MySQL directly

2. **Write Path (UpsertBatch)**
   - Write to MySQL first
   - Invalidate related cache entries using pattern matching
   - Pattern: `candles:{symbol}:{interval}:*`

### Graceful Degradation

The caching layer is designed to fail gracefully:
- If Redis is unavailable, requests are served directly from MySQL
- Cache write failures are logged but don't fail the request
- Corrupted cache entries are automatically deleted

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TWELVE_DATA_API_KEY` | API key for TwelveData market data | Yes (for ingestion) |

**Note:** Redis and MySQL connection settings are configured at the application level, not specific to this feature.

## Future Enhancements

- Real-time data streaming via WebSocket
- Additional technical indicators (SMA, EMA, RSI, etc.)
- Historical data backfill functionality
- Custom interval support (5min, 15min, 1hour)
- Data export in CSV/Excel formats
