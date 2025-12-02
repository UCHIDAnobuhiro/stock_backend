// Package usecase implements the business logic for the auth feature.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"stock_backend/internal/feature/auth/domain/entity"

	"golang.org/x/crypto/bcrypt"
)

const (
	// minPasswordLength defines the minimum acceptable password length.
	minPasswordLength = 8
)

// UserRepository abstracts the persistence layer for user entities.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (adapters).
type UserRepository interface {
	// Create persists a new user to the storage.
	// It returns an error if a user with the same email already exists.
	Create(ctx context.Context, user *entity.User) error

	// FindByEmail retrieves a user matching the specified email address.
	// It returns an error if the user does not exist.
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByID retrieves a user matching the specified ID.
	// It returns an error if the user does not exist.
	FindByID(ctx context.Context, id uint) (*entity.User, error)
}

// JWTGenerator defines the interface for generating JWT tokens.
// Following Go convention: interfaces are defined by the consumer (usecase), not the provider (platform/jwt).
type JWTGenerator interface {
	// GenerateToken creates a signed JWT token for the given user.
	GenerateToken(userID uint, email string) (string, error)
}

// authUsecase implements authentication business logic.
type authUsecase struct {
	users        UserRepository
	jwtGenerator JWTGenerator
}

// NewAuthUsecase creates a new authUsecase instance.
func NewAuthUsecase(users UserRepository, jwtGenerator JWTGenerator) *authUsecase {
	return &authUsecase{
		users:        users,
		jwtGenerator: jwtGenerator,
	}
}

// validatePassword checks if the password meets security requirements.
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	}
	return nil
}

// Signup registers a new user with a hashed password.
func (u *authUsecase) Signup(ctx context.Context, email, password string) error {
	// Validate password strength
	if err := validatePassword(password); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user := &entity.User{Email: email, Password: string(hashed)}
	return u.users.Create(ctx, user)
}

// Login authenticates a user and returns a JWT token on success.
// It verifies the email and password, then generates a signed JWT token.
// To prevent timing attacks, bcrypt comparison is performed even when user doesn't exist.
func (u *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	// Find user by email
	user, err := u.users.FindByEmail(ctx, email)

	// Use a dummy hash for timing attack mitigation when user doesn't exist
	// This ensures bcrypt.CompareHashAndPassword is always called
	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy" // dummy hash
	if err == nil {
		passwordHash = user.Password
	}

	// Always verify password to prevent timing attacks
	// First argument is the hashed password, second is the plaintext password
	compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

	// If user not found or password incorrect, return generic error
	if err != nil || compareErr != nil {
		slog.Debug("login failed", "email", email)
		return "", errors.New("invalid email or password")
	}

	// Generate JWT token using injected generator
	token, tokenErr := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if tokenErr != nil {
		return "", fmt.Errorf("failed to generate token: %w", tokenErr)
	}

	slog.Info("login successful", "user_id", user.ID)
	return token, nil
}
