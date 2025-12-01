package usecase

import (
	"errors"
	"stock_backend/internal/feature/auth/domain/entity"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a mock implementation of repository.UserRepository interface.
// It simulates database operations during testing.
type mockUserRepository struct {
	// CreateFunc is called when the Create method is invoked.
	CreateFunc func(user *entity.User) error
	// FindByEmailFunc is called when the FindByEmail method is invoked.
	FindByEmailFunc func(email string) (*entity.User, error)
	// FindByIDFunc is called when the FindByID method is invoked.
	FindByIDFunc func(id uint) (*entity.User, error)
}

// mockJWTGenerator is a mock implementation of JWTGenerator interface.
// It simulates JWT token generation during testing.
type mockJWTGenerator struct {
	// GenerateTokenFunc is called when the GenerateToken method is invoked.
	GenerateTokenFunc func(userID uint, email string) (string, error)
}

// GenerateToken is the mock implementation of the GenerateToken method.
func (m *mockJWTGenerator) GenerateToken(userID uint, email string) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(userID, email)
	}
	// Default: return a dummy token
	return "mock-jwt-token", nil
}

// Create is the mock implementation of the Create method.
func (m *mockUserRepository) Create(user *entity.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil // Default: success
}

// FindByEmail is the mock implementation of the FindByEmail method.
func (m *mockUserRepository) FindByEmail(email string) (*entity.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(email)
	}
	// Default: return user not found error
	return nil, errors.New("user not found")
}

// FindByID is the mock implementation of the FindByID method.
func (m *mockUserRepository) FindByID(id uint) (*entity.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	// Default: return user not found error
	return nil, errors.New("user not found")
}

// createTestUser creates a test user with hashed password for testing.
// This helper reduces code duplication and makes tests more maintainable.
func createTestUser(t *testing.T, id uint, email, password string) *entity.User {
	t.Helper()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return &entity.User{
		ID:       id,
		Email:    email,
		Password: string(hashedPassword),
	}
}

// assertError checks if an error matches expectations.
// This helper standardizes error assertions across all tests.
func assertError(t *testing.T, err error, wantErr bool, errMsg string) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		if errMsg != "" && err.Error() != errMsg {
			t.Errorf("expected error '%s', got: '%s'", errMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// verifyBcryptHash checks if a hashed password matches the plaintext password.
// This helper encapsulates the bcrypt verification logic.
func verifyBcryptHash(t *testing.T, hashedPassword, plainPassword string) {
	t.Helper()
	if len(hashedPassword) == 0 || hashedPassword == plainPassword {
		t.Error("password is not hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword)); err != nil {
		t.Errorf("invalid bcrypt hash: %v", err)
	}
}

func TestAuthUsecase_Signup(t *testing.T) {
	t.Parallel() // enable parallel execution for test function

	tests := []struct {
		name              string
		email             string
		password          string
		wantErr           bool
		errMsg            string
		verifyBcryptHash  bool
		repositoryErr     error
	}{
		{
			name:             "successful signup",
			email:            "test@example.com",
			password:         "password123",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:     "password too short",
			email:    "test@example.com",
			password: "short",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:             "password at minimum length",
			email:            "test@example.com",
			password:         "12345678",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:             "long password",
			email:            "test@example.com",
			password:         "this-is-a-very-long-password-with-many-characters",
			wantErr:          false,
			verifyBcryptHash: true,
		},
		{
			name:          "repository create failure",
			email:         "test@example.com",
			password:      "password123",
			wantErr:       true,
			repositoryErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // enable parallel execution for subtests

			mockRepo := &mockUserRepository{
				CreateFunc: func(user *entity.User) error {
					if tt.verifyBcryptHash {
						verifyBcryptHash(t, user.Password, tt.password)
					}
					if tt.repositoryErr != nil {
						return tt.repositoryErr
					}
					return nil
				},
			}
			mockJWT := &mockJWTGenerator{}

			uc := NewAuthUsecase(mockRepo, mockJWT)
			err := uc.Signup(tt.email, tt.password)

			// Use helper function for error assertions
			if tt.wantErr {
				assertError(t, err, true, tt.errMsg)
				if tt.repositoryErr != nil && !errors.Is(err, tt.repositoryErr) {
					t.Errorf("expected error '%v', got: %v", tt.repositoryErr, err)
				}
			} else {
				assertError(t, err, false, "")
			}
		})
	}
}

func TestAuthUsecase_Login(t *testing.T) {
	t.Parallel() // enable parallel execution for test function

	// Create test user using helper function
	testUser := createTestUser(t, 1, "test@example.com", "password123")

	tests := []struct {
		name              string
		email             string
		password          string
		wantErr           bool
		errMsg            string
		expectedToken     string
		findByEmailResult *entity.User
		findByEmailErr    error
		jwtGenerateErr    error
		verifyJWTParams   bool
	}{
		{
			name:              "successful login",
			email:             "test@example.com",
			password:          "password123",
			wantErr:           false,
			expectedToken:     "mock-jwt-token",
			findByEmailResult: testUser,
			verifyJWTParams:   true,
		},
		{
			name:           "user not found",
			email:          "wrong@example.com",
			password:       "password123",
			wantErr:        true,
			errMsg:         "invalid email or password",
			findByEmailErr: errors.New("user not found"),
		},
		{
			name:              "incorrect password",
			email:             "test@example.com",
			password:          "wrong-password",
			wantErr:           true,
			errMsg:            "invalid email or password",
			findByEmailResult: testUser,
		},
		{
			name:              "JWT generation failure",
			email:             "test@example.com",
			password:          "password123",
			wantErr:           true,
			errMsg:            "failed to generate token: failed to sign token",
			findByEmailResult: testUser,
			jwtGenerateErr:    errors.New("failed to sign token"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // enable parallel execution for subtests

			mockRepo := &mockUserRepository{
				FindByEmailFunc: func(email string) (*entity.User, error) {
					if tt.findByEmailErr != nil {
						return nil, tt.findByEmailErr
					}
					return tt.findByEmailResult, nil
				},
			}
			mockJWT := &mockJWTGenerator{
				GenerateTokenFunc: func(userID uint, email string) (string, error) {
					if tt.verifyJWTParams {
						if userID != testUser.ID || email != testUser.Email {
							t.Errorf("unexpected userID or email: got userID=%d, email=%s", userID, email)
						}
					}
					if tt.jwtGenerateErr != nil {
						return "", tt.jwtGenerateErr
					}
					return tt.expectedToken, nil
				},
			}

			uc := NewAuthUsecase(mockRepo, mockJWT)
			token, err := uc.Login(tt.email, tt.password)

			// Use helper function for error assertions
			assertError(t, err, tt.wantErr, tt.errMsg)

			// Additional success case validations
			if !tt.wantErr {
				if token == "" {
					t.Error("token is empty")
				}
				if tt.expectedToken != "" && token != tt.expectedToken {
					t.Errorf("expected token '%s', got: '%s'", tt.expectedToken, token)
				}
			}
		})
	}
}
