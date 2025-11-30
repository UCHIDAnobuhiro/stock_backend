# 📈 Stock View API (Go / Gin / Clean Architecture)

## 🧭 Overview

**Backend API for stock data delivery and authentication**
Built with Go and the Gin framework, it integrates with the frontend (Kotlin / Jetpack Compose).
As a REST API, it provides user authentication, stock data delivery, and cache optimization.

## ⚙️ Key Features

- **User Authentication**

  - Email/Password login
  - JWT issuance (short-lived access tokens + planned refresh token implementation)
  - Authorization via token verification middleware

- **Stock Data Retrieval**

  - Fetches stock data from external APIs (e.g., Twelve Data)
  - Returns candlestick data for daily, weekly, and monthly intervals
  - Caches recent data using Redis

- **Cache Optimization**

  - Redis caching for candlestick and symbol data
  - TTL configuration and automatic refresh
  - On cache miss: API call + DB storage

- **Database Persistence**
  - Data persistence via MySQL / Cloud SQL
  - ORM management using GORM

---

## 🛠️ Tech Stack

| Category      | Technology                                                          |
| ------------- | ------------------------------------------------------------------- |
| Language      | Go (1.24)                                                           |
| Web Framework | Gin                                                                 |
| ORM           | GORM                                                                |
| DB            | MySQL / Cloud SQL                                                   |
| Cache         | Redis                                                               |
| Auth          | JWT / bcrypt                                                        |
| Config        | **.env.docker (local) / Secret Manager (production) + os.Getenv()** |
| Container     | Docker / Docker Compose                                             |
| Cloud         | Google Cloud Run / Cloud SQL / Secret Manager / Artifact Registry   |
| CI/CD         | GitHub Actions                                                      |

## 📂 Directory Structure

```text
.
├── cmd/
│   ├── ingest/                 # Data fetching and ingestion (scheduled jobs)
│   └── server/                 # Main entry point (main.go)
│
├── internal/
│   ├── app/                    # Application foundation
│   │   ├── di/                 # Dependency Injection
│   │   └── router/             # Routing configuration
│   │
│   ├── feature/                # Feature modules (vertical slices)
│   │   ├── auth/               # Authentication feature
│   │   │   ├── domain/         # Domain layer
│   │   │   │   └── entity/     # Entities (User)
│   │   │   ├── usecase/        # Use cases (defines repository interfaces, business logic)
│   │   │   ├── adapters/       # Adapters (repository implementations)
│   │   │   └── transport/      # Transport layer
│   │   │       ├── handler/    # HTTP handlers
│   │   │       └── http/dto/   # Request/Response DTOs
│   │   │
│   │   ├── candles/            # Candlestick data feature
│   │   │   ├── domain/
│   │   │   │   └── entity/     # Entities (Candle)
│   │   │   ├── usecase/        # Use cases (defines repository interfaces, fetch/store logic)
│   │   │   ├── adapters/       # MySQL implementation
│   │   │   └── transport/
│   │   │       ├── handler/    # HTTP handlers
│   │   │       └── http/dto/   # Request/Response DTOs
│   │   │
│   │   └── symbollist/         # Symbol list feature
│   │       ├── domain/
│   │       │   └── entity/     # Entities (Symbol)
│   │       ├── usecase/        # Use cases (defines repository interfaces)
│   │       ├── adapters/       # Repository implementations
│   │       └── transport/
│   │           ├── handler/    # HTTP handlers
│   │           └── http/dto/   # Request/Response DTOs
│   │
│   ├── platform/               # Infrastructure layer (external dependencies)
│   │   ├── cache/              # Redis caching decorator
│   │   ├── db/                 # Database connection initialization
│   │   ├── externalapi/        # External API clients
│   │   │   └── twelvedata/     # Twelve Data API implementation
│   │   ├── http/               # HTTP client configuration
│   │   ├── jwt/                # JWT generation/verification/middleware
│   │   └── redis/              # Redis client implementation
│   │
│   └── shared/                 # Shared utilities
│       └── ratelimiter/        # Rate limiting
│
├── docker/                     # Docker-related files
│   ├── Dockerfile.ingest       # Dockerfile for ingest (production)
│   ├── Dockerfile.server       # Dockerfile for API server (production)
│   ├── Dockerfile.server.dev   # Dockerfile for API server (local development)
│   ├── docker-compose.yml      # Common Docker configuration (service definitions, network setup)
│   ├── docker-compose.dev.yml  # Local development override configuration
│   └── mysql/                  # MySQL initialization scripts
│
├── .env.docker                 # Local environment variables (recommended for .gitignore)
├── go.mod
├── go.sum
└── .github/
    └── workflows/              # CI/CD (test, build, deploy)
```

## 🔒 Authentication Design (JWT + Refresh Token)

### Current Implementation

- JWT access token authentication
- Verification via `Authorization: Bearer <token>` header

### Future Plans (Hybrid Authentication)

- Implement **short-lived JWT (5-10 minutes)** + **server-managed refresh token** approach
- Automatic access token renewal via `/auth/refresh`
- Immediate revocation per device via `/auth/logout`
- Refresh tokens stored in DB or Redis with **rotation management**

## 💾 Data Flow (Example: Stock Price Retrieval)

1. Batch process (`cmd/ingest`) fetches stock data from external API (e.g., Twelve Data)
2. Stores fetched candlestick data in MySQL (or Cloud SQL)
3. Frontend requests `/api/v1/candles?symbol=AAPL&interval=1day`
4. Handler calls `CandlesUsecase`
5. Usecase checks **Redis cache** via Repository
   - **Cache hit**: Returns immediately from Redis
   - **Cache miss**: Fetches from MySQL → Caches in Redis → Returns response
6. Returns result as JSON to frontend

## 📚 API Endpoints

### 🩺 Health Check

| Method | Path       | Auth         | Description                           |
| ------ | ---------- | ------------ | ------------------------------------- |
| GET    | `/healthz` | Not required | Service health check (returns 200 OK) |

---

### 🔐 Authentication

| Method | Path      | Auth         | Description                     |
| ------ | --------- | ------------ | ------------------------------- |
| POST   | `/signup` | Not required | New user registration           |
| POST   | `/login`  | Not required | Login (issues JWT access token) |

---

### 💹 Stock Data (Candles / Symbols)

| Method | Path             | Auth     | Description                                            |
| ------ | ---------------- | -------- | ------------------------------------------------------ |
| GET    | `/symbols`       | Required | Fetch symbol list                                      |
| GET    | `/candles/:code` | Required | Fetch candlestick data for specified code (e.g., AAPL) |

### 💡 Notes

- `/candles` and `/symbols` require **JWT authentication (`Authorization: Bearer <token>`)**.
- Plans to add `/auth/refresh` and `/auth/logout` for refresh token support.

## ☁️ Cloud Architecture (Google Cloud)

- **Cloud Run**: Deploys Docker images
- **Cloud SQL (MySQL)**: Application data persistence
- **Redis (Cloud Memorystore)**: Cache management
- **Secret Manager**: Securely manages API keys, DB passwords, and JWT secret keys
- Loads at startup via `os.Getenv()` + Secret Manager API
- **Local development reads from `.env.docker`**

## 🧪 CI/CD

- **GitHub Actions** runs automated tests on pull request creation
- After merge, **Cloud Build** builds Docker images and stores them in **Artifact Registry**
- Uses **Workload Identity Federation** for secure deployment from GitHub to GCP
- Automatically deploys to **Cloud Run** and injects environment variables via Secret Manager

## ⚙️ Setup

### Prerequisites

- Docker / Docker Compose installed
- Go is not required (everything runs in Docker)
- Configure local environment variables in `.env.docker`

---

### Steps

```bash
# Clone repository
git clone https://github.com/UCHIDAnobuhiro/stock_backend.git
cd stock_backend

# Copy environment variables
cp example.env.docker .env.docker
```

### 🔑 Obtaining Twelve Data API Key

This application uses the [Twelve Data API](https://twelvedata.com/).
A free API key is required to fetch stock data.

1. Create an account on the Twelve Data website
2. Issue a key from "Dashboard > API Keys"
3. Copy and set it in .env.docker as TWELVE_DATA_API_KEY
   Example: `TWELVE_DATA_API_KEY=your_api_key_here`

### ⚠️ Twelve Data Free Plan Limitations

- Free plan allows **up to 8 requests per minute**

To address this limitation, this application:

- **Pre-fetches data via scheduled batch (ingest) processes**
- **Minimizes requests through Redis caching**

### 🧩 Starting the API Server

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock up backend-dev
```

### 🧠 Starting Batch Process (Stock Data Ingestion)

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock run --rm --no-deps ingest
```

### 💡 Notes

- **API Server**: <http://localhost:8080>
- **MySQL**: `localhost:3306`
- **Redis**: `localhost:6379`
- **View logs**: docker logs -f stock-backend-dev
- **Batch process**: ingest container fetches stock prices from external API and stores them in MySQL
