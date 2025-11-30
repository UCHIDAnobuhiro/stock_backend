package jwtmw

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Generator defines the interface for JWT token generation.
type Generator interface {
	// GenerateToken creates a signed JWT token for the given user.
	GenerateToken(userID uint, email string) (string, error)
}

// generator implements the Generator interface.
type generator struct {
	secret     []byte
	expiration time.Duration
}

// NewGenerator creates a new JWT generator with the provided secret and expiration duration.
func NewGenerator(secret string, expiration time.Duration) Generator {
	return &generator{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// GenerateToken creates a signed JWT token with standard claims.
func (g *generator) GenerateToken(userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"exp":   time.Now().Add(g.expiration).Unix(),
		"iat":   time.Now().Unix(),
		"email": email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}
