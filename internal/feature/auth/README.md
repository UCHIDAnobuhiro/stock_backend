# Auth Feature

## Overview

The Auth feature provides a JWT (JSON Web Token) based authentication system. It handles user registration, login, and JWT token issuance and verification.

### Key Features

- **User Signup**: Register new users with email and password
- **Login**: Authenticate credentials and issue JWT tokens
- **Password Encryption**: Secure password hashing with bcrypt
- **JWT Authentication**: Issue JWT tokens valid for 1 hour to control access to protected endpoints

## Sequence Diagrams

### User Signup Flow

```mermaid
sequenceDiagram
    participant Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant Repository as UserRepository
    participant DB as MySQL

    Client->>Handler: POST /signup<br/>{email, password}
    Handler->>Handler: Validate Request (email format, min 8 chars)

    alt Validation Failed
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: Signup(email, password)
    Usecase->>Usecase: Hash password with bcrypt
    Usecase->>Repository: Create(user)
    Repository->>DB: INSERT user

    alt Email Already Exists
        DB-->>Repository: Error (duplicate key)
        Repository-->>Usecase: Error
        Usecase-->>Handler: Error
        Handler-->>Client: 409 Conflict
    end

    DB-->>Repository: Success
    Repository-->>Usecase: Success
    Usecase-->>Handler: Success
    Handler-->>Client: 201 Created<br/>{message: "ok"}
```

### Login Flow

```mermaid
sequenceDiagram
    participant Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant Repository as UserRepository
    participant DB as MySQL

    Client->>Handler: POST /login<br/>{email, password}
    Handler->>Handler: Validate Request

    alt Validation Failed
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: Login(email, password)
    Usecase->>Repository: FindByEmail(email)
    Repository->>DB: SELECT * FROM users WHERE email = ?

    alt User Not Found
        DB-->>Repository: Not Found
        Repository-->>Usecase: Error
        Usecase-->>Handler: Error
        Handler-->>Client: 401 Unauthorized<br/>"invalid email or password"
    end

    DB-->>Repository: User Entity
    Repository-->>Usecase: User Entity
    Usecase->>Usecase: bcrypt.CompareHashAndPassword()

    alt Password Mismatch
        Usecase-->>Handler: Error
        Handler-->>Client: 401 Unauthorized<br/>"invalid email or password"
    end

    Usecase->>Usecase: Generate JWT Token<br/>(sub: userID, exp: 1h, email)
    Usecase->>Usecase: Sign with JWT_SECRET
    Usecase-->>Handler: JWT Token
    Handler-->>Client: 200 OK<br/>{token: "eyJhbGc..."}
```

## API Specification

### POST /signup

Registers a new user.

**Request**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Validation Rules**
- `email`: Required, valid email format
- `password`: Required, minimum 8 characters

**Response**

- **201 Created** - Registration successful
  ```json
  {
    "message": "ok"
  }
  ```

- **400 Bad Request** - Validation error
  ```json
  {
    "error": "Key: 'SignupReq.Password' Error:Field validation for 'Password' failed on the 'min' tag"
  }
  ```

- **409 Conflict** - Email address already in use
  ```json
  {
    "error": "failed to hash password: ..."
  }
  ```

### POST /login

Authenticates a user and issues a JWT token.

**Request**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Validation Rules**
- `email`: Required, valid email format
- `password`: Required

**Response**

- **200 OK** - Authentication successful
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
  ```

  **JWT Claims:**
  - `sub`: User ID (uint)
  - `email`: User's email address
  - `iat`: Issued at (Unix timestamp)
  - `exp`: Expiration time (issued at + 1 hour)

- **400 Bad Request** - Validation error
  ```json
  {
    "error": "Key: 'LoginReq.Email' Error:Field validation for 'Email' failed on the 'email' tag"
  }
  ```

- **401 Unauthorized** - Authentication failed (invalid email or password)
  ```json
  {
    "error": "invalid email or password"
  }
  ```

## Dependency Diagram

```mermaid
graph TB
    subgraph "Transport Layer"
        Handler[AuthHandler<br/>transport/handler]
        DTO[DTOs<br/>transport/http/dto]
    end

    subgraph "Usecase Layer"
        Usecase[AuthUsecase<br/>usecase]
    end

    subgraph "Domain Layer"
        Entity[User Entity<br/>domain/entity]
        RepoInterface[UserRepository Interface<br/>domain/repository]
    end

    subgraph "Adapters Layer"
        RepoImpl[UserMySQL<br/>adapters]
    end

    subgraph "External Dependencies"
        DB[(MySQL)]
        BCrypt[bcrypt<br/>Password Hashing]
        JWT[golang-jwt/jwt<br/>Token Generation]
    end

    Handler -->|depends on| Usecase
    Handler -->|uses| DTO
    Usecase -->|depends on| RepoInterface
    Usecase -->|uses| Entity
    Usecase -->|uses| BCrypt
    Usecase -->|uses| JWT
    RepoImpl -.->|implements| RepoInterface
    RepoImpl -->|uses| Entity
    RepoImpl -->|accesses| DB

    style Handler fill:#e1f5ff
    style Usecase fill:#fff4e1
    style Entity fill:#e8f5e9
    style RepoInterface fill:#e8f5e9
    style RepoImpl fill:#f3e5f5
    style DB fill:#ffebee
```

### Dependency Explanation

#### Transport Layer ([transport/handler/auth_handler.go](transport/handler/auth_handler.go))
- **AuthHandler**: Processes HTTP requests and calls AuthUsecase
- **DTOs** ([transport/http/dto/](transport/http/dto/)): Define request/response data structures
  - [SignupReq](transport/http/dto/signup_request.go): User registration request
  - [LoginReq](transport/http/dto/login_request.go): Login request

#### Usecase Layer ([usecase/auth_usecase.go](usecase/auth_usecase.go))
- **AuthUsecase**: Implements authentication business logic
  - Password hashing (bcrypt)
  - Password verification
  - JWT token generation and signing

#### Domain Layer
- **User Entity** ([domain/entity/user.go](domain/entity/user.go)): User domain model
- **UserRepository Interface** ([domain/repository/user_repository.go](domain/repository/user_repository.go)): Abstract repository interface
  - `Create(user)`: Create user
  - `FindByEmail(email)`: Find user by email address
  - `FindByID(id)`: Find user by ID

#### Adapters Layer ([adapters/user_mysql.go](adapters/user_mysql.go))
- **UserMySQL**: MySQL implementation of UserRepository (using GORM)

### Architectural Characteristics

1. **Clean Architecture**: Domain layer is independent of infrastructure layer
2. **Dependency Inversion**: Usecase depends on Repository interface, not concrete implementations
3. **Security**:
   - Passwords are hashed with bcrypt before storage
   - JWT tokens are signed with HS256 algorithm
   - Signing uses `JWT_SECRET` environment variable

## Directory Structure

```
auth/
├── README.md                          # This file
├── domain/
│   ├── entity/
│   │   └── user.go                   # User entity definition
│   └── repository/
│       └── user_repository.go        # UserRepository interface
├── usecase/
│   ├── auth_usecase.go               # Authentication business logic
│   └── auth_usecase_test.go          # Usecase tests
├── adapters/
│   ├── user_mysql.go                 # MySQL repository implementation
│   └── user_mysql_test.go            # Repository tests
└── transport/
    ├── handler/
    │   ├── auth_handler.go           # HTTP handlers
    │   └── auth_handler_test.go      # Handler tests
    └── http/dto/
        ├── signup_request.go         # Signup request DTO
        └── login_request.go          # Login request DTO
```

## Testing

All tests in the auth feature follow a **table-driven testing pattern** for consistency and maintainability.

### Test Structure and Patterns

#### Common Patterns Across All Tests

1. **Table-Driven Tests**: All test functions use a `tests` slice with struct fields:
   - `name`: Test case description (e.g., `"success: user creation"`, `"failure: duplicate email"`)
   - `wantErr`: Boolean flag indicating if an error is expected
   - Additional fields specific to each test type (see below)

2. **Parallel Execution**: All tests use `t.Parallel()` to enable concurrent execution:
   ```go
   func TestSomething(t *testing.T) {
       t.Parallel()  // Enable parallel execution

       tests := []struct { /* ... */ }{/* ... */}

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()  // Enable parallel subtests
               // Test logic...
           })
       }
   }
   ```

3. **Helper Functions**: Each test file includes helper functions to reduce code duplication:
   - Usecase: `createTestUser()`, `assertError()`, `verifyBcryptHash()`
   - Handler: `makeRequest()`, `assertJSONResponse()`
   - Repository: `setupTestDB()`, `seedUser()`

#### Usecase Tests ([usecase/auth_usecase_test.go](usecase/auth_usecase_test.go))

Uses **mock repositories** to test business logic in isolation.

**Test Case Structure:**
```go
tests := []struct {
    name              string
    email             string
    password          string
    wantErr           bool
    errMsg            string           // Expected error message
    verifyBcryptHash  bool             // Should verify password hashing
    repositoryErr     error            // Mock repository error
}{/* ... */}
```

**Key Features:**
- Mock implementations with customizable behavior via function fields
- bcrypt password verification
- JWT token generation validation

**Run Command:**
```bash
go test ./internal/feature/auth/usecase/... -v
```

#### Handler Tests ([transport/handler/auth_handler_test.go](transport/handler/auth_handler_test.go))

Uses **mock usecases** to test HTTP request/response handling.

**Test Case Structure:**
```go
tests := []struct {
    name           string
    requestBody    gin.H
    mockSignupFunc func(ctx context.Context, email, password string) error
    expectedStatus int
    expectedBody   gin.H
}{/* ... */}
```

**Key Features:**
- HTTP request/response validation
- DTO validation testing
- Status code verification
- JSON response body matching

**Run Command:**
```bash
go test ./internal/feature/auth/transport/handler/... -v
```

#### Repository Tests ([adapters/user_mysql_test.go](adapters/user_mysql_test.go))

Uses **in-memory SQLite database** for integration testing.

**Test Case Structure:**
```go
tests := []struct {
    name         string
    email        string          // (or user, userID depending on the test)
    wantErr      bool
    expectedErr  error           // Specific error type (e.g., usecase.ErrUserNotFound)
    setupFunc    func(t *testing.T, db *gorm.DB) *entity.User  // Setup test data
    validateFunc func(t *testing.T, expected, found *entity.User)  // Validate results
}{/* ... */}
```

**Key Features:**
- Each test gets a fresh in-memory SQLite database
- `setupFunc`: Prepares test data before execution
- `validateFunc`: Custom validation logic for success cases
- Tests database constraints (unique email, timestamps, etc.)

**Run Command:**
```bash
go test ./internal/feature/auth/adapters/... -v
```

### Run All Tests

```bash
go test ./internal/feature/auth/... -v -race -cover
```

### Example Test Output

```
=== RUN   TestAuthUsecase_Signup
=== PAUSE TestAuthUsecase_Signup
=== CONT  TestAuthUsecase_Signup
=== RUN   TestAuthUsecase_Signup/success:_user_creation
=== PAUSE TestAuthUsecase_Signup/success:_user_creation
=== RUN   TestAuthUsecase_Signup/failure:_password_too_short
=== PAUSE TestAuthUsecase_Signup/failure:_password_too_short
...
--- PASS: TestAuthUsecase_Signup (0.01s)
    --- PASS: TestAuthUsecase_Signup/success:_user_creation (0.00s)
    --- PASS: TestAuthUsecase_Signup/failure:_password_too_short (0.00s)
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `JWT_SECRET` | Secret key for signing JWT tokens | ✅ |

**Configuration Example** (`.env.docker`):
```
JWT_SECRET=your-super-secret-key-change-this-in-production
```

## Security Considerations

1. **Password Hashing**: Uses bcrypt (default cost: 10)
2. **JWT Expiration**: Automatically expires after 1 hour
3. **Error Messages**: Login failures return unified "invalid email or password" message (prevents enumeration attacks)
4. **JWT_SECRET**: Managed via environment variable; use a strong secret key in production

## Future Enhancements

- Refresh token implementation
- Password reset functionality
- Email verification
- Two-factor authentication (2FA)
- OAuth2 provider integration
