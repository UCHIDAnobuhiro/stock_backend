// Package usecase implements the business logic for the auth feature.
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"stock_backend/internal/feature/auth/domain/entity"

	"golang.org/x/crypto/bcrypt"
)

const (
	// minPasswordLength defines the minimum acceptable password length.
	minPasswordLength = 8

	// refreshTokenDuration is the lifetime of a refresh token.
	refreshTokenDuration = 7 * 24 * time.Hour // 7 days

	// maxSessionsPerUser is the maximum number of concurrent sessions per user.
	maxSessionsPerUser = 5

	// accessTokenExpiresIn is the access token expiration time in seconds.
	accessTokenExpiresIn = 900 // 15 minutes
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

// LoginResult contains the result of a successful login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
}

// RefreshResult contains the result of a successful token refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// authUsecase implements authentication business logic.
type authUsecase struct {
	users        UserRepository
	sessions     SessionRepository
	jwtGenerator JWTGenerator
}

// NewAuthUsecase creates a new authUsecase instance.
func NewAuthUsecase(users UserRepository, sessions SessionRepository, jwtGenerator JWTGenerator) *authUsecase {
	return &authUsecase{
		users:        users,
		sessions:     sessions,
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

// generateRefreshToken generates a cryptographically secure refresh token.
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
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

// Login authenticates a user and returns tokens on success.
// It verifies the email and password, creates a session, and generates tokens.
// To prevent timing attacks, bcrypt comparison is performed even when user doesn't exist.
func (u *authUsecase) Login(ctx context.Context, email, password, userAgent, ipAddress string) (*LoginResult, error) {
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
		return nil, errors.New("invalid email or password")
	}

	// Check session count and delete oldest if needed
	count, err := u.sessions.CountByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}
	if count >= maxSessionsPerUser {
		if err := u.sessions.DeleteOldestByUserID(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("failed to delete oldest session: %w", err)
		}
	}

	// Generate refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Create session
	now := time.Now()
	session := &entity.Session{
		ID:        refreshToken,
		UserID:    user.ID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		CreatedAt: now,
		ExpiresAt: now.Add(refreshTokenDuration),
	}
	if err := u.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT token using injected generator
	accessToken, tokenErr := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if tokenErr != nil {
		return nil, fmt.Errorf("failed to generate token: %w", tokenErr)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    accessTokenExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken validates a refresh token and issues new tokens.
// Implements refresh token rotation: old token is revoked, new token is issued.
func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken, userAgent, ipAddress string) (*RefreshResult, error) {
	// Find session by refresh token
	session, err := u.sessions.FindByID(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Check if session is valid
	if session.IsRevoked() {
		return nil, ErrSessionRevoked
	}
	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	// Get user
	user, err := u.users.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Revoke old session (rotation)
	if err := u.sessions.Revoke(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to revoke session: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Create new session
	now := time.Now()
	newSession := &entity.Session{
		ID:        newRefreshToken,
		UserID:    user.ID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		CreatedAt: now,
		ExpiresAt: now.Add(refreshTokenDuration),
	}
	if err := u.sessions.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate new access token
	accessToken, err := u.jwtGenerator.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    accessTokenExpiresIn,
	}, nil
}

// Logout revokes a refresh token.
// Always returns nil to prevent token enumeration attacks.
func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	// Attempt to revoke, but ignore errors to prevent enumeration
	_ = u.sessions.Revoke(ctx, refreshToken)
	return nil
}

// LogoutAll revokes all sessions for a user.
func (u *authUsecase) LogoutAll(ctx context.Context, userID uint) error {
	return u.sessions.RevokeAllByUserID(ctx, userID)
}
