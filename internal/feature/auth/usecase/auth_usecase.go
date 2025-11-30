// Package usecase implements the business logic for the auth feature.
package usecase

import (
	"errors"
	"fmt"
	"log/slog"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/domain/repository"

	"golang.org/x/crypto/bcrypt"
)

// JWTGenerator defines the interface for generating JWT tokens.
type JWTGenerator interface {
	// GenerateToken creates a signed JWT token for the given user.
	GenerateToken(userID uint, email string) (string, error)
}

// AuthUsecase defines use cases for authentication operations.
type AuthUsecase interface {
	// Signup registers a new user with the given email and password.
	Signup(email, password string) error
	// Login authenticates a user and returns a JWT token on success.
	Login(email, password string) (string, error)
}

// authUsecase implements the AuthUsecase interface.
type authUsecase struct {
	users        repository.UserRepository
	jwtGenerator JWTGenerator
}

// NewAuthUsecase creates a new AuthUsecase instance.
func NewAuthUsecase(users repository.UserRepository, jwtGenerator JWTGenerator) AuthUsecase {
	return &authUsecase{
		users:        users,
		jwtGenerator: jwtGenerator,
	}
}

// Signup registers a new user with a hashed password.
func (u *authUsecase) Signup(email, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user := &entity.User{Email: email, Password: string(hashed)}
	return u.users.Create(user)
}

// Login authenticates a user and returns a JWT token on success.
// It verifies the email and password, then generates a signed JWT token.
func (u *authUsecase) Login(email, password string) (string, error) {
	// Find user by email
	user, err := u.users.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	// Verify password with bcrypt
	// First argument is the hashed password, second is the plaintext password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		slog.Debug("password verification failed", "user_id", user.ID, "error", err)
		return "", errors.New("invalid email or password")
	}
	slog.Debug("password verified", "user_id", user.ID)

	// Generate JWT token using injected generator
	token, err := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	slog.Info("login successful", "user_id", user.ID, "email", user.Email)
	return token, nil
}
