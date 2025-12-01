// Package handler provides HTTP handlers for the auth feature.
package handler

import (
	"errors"
	"net/http"

	"stock_backend/internal/feature/auth/domain"
	"stock_backend/internal/feature/auth/transport/http/dto"

	"github.com/gin-gonic/gin"
)

// AuthUsecase defines use cases for authentication operations.
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type AuthUsecase interface {
	// Signup registers a new user with the given email and password.
	Signup(email, password string) error
	// Login authenticates a user and returns a JWT token on success.
	Login(email, password string) (string, error)
}

// AuthHandler handles HTTP requests for authentication operations.
// It depends on the AuthUsecase interface and handles JSON requests/responses.
type AuthHandler struct {
	auth AuthUsecase
}

// NewAuthHandler creates a new AuthHandler instance.
// This is a constructor for dependency injection, injecting AuthUsecase from outside.
func NewAuthHandler(auth AuthUsecase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Signup handles the user registration API endpoint.
// - Binds the request JSON to SignupReq
// - Returns 400 on validation errors
// - Returns 409 on duplicate email (domain.ErrUserAlreadyExists)
// - Returns 500 on internal server errors
// - Returns 201 on success
func (h *AuthHandler) Signup(c *gin.Context) {
	var req dto.SignupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.Signup(req.Email, req.Password); err != nil {
		// Map domain errors to appropriate HTTP status codes
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
			return
		}
		// For all other errors (e.g., database errors), return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "ok"})
}

// Login handles the user login API endpoint.
// - Binds the request JSON to LoginReq
// - Returns 400 on validation errors
// - Returns 401 on authentication failure (domain.ErrInvalidCredentials)
// - Returns 500 on internal server errors
// - Returns 200 with JWT token on successful authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		// Map domain errors to appropriate HTTP status codes
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		// For all other errors (e.g., JWT generation errors), return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
