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

func TestAuthUsecase_Signup(t *testing.T) {
	t.Run("successful signup", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			CreateFunc: func(user *entity.User) error {
				// Verify that the password is hashed
				if len(user.Password) == 0 || user.Password == "password123" {
					t.Errorf("password is not hashed")
				}
				// Verify that it's a valid bcrypt hash
				if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")); err != nil {
					t.Errorf("invalid bcrypt hash: %v", err)
				}
				return nil
			},
		}
		mockJWT := &mockJWTGenerator{}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		err := uc.Signup("test@example.com", "password123")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("repository create failure", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo := &mockUserRepository{
			CreateFunc: func(user *entity.User) error {
				return expectedErr
			},
		}
		mockJWT := &mockJWTGenerator{}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		err := uc.Signup("test@example.com", "password123")

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error '%v', got: %v", expectedErr, err)
		}
	})
}

func TestAuthUsecase_Login(t *testing.T) {
	// Hashed password for testing
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	testUser := &entity.User{
		ID:       1,
		Email:    "test@example.com",
		Password: string(hashedPassword),
	}

	t.Run("successful login", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				if email == testUser.Email {
					return testUser, nil
				}
				return nil, errors.New("user not found")
			},
		}
		mockJWT := &mockJWTGenerator{
			GenerateTokenFunc: func(userID uint, email string) (string, error) {
				if userID != testUser.ID || email != testUser.Email {
					t.Errorf("unexpected userID or email: got userID=%d, email=%s", userID, email)
				}
				return "mock-jwt-token", nil
			},
		}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		token, err := uc.Login("test@example.com", "password123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if token == "" {
			t.Error("token is empty")
		}

		if token != "mock-jwt-token" {
			t.Errorf("expected token 'mock-jwt-token', got: '%s'", token)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return nil, errors.New("user not found")
			},
		}
		mockJWT := &mockJWTGenerator{}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		_, err := uc.Login("wrong@example.com", "password123")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		expectedErrMsg := "invalid email or password"
		if err.Error() != expectedErrMsg {
			t.Errorf("expected error message '%s', got: '%s'", expectedErrMsg, err.Error())
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return testUser, nil
			},
		}
		mockJWT := &mockJWTGenerator{}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		_, err := uc.Login("test@example.com", "wrong-password")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		expectedErrMsg := "invalid email or password"
		if err.Error() != expectedErrMsg {
			t.Errorf("expected error message '%s', got: '%s'", expectedErrMsg, err.Error())
		}
	})

	t.Run("JWT generation failure", func(t *testing.T) {
		mockRepo := &mockUserRepository{
			FindByEmailFunc: func(email string) (*entity.User, error) {
				return testUser, nil
			},
		}
		mockJWT := &mockJWTGenerator{
			GenerateTokenFunc: func(userID uint, email string) (string, error) {
				return "", errors.New("failed to sign token")
			},
		}

		uc := NewAuthUsecase(mockRepo, mockJWT)
		_, err := uc.Login("test@example.com", "password123")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		expectedErrMsg := "failed to generate token: failed to sign token"
		if err.Error() != expectedErrMsg {
			t.Errorf("expected error message '%s', got: '%s'", expectedErrMsg, err.Error())
		}
	})
}
