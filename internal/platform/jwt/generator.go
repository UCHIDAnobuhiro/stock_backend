package jwtmw

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Generator implements JWT token generation.
// It implements the JWTGenerator interface defined by consumers (e.g., auth/usecase).
type Generator struct {
	secret     []byte
	expiration time.Duration
}

// NewGenerator creates a new JWT generator with the provided secret and expiration duration.
func NewGenerator(secret string, expiration time.Duration) *Generator {
	return &Generator{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// GenerateToken creates a signed JWT token with standard claims.
func (g *Generator) GenerateToken(userID uint, email string) (string, error) {
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
