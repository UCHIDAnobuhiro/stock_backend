package jwtmw

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const ContextUserID = "userID"

// AuthRequired returns a Gin middleware function that validates JWT tokens
// and restricts access to authenticated users only.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get Authorization header
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		// 2. Load secret key from environment variable
		secret := os.Getenv(EnvKeyJWTSecret)
		if secret == "" {
			// Server misconfiguration (JWT_SECRET not set)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}

		// 3. Parse and verify JWT signature
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// Check signing algorithm (only HMAC allowed)
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			// Validation error or invalid token
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 4. Extract claims (payload)
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(float64); ok { // JWT numbers are decoded as float64
				c.Set(ContextUserID, uint(sub))
			}
		}
		// 5. Pass control to the next handler
		c.Next()
	}
}
