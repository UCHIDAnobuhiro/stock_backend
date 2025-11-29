// Package usecase implements the business logic for the auth feature.
package usecase

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"stock_backend/internal/feature/auth/domain/entity"
	"stock_backend/internal/feature/auth/domain/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// jwtExpiration defines the validity period for issued JWT tokens.
	jwtExpiration = 1 * time.Hour
)

// AuthUsecase defines use cases for authentication operations.
type AuthUsecase interface {
	// Signup registers a new user with the given email and password.
	Signup(email, password string) error
	// Login authenticates a user and returns a JWT token on success.
	Login(email, password string) (string, error)
}

// authUsecase implements the AuthUsecase interface.
type authUsecase struct {
	users repository.UserRepository
}

// NewAuthUsecase creates a new AuthUsecase instance.
func NewAuthUsecase(users repository.UserRepository) AuthUsecase {
	return &authUsecase{users: users}
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
		log.Printf("[LOGIN] bcrypt NG: %v", err)
		return "", errors.New("invalid email or password")
	}
	log.Printf("[LOGIN] bcrypt OK for id=%d", user.ID)

	// Generate JWT token
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("server misconfigured: JWT_SECRET missing")
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"exp":   time.Now().Add(jwtExpiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": user.Email,
	}

	// Sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	log.Printf("[LOGIN] success id=%d", user.ID)
	return signed, nil
}
